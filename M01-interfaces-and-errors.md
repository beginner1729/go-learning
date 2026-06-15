# Module 1 — Interfaces, Composition & Idiomatic Error Handling

> **Capstone contribution:** this module defines Pulse's **domain model**
> (`Customer`, `Campaign`, `Journey`, value objects) and its **error taxonomy**
> (sentinel + typed errors, wrapping conventions) — the vocabulary every later
> module speaks.

---

## 0. Setup & run

The complete, runnable code for this module lives in `m01-interfaces-and-errors/`.

```bash
cd go-cxm-course/m01-interfaces-and-errors
go mod tidy        # no external deps in M1 — stdlib only
go vet ./...
go test ./...      # customer + errorx packages
go run ./cmd/demo  # see validation, the LoggingRepo decorator, and errorx in action
```

Layout:

```
m01-interfaces-and-errors/
  go.mod                 # module cxm/m01
  customer/              # domain: value types, Customer, repo iface, InMemoryRepo, validation
  errorx/                # codes + HTTP mapping (the "implement it yourself" reference solution)
  cmd/demo/              # runnable demo wiring it together
```

---

## 1. Learning objectives

By the end you will be able to:

- Design small, behavior-focused **interfaces** and explain why Go favors them
  over inheritance.
- Use **struct embedding** for composition (and know when it's the wrong tool).
- Distinguish and choose between **sentinel errors**, **typed errors**, and
  **opaque/wrapped errors**.
- Wrap errors with `%w`, and inspect them with `errors.Is` and `errors.As`.
- Avoid the classic interface/error pitfalls: the typed-nil trap, premature
  interfaces, leaking implementation details, and stringly-typed error checks.

---

## 2. Concepts

### 2.1 Interfaces describe behavior, not data

In Go an interface is a set of method signatures. A type satisfies an interface
**implicitly** — no `implements` keyword. This is the single most important
design lever in Go.

```go
// A small interface: it says nothing about *how* we store customers,
// only *what* we can do. Consumers depend on this, not on Postgres or Mongo.
type CustomerRepository interface {
    Create(ctx context.Context, c Customer) (Customer, error)
    ByID(ctx context.Context, id CustomerID) (Customer, error)
}
```

**The idiom: "accept interfaces, return structs."** A function should accept the
*smallest* interface it needs, and usually return a concrete type. This keeps
callers flexible and implementations honest.

**The other idiom: define interfaces where they're *used*, not where they're
*implemented*.** In Go you do **not** ship an interface next to its
implementation "just in case." The Postgres package returns a concrete
`*PostgresCustomerRepo`. The *service* package that consumes it declares the
`CustomerRepository` interface it needs. This is the opposite of Java and it's
deliberate: the consumer owns the contract, so interfaces stay minimal and you
never get a 20-method interface with one implementation.

> **Rule of thumb:** the bigger the interface, the weaker the abstraction.
> `io.Reader` (one method) is reused everywhere. A 15-method interface is reused
> nowhere.

#### Pitfall: premature interfaces

Don't create an interface until you have (or can clearly foresee) **a second
implementation or a test seam**. A single implementation behind an interface is
just indirection. Start concrete; extract an interface when a consumer needs to
vary the implementation (real DB vs in-memory fake, for instance).

#### Pitfall: the typed-nil interface trap

This bites everyone once:

```go
func mightFail() *MyError { return nil } // returns a nil *MyError

func do() error {
    return mightFail() // BUG: error is NOT nil!
}

func main() {
    if err := do(); err != nil {
        fmt.Println("oops, this prints") // it prints!
    }
}
```

An interface value is `(type, value)`. `do()` returns an `error` whose *type* is
`*MyError` and whose *value* is `nil`. The interface itself is non-nil because
its type word is set. **Fix:** return the `error` type directly, or explicitly
`return nil`:

```go
func do() error {
    var err *MyError = mightFail()
    if err != nil {
        return err
    }
    return nil // explicit untyped nil
}
```

The general rule: **don't return concrete error types from functions whose
signature is `error`.** Have functions return `error`.

### 2.2 Composition via embedding

Go has no inheritance. It has **embedding**: put a type inside a struct without a
field name, and its methods/fields are promoted.

```go
type AuditFields struct {
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Customer struct {
    ID    CustomerID
    Email Email
    AuditFields // embedded — Customer now "has" CreatedAt/UpdatedAt
}

c := Customer{}
c.CreatedAt = time.Now() // promoted field
```

You can also embed interfaces to compose behavior, or embed an implementation to
decorate it:

```go
// Decorator: wrap a repo to add logging without changing the interface.
type LoggingRepo struct {
    CustomerRepository // embedded interface — promotes all its methods
    log *slog.Logger
}

func (r LoggingRepo) Create(ctx context.Context, c Customer) (Customer, error) {
    r.log.Info("creating customer", "email", c.Email)
    return r.CustomerRepository.Create(ctx, c) // delegate to wrapped impl
}
// ByID is automatically promoted — we only override what we want.
```

#### Pitfall: embedding for code reuse when you mean "has-a"

Embedding expresses "is-a-kind-of / shares the surface of." If you only want to
*use* another type's behavior internally, prefer a **named field** (composition
by reference). Embedding leaks the embedded type's methods into your public API,
which you may not want.

