# Module 2 — Generics & Idiomatic Project Idioms

> **Capstone contribution:** a small, reusable **`genx`** toolkit (sets, slice
> helpers, pagination) and the **functional-options** constructor pattern Pulse
> uses for every service/server. Also a **generic in-memory store** that removes
> the per-entity boilerplate you wrote in M1.

---

## 0. Setup & run

```bash
cd go-cxm-course/m02-generics-and-idioms
go mod tidy
go vet ./...
go test ./...
go run ./cmd/demo
```

Layout:

```
m02-generics-and-idioms/
  go.mod                 # module cxm/m02
  genx/                  # generic helpers: Set, Map/Filter/Keys, Page pagination
  store/                 # generic in-memory Store[K,V] (replaces M1's hand-written repo)
  config/                # functional-options pattern (ServerConfig)
  cmd/demo/
```

---

## 1. Learning objectives

By the end you will be able to:

- Write generic functions and types with **type parameters** and **constraints**.
- Choose correctly between **generics, interfaces, and code generation**.
- Use the `~` (underlying type) and `comparable` constraints.
- Apply the **functional-options** pattern for flexible, backward-compatible constructors.
- Recognize Go project idioms: zero-value-useful, constructors, `Must*`, package naming.
- Avoid the big generics anti-patterns (using them where an interface is clearer).

---

## 2. Concepts

### 2.1 Type parameters & constraints

Generics (Go 1.18+) let you write code parameterized by type:

```go
// Map applies f to each element, returning a new slice. [T any, U any] are
// type parameters; `any` is their constraint (alias for interface{}).
func Map[T, U any](s []T, f func(T) U) []U {
    out := make([]U, len(s))
    for i, v := range s {
        out[i] = f(v)
    }
    return out
}

names := Map([]Customer{{Name: "Ada"}}, func(c Customer) string { return c.Name })
```

A **constraint** is an interface used as a type bound. It can list methods,
types, or both:

```go
type Number interface {
    ~int | ~int64 | ~float64 // ~ means "any type whose underlying type is this"
}

func Sum[T Number](xs []T) T {
    var total T // zero value of T
    for _, x := range xs {
        total += x
    }
    return total
}
```

The `~` matters: `type CustomerCount int` has underlying type `int`, so it
satisfies `~int` but not a bare `int` union. Use `~` when you want to accept
named types built on a primitive.

`comparable` is a built-in constraint for types usable with `==` / map keys:

```go
func Contains[T comparable](s []T, target T) bool {
    for _, v := range s {
        if v == target {
            return true
        }
    }
    return false
}
```

### 2.2 When generics vs interfaces vs codegen

This is the senior judgment call:

| Use | When |
|---|---|
| **Interface** | Behavior varies, types don't need to be related, you call *methods*. (`io.Writer`, repositories.) Most polymorphism in Go is still interfaces. |
| **Generics** | You operate on *data* uniformly regardless of type and want to avoid `interface{}` + type assertions: containers (Set, cache, queue), slice/map utilities, type-safe pools. |
| **Codegen** | The variation is structural and known at build time (e.g. `sqlc`, protobuf). Generates concrete, debuggable code. |

> **Anti-pattern:** reaching for generics when an interface is clearer. If your
> generic function immediately calls methods on `T` via a constraint interface,
> you probably just want a plain interface parameter. Generics shine when the
> type is *carried through* (in, out, stored) without method calls.

Other generics limits to know: **no type parameters on methods** (only on funcs
and types), and generics don't give you specialization/overloading.

### 2.3 The functional-options pattern

The idiomatic way to build a configurable constructor that stays
backward-compatible as options grow:

```go
type Server struct {
    addr    string
    timeout time.Duration
    log     *slog.Logger
}

type Option func(*Server)

func WithTimeout(d time.Duration) Option { return func(s *Server) { s.timeout = d } }
func WithLogger(l *slog.Logger) Option    { return func(s *Server) { s.log = l } }

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{ // sensible defaults
        addr:    addr,
        timeout: 30 * time.Second,
        log:     slog.Default(),
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

srv := NewServer(":8080", WithTimeout(5*time.Second)) // logger defaults
```

**Why not a config struct?** A struct is fine and simpler for purely-data config.
Options win when you need defaults, validation, computed/dependent fields, or to
keep some fields unexported. Both are idiomatic — choose by complexity.

### 2.4 Other idioms you'll use constantly

- **Zero value useful:** design types so the zero value works (`var b bytes.Buffer`
  is ready to use; `sync.Mutex` zero value is unlocked). Avoid requiring a
  constructor unless you must.
- **Constructors `NewT`** return `*T` or `T`; if construction can fail, return
  `(T, error)`.
- **`Must*` helpers** panic on error — only for package init / tests where a
  failure is a programming error: `var re = regexp.MustCompile(...)`.
- **Package naming:** short, lower-case, no underscores; the name is part of the
  API (`customer.Customer` reads oddly — prefer `customer.Repository`). Avoid
  stutter and avoid grab-bag `util`/`common` packages.

#### Pitfall: leaking generics into your public API needlessly

If `Page[T]` appears in every signature, callers must thread type params
everywhere. Keep generic plumbing internal where you can; expose concrete types
at API boundaries (e.g. REST returns `PageResponse` JSON, not `Page[Customer]`).

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — A generic `Set[T]`

**Task:** Implement `Set[T comparable]` with `Add`, `Has`, `Remove`, `Len`, and
`Items() []T`. Make the zero value unusable on purpose? No — make `New` required
since it needs a map.

<details>
<summary>Reference solution</summary>