### 2.3 Errors are values

`error` is just an interface:

```go
type error interface {
    Error() string
}
```

Go's philosophy: errors are ordinary values you handle explicitly. There are
three patterns, and choosing well is a senior skill.

#### (a) Sentinel errors — a known, comparable singleton

```go
package customer

import "errors"

var ErrNotFound = errors.New("customer: not found")
```

Callers check identity with `errors.Is`:

```go
c, err := repo.ByID(ctx, id)
if errors.Is(err, customer.ErrNotFound) {
    // 404
}
```

Use sentinels for **expected, branchable conditions** that carry no extra data
(`io.EOF`, `sql.ErrNoRows`, `ErrNotFound`). Cost: they're part of your public
API forever, and they create package coupling.

#### (b) Typed errors — carry structured data

When the caller needs *details*, define a type:

```go
type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Msg)
}
```

Callers extract it with `errors.As`:

```go
var ve *ValidationError
if errors.As(err, &ve) {
    // use ve.Field, ve.Msg to build a structured 400 response
}
```

#### (c) Opaque errors — wrap and propagate

Most errors are neither branched on nor inspected; you just add context and pass
them up. Wrap with `%w` so the chain stays inspectable:

```go
func (s *Service) Onboard(ctx context.Context, email string) error {
    c, err := s.repo.Create(ctx, Customer{Email: Email(email)})
    if err != nil {
        // %w keeps the wrapped error reachable by errors.Is/As
        return fmt.Errorf("onboard %q: %w", email, err)
    }
    _ = c
    return nil
}
```

**`%w` vs `%v`:** use `%w` when callers might want to inspect the cause; use `%v`
when you intentionally want to *hide* the underlying error (e.g. don't leak a DB
error type across a package boundary). Wrapping is a deliberate API decision.

#### Adding context — the rules

- Add context that the caller doesn't already have. Good: `"onboard %q: %w"`.
  Bad: `"failed to: %w"` (the word "failed" is noise; every error is a failure).
- **Don't** capitalize error strings or end with punctuation (`go vet` flags it).
  They get concatenated: `"onboard: create: not found"`.
- Wrap **once per layer**, not at every call. Wrap when you cross a meaningful
  boundary and can add a name/identifier.

#### Pitfall: stringly-typed error checks

Never do `if strings.Contains(err.Error(), "not found")`. It's brittle and breaks
the moment a message changes. Use `errors.Is`/`errors.As`.

#### Pitfall: handling an error twice

Either **handle** an error (log it, recover, return a response) **or** return it —
not both. Logging an error and then returning it produces duplicate log lines at
every layer. Log at the boundary (the HTTP handler / top of a goroutine); return
everywhere below.

#### `errors.Join` (Go 1.20+)

Combine multiple errors (e.g. validating several fields at once):

```go
err := errors.Join(
    validateEmail(c.Email),
    validateName(c.Name),
) // nil if all inputs are nil; errors.Is works against any joined member
```

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Design the domain & a repository interface

**Task:** Create a `customer` package with:
- A `CustomerID` and `Email` value type (string-based).
- A `Customer` struct embedding shared `AuditFields`.
- A `CustomerRepository` interface with `Create` and `ByID`.
- A sentinel `ErrNotFound`.
- An in-memory implementation for testing.

Try it before reading the solution.

<details>
<summary>Reference solution</summary>

```go
// file: customer/customer.go
package customer

import (
    "context"
    "errors"
    "sync"
    "time"
)

type CustomerID string
type Email string

var ErrNotFound = errors.New("customer: not found")

type AuditFields struct {
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Customer struct {
    ID    CustomerID
    Email Email
    Name  string
    AuditFields
}

// Interface defined where it's *used* (the service layer would import this).
type CustomerRepository interface {
    Create(ctx context.Context, c Customer) (Customer, error)
    ByID(ctx context.Context, id CustomerID) (Customer, error)
}

// InMemoryRepo is a concrete implementation, handy for tests.
type InMemoryRepo struct {
    mu sync.RWMutex
    m  map[CustomerID]Customer
}

func NewInMemoryRepo() *InMemoryRepo {
    return &InMemoryRepo{m: make(map[CustomerID]Customer)}
}

func (r *InMemoryRepo) Create(_ context.Context, c Customer) (Customer, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    now := time.Now().UTC()
    c.CreatedAt, c.UpdatedAt = now, now
    r.m[c.ID] = c
    return c, nil
}

func (r *InMemoryRepo) ByID(_ context.Context, id CustomerID) (Customer, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    c, ok := r.m[id]
    if !ok {
        return Customer{}, ErrNotFound // sentinel; caller branches with errors.Is
    }
    return c, nil
}
```