```go
type Set[T comparable] struct {
    m map[T]struct{}
}

func NewSet[T comparable](items ...T) *Set[T] {
    s := &Set[T]{m: make(map[T]struct{}, len(items))}
    for _, it := range items {
        s.Add(it)
    }
    return s
}

func (s *Set[T]) Add(v T)    { s.m[v] = struct{}{} }
func (s *Set[T]) Remove(v T) { delete(s.m, v) }
func (s *Set[T]) Has(v T) bool {
    _, ok := s.m[v]
    return ok
}
func (s *Set[T]) Len() int { return len(s.m) }
func (s *Set[T]) Items() []T {
    out := make([]T, 0, len(s.m))
    for v := range s.m {
        out = append(out, v)
    }
    return out
}
```

**Reasoning:** `struct{}{}` values cost zero bytes — the canonical Go set. The
constructor is required because a nil map can't be written to (writing to a nil
map panics; reading is fine). `comparable` is exactly the constraint map keys need.

</details>

### Exercise 3.2 — Generic pagination `Page[T]`

**Task:** Implement `Paginate[T any](items []T, page, size int) Page[T]` returning
a slice window plus metadata (total, page, size, totalPages, hasNext). Guard bad inputs.

<details>
<summary>Reference solution</summary>

```go
type Page[T any] struct {
    Items      []T
    Page       int
    Size       int
    Total      int
    TotalPages int
    HasNext    bool
}

func Paginate[T any](items []T, page, size int) Page[T] {
    if page < 1 {
        page = 1
    }
    if size < 1 {
        size = 20
    }
    total := len(items)
    start := (page - 1) * size
    if start > total {
        start = total
    }
    end := start + size
    if end > total {
        end = total
    }
    totalPages := (total + size - 1) / size
    return Page[T]{
        Items:      items[start:end],
        Page:       page,
        Size:       size,
        Total:      total,
        TotalPages: totalPages,
        HasNext:    page < totalPages,
    }
}
```

**Reasoning:** All index math is clamped so out-of-range pages return an empty
window instead of panicking. `(total+size-1)/size` is the integer ceil-division
idiom. This `Page[T]` is internal; the REST layer (M4) maps it to a JSON DTO.

</details>

---

## 4. Fill-in-the-blank

Complete the generic in-memory store that replaces M1's hand-written `InMemoryRepo`.

```go
type Store[K comparable, V any] struct {
    mu sync.RWMutex
    m  map[K]V
}

func NewStore[K comparable, V any]() *Store[K, V] {
    return &Store[K, V]{m: /* ___1 ___ */}
}

func (s *Store[K, V]) Put(key K, val V) {
    s.mu.Lock()
    /* ___2: unlock at end ___ */
    s.m[key] = val
}

func (s *Store[K, V]) Get(key K) (V, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    v, ok := s.m[key]
    return /* ___3 ___ */
}
```

<details>
<summary>Answers</summary>

```go
func NewStore[K comparable, V any]() *Store[K, V] {
    return &Store[K, V]{m: make(map[K]V)} // 1
}

func (s *Store[K, V]) Put(key K, val V) {
    s.mu.Lock()
    defer s.mu.Unlock() // 2
    s.m[key] = val
}

func (s *Store[K, V]) Get(key K) (V, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    v, ok := s.m[key]
    return v, ok // 3
}
```

Note: `Get` returns `(V, bool)` rather than `V` + sentinel — the comma-ok idiom
generalizes cleanly and avoids needing a typed zero/sentinel per V.

</details>

---

## 5. Implement it yourself

**Problem:** Build a generic, thread-safe **TTL cache** `Cache[K comparable, V any]`:
- `Set(k, v, ttl)`, `Get(k) (V, bool)` (miss if expired), `Delete(k)`.
- A background janitor goroutine that evicts expired entries, stopped via a
  `Close()` method. (You'll formalize the goroutine-lifecycle/`context` parts in M3 —
  here just get it working and race-free.)
- Run your tests with `-race`.

This becomes Pulse's lookup cache (e.g. caching customer profiles fetched over gRPC).

**Curated resources:**
- *An Introduction to Generics* (Go blog) — https://go.dev/blog/intro-generics
- *When To Use Generics* (Go blog) — https://go.dev/blog/when-generics
- Tutorial: Getting started with generics — https://go.dev/doc/tutorial/generics
- `sync.RWMutex` / `sync.Map` docs — https://pkg.go.dev/sync
- Dave Cheney, *Functional options for friendly APIs* —
  https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

**Hints:** store `struct{ val V; expiresAt time.Time }`. The janitor uses a
`time.Ticker`; `Close()` closes a `done` channel the janitor selects on. Guard the
map with `sync.RWMutex`. Test expiry by injecting a clock or using short TTLs.

---

## 6. Capstone contribution

Added to Pulse:
- `genx`: `Set`, `Map`/`Filter`/`Keys`/`Values`, `Page`/`Paginate` — used by the
  REST list endpoints and service layer.
- `store.Store[K,V]`: a generic in-memory store; the M1 `InMemoryRepo` can now be
  expressed in a few lines, and it's reused for test fakes throughout the course.
- `config` functional options: the constructor style every Pulse server/service uses.
- (Your TTL cache from §5 → profile cache in front of gRPC calls.)

---

## 7. Self-check — you should now be able to…

- [ ] Write a generic function/type with appropriate constraints (`any`, `comparable`, `~`).
- [ ] Articulate when generics beat interfaces and when they don't.
- [ ] Implement the functional-options pattern and explain why it beats a growing constructor.
- [ ] Build a thread-safe generic container and test it with `-race`.
- [ ] Keep generics out of your public API surface where concrete types read better.