**Reasoning:** Value types (`CustomerID`, `Email`) make signatures self-documenting
and prevent argument-order bugs (`ByID(email)` won't compile). `AuditFields` is
embedded because every entity shares it — and we'll reuse it on `Campaign`. The
repo returns the *concrete* `*InMemoryRepo`; the interface lives wherever a
consumer needs to swap implementations. `RWMutex` because reads dominate. We
return the sentinel directly so `errors.Is` works upstream.

</details>

### Exercise 3.2 — A validation error that carries data

**Task:** Add `Validate()` to `Customer` returning a `*ValidationError` (typed)
for a bad email, and combine multiple field errors with `errors.Join`. Then write
a caller that uses `errors.As` to report the offending field.

<details>
<summary>Reference solution</summary>

```go
// file: customer/validate.go
package customer

import (
    "errors"
    "fmt"
    "strings"
)

type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Msg)
}

func (c Customer) Validate() error {
    return errors.Join(
        validateEmail(c.Email),
        validateName(c.Name),
    )
}

func validateEmail(e Email) error {
    if !strings.Contains(string(e), "@") {
        return &ValidationError{Field: "email", Msg: "must contain @"}
    }
    return nil
}

func validateName(n string) error {
    if strings.TrimSpace(n) == "" {
        return &ValidationError{Field: "name", Msg: "must not be empty"}
    }
    return nil
}
```

```go
// caller
func report(err error) {
    var ve *ValidationError
    if errors.As(err, &ve) {
        fmt.Printf("bad field %q: %s\n", ve.Field, ve.Msg)
    }
}
```

**Reasoning:** `Validate` returns `error` (not `*ValidationError`) — avoiding the
typed-nil trap and keeping the door open for `errors.Join`. `errors.As` walks the
(possibly joined, possibly wrapped) chain and finds the first `*ValidationError`.

</details>

---

## 4. Fill-in-the-blank

Complete the `LoggingRepo` decorator so it logs and delegates, and the error
inspection helper. Blanks marked `/* ___ */`.

```go
type LoggingRepo struct {
    /* ___1: embed the interface so unoverridden methods are promoted ___ */
    log *slog.Logger
}

func (r LoggingRepo) ByID(ctx context.Context, id CustomerID) (Customer, error) {
    c, err := /* ___2: delegate to the embedded impl ___ */
    if err != nil {
        // log only NON-expected errors; ErrNotFound is normal control flow
        if !errors.Is(err, /* ___3 ___ */) {
            r.log.Error("byID failed", "id", id, "err", err)
        }
        return Customer{}, err
    }
    return c, nil
}
```

<details>
<summary>Answers</summary>

```go
type LoggingRepo struct {
    CustomerRepository                 // 1
    log *slog.Logger
}

func (r LoggingRepo) ByID(ctx context.Context, id CustomerID) (Customer, error) {
    c, err := r.CustomerRepository.ByID(ctx, id) // 2
    if err != nil {
        if !errors.Is(err, ErrNotFound) {        // 3
            r.log.Error("byID failed", "id", id, "err", err)
        }
        return Customer{}, err
    }
    return c, nil
}
```

Note blank 3's point: a "not found" is *expected*, so we don't log it as an error.
Logging expected conditions is a top source of alert fatigue.

</details>

---

## 5. Implement it yourself

**Problem:** Build an `errorx` helper package (used across Pulse) that provides:

1. A way to attach a stable, machine-readable **error code** (e.g. `NOT_FOUND`,
   `VALIDATION`, `CONFLICT`, `INTERNAL`) to any error, recoverable via a function
   like `Code(err) string` that walks the wrap chain.
2. An HTTP-status mapping (`HTTPStatus(err) int`) built on top of those codes.
3. Preserves wrapping — `errors.Is`/`errors.As` must still work through your type.

This is the bridge between your domain errors and the REST layer in M4.

**Curated resources:**
- Go blog, *Working with Errors in Go 1.13* — https://go.dev/blog/go1.13-errors
- Go blog, *Error handling and Go* — https://go.dev/blog/error-handling-and-go
- `errors` package docs — https://pkg.go.dev/errors
- Dave Cheney, *Don't just check errors, handle them gracefully* —
  https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully
- Effective Go, *Errors* section — https://go.dev/doc/effective_go#errors

**Hints:** define `type codedError struct { code string; err error }`, implement
`Error() string` and crucially `Unwrap() error` (this is what makes `errors.Is/As`
traverse your wrapper). Provide `WithCode(err error, code string) error`. For
`Code(err)`, loop with `errors.As` looking for `*codedError`.

---

## 6. Capstone contribution

You now have, in the Pulse repo:
- `customer` package: value types, `Customer` + `AuditFields`, `CustomerRepository`
  interface, `InMemoryRepo`, validation with typed errors, `ErrNotFound` sentinel.
- `errorx` package: codes + HTTP mapping (your "implement it yourself" output).

These are the contracts M4 (REST) and M6 (Postgres) implement against.

---

## 7. Self-check — you should now be able to…

- [ ] Explain why Go interfaces are defined at the point of use and kept small.
- [ ] Spot and fix the typed-nil interface bug.
- [ ] Choose sentinel vs typed vs wrapped errors for a given situation and justify it.
- [ ] Use `%w`, `errors.Is`, `errors.As`, and `errors.Join` correctly.
- [ ] Use embedding to build a decorator without rewriting promoted methods.

If any box is unchecked, re-read §2 and redo Exercise 3.2 before M2.
