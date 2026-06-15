# Module 11 — Testing & Quality Gates

> **Capstone contribution:** Pulse's test suite and quality gates — table-driven
> unit tests, hand-written fakes, integration tests with **testcontainers**,
> coverage, **`golangci-lint`**, and **pre-commit hooks**. The safety net that
> lets you refactor the capstone fearlessly.

---

## 0. Setup & run

```bash
cd go-cxm-course/m11-testing
go mod tidy
go test ./...                      # unit tests (fast)
go test -race -cover ./...         # with race detector + coverage
go test -run Integration ./...     # integration tests (need Docker; auto-skip otherwise)
golangci-lint run ./...            # if installed (see §2.5)
```

Layout:

```
m11-testing/
  go.mod
  notify/                # subject under test: a notifier with a Sender dependency
  notify/notify_test.go  # table-driven unit tests + fake Sender
  notify/integration_test.go  # testcontainers Postgres example (build-tagged)
  .golangci.yml          # lint config
  .pre-commit-config.yaml
```

---

## 1. Learning objectives

By the end you will be able to:

- Write idiomatic **table-driven tests** and subtests with `t.Run`.
- Build **fakes/mocks** by hand (and know when to use a mock library).
- Write **integration tests** against real dependencies via **testcontainers-go**.
- Measure and reason about **coverage** without chasing 100%.
- Configure **`golangci-lint`** and **pre-commit** hooks as CI gates.

---

## 2. Concepts

### 2.1 Table-driven tests — the Go default

The idiomatic pattern: a slice of cases, one loop, subtests for isolation:

```go
func TestClassify(t *testing.T) {
    tests := []struct {
        name string
        in   int
        want string
    }{
        {"negative", -1, "invalid"},
        {"zero", 0, "empty"},
        {"positive", 5, "ok"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := Classify(tt.in); got != tt.want {
                t.Errorf("Classify(%d) = %q, want %q", tt.in, got, tt.want)
            }
        })
    }
}
```

Why: adding a case is one line; `t.Run` gives each case its own name in output and
lets you run one with `-run TestClassify/zero`; failures don't stop other cases
(`Errorf` not `Fatalf` when you want to see all failures).

> **Use the standard library `testing` + plain assertions** for most things.
> `testify/require` + `testify/assert` are widely used and fine for terser
> assertions; just be consistent. Avoid heavyweight BDD frameworks — they're
> non-idiomatic in Go.

### 2.2 Fakes vs mocks vs stubs

Go interfaces make test doubles trivial. Prefer **hand-written fakes** — a small
struct implementing the interface with controllable behavior:

```go
type fakeSender struct {
    sent     []notify.Message
    failNext bool
}

func (f *fakeSender) Send(_ context.Context, m notify.Message) error {
    if f.failNext {
        f.failNext = false
        return errors.New("boom")
    }
    f.sent = append(f.sent, m)
    return nil
}
```

This is clearer and more refactor-stable than generated mocks for small
interfaces. Reach for a **mock library** (`mockery`/`gomock`) only when interfaces
are large or you need rich call-expectation assertions. The Go ethos: small
interfaces → hand-written fakes.

> **Anti-pattern:** mocking types you don't own (e.g. the `*sql.DB`). Mock at
> *your* interface boundary (the `Repository`), not the database driver. For the
> database itself, use a real one via testcontainers (§2.4).

### 2.3 Test structure & helpers

- **Arrange-Act-Assert** in each test.
- `t.Helper()` in helper funcs so failures report the caller's line.
- `t.Cleanup(fn)` for teardown (runs even on failure; composes better than `defer`).
- `t.Parallel()` for independent tests (careful with shared state).
- **Golden files** for large outputs: compare against `testdata/*.golden`, update
  with a `-update` flag.

### 2.4 Integration tests with testcontainers

Unit tests use fakes; **integration tests use the real thing** in an ephemeral
Docker container, so they're hermetic and CI-friendly:

```go
//go:build integration

func TestIntegration_Repo(t *testing.T) {
    ctx := context.Background()
    pg, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("pulse"),
        postgres.WithUsername("pulse"), postgres.WithPassword("pulse"),
        testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
    )
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { _ = pg.Terminate(ctx) })

    dsn, _ := pg.ConnectionString(ctx, "sslmode=disable")
    // ... migrate, construct repo, run real queries ...
}
```

Gate them behind a **build tag** (`//go:build integration`) or skip when Docker is
absent, so `go test ./...` stays fast and unit-only by default; run
`go test -tags integration ./...` in CI's integration stage.

### 2.5 `golangci-lint`

The standard meta-linter — runs many linters in one pass. A sane starter config:

```yaml
# .golangci.yml
run:
  timeout: 5m
linters:
  enable:
    - errcheck      # unchecked errors
    - govet         # suspicious constructs
    - staticcheck   # the big one — correctness + simplifications
    - revive        # style
    - ineffassign   # ineffective assignments
    - gosec         # security issues
    - bodyclose     # unclosed http bodies
issues:
  exclude-rules:
    - path: _test\.go
      linters: [gosec]
```

Run it in CI as a required check. It catches whole classes of bugs (unchecked
errors, unclosed bodies, races in patterns) before review.

### 2.6 Pre-commit hooks

Run fast checks before each commit so problems never reach CI:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: gofmt
        name: gofmt
        entry: gofmt -l -w
        language: system
        types: [go]
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        pass_filenames: false
```

#### Pitfalls

- **Chasing 100% coverage** → tests for trivial getters, brittle suites. Cover
  behavior and edge cases; coverage is a *signal*, not a goal.
- **Testing implementation details** → tests break on every refactor. Test public
  behavior through the interface.
- **Flaky tests from time/concurrency** → inject a clock; use `synctest`/channels,
  not `time.Sleep`, to coordinate. Run `-race` always.
- **Shared mutable state across parallel tests** → subtle failures; isolate
  per-test (fresh DB schema, unique keys).

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Table-test a notifier with a fake

**Task:** Test a `Notifier.Welcome(ctx, customer)` that calls a `Sender`. Cover:
success records a message; a `Sender` error is propagated; an invalid customer is
rejected *before* calling `Sender`.

<details>
<summary>Reference solution</summary>

See `m11-testing/notify/notify_test.go`. Key shape:

```go
func TestNotifier_Welcome(t *testing.T) {
    tests := []struct {
        name      string
        cust      notify.Customer
        failSend  bool
        wantErr   bool
        wantSends int
    }{
        {"ok", notify.Customer{Email: "a@b.com", Name: "Ada"}, false, false, 1},
        {"send fails", notify.Customer{Email: "a@b.com", Name: "Ada"}, true, true, 0},
        {"invalid email", notify.Customer{Email: "x", Name: "Ada"}, false, true, 0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            f := &fakeSender{failNext: tt.failSend}
            n := notify.New(f)
            err := n.Welcome(context.Background(), tt.cust)
            if (err != nil) != tt.wantErr {
                t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
            }
            if len(f.sent) != tt.wantSends {
                t.Fatalf("sends = %d, want %d", len(f.sent), tt.wantSends)
            }
        })
    }
}
```

**Reasoning:** the fake lets us assert *both* the returned error and the side
effect (did it actually send?). The "invalid email" case proves validation happens
before the dependency is touched — a behavior worth locking in.

</details>

### Exercise 3.2 — A `t.Cleanup`-based fixture

**Task:** Write a `newFixture(t)` helper that builds a `Notifier` + fake and
registers cleanup, returning both — so each test is one line of setup.

<details>
<summary>Reference solution</summary>

```go
func newFixture(t *testing.T) (*notify.Notifier, *fakeSender) {
    t.Helper()
    f := &fakeSender{}
    n := notify.New(f)
    t.Cleanup(func() { /* close resources, reset globals, etc. */ })
    return n, f
}
```

**Reasoning:** `t.Helper()` makes assertion failures point at the test, not the
helper. `t.Cleanup` runs even if the test fails or calls `t.Fatal`, and composes
across nested helpers better than stacked `defer`s.

</details>

---

## 4. Fill-in-the-blank

```go
func TestRetry(t *testing.T) {
    cases := []struct {
        name    string
        fails   int
        wantErr bool
    }{
        {"succeeds first try", 0, false},
        {"succeeds after retries", 2, false},
        {"exhausts retries", 5, true},
    }
    for _, tt := range cases {
        t./* ___1: subtest ___ */(tt.name, func(t *testing.T) {
            f := &fakeSender{failTimes: tt.fails}
            err := SendWithRetry(context.Background(), f, msg, 3)
            if (err != nil) != tt.wantErr {
                t./* ___2: fail and stop this subtest ___ */("err=%v want %v", err, tt.wantErr)
            }
        })
    }
}
```

<details>
<summary>Answers</summary>

```go
t.Run(tt.name, func(t *testing.T) { ... }) // 1
t.Fatalf("err=%v want %v", err, tt.wantErr) // 2
```

</details>

---

## 5. Implement it yourself

**Problem:** Build the capstone's test pyramid:
- **Unit tests** for every service in `internal/` using hand-written fakes for
  repos/senders. Aim for fast (<1s) and deterministic.
- **Integration tests** (build tag `integration`) with testcontainers for the
  Postgres repo (M6), the Mongo repo (M7), and a JetStream flow (M9).
- An **HTTP e2e test** that spins the `customer-api` with `httptest.Server` and
  exercises create→get→list.
- Wire `golangci-lint` + pre-commit, and add a `make test` / `make lint` target.
- Add a coverage gate in CI (e.g. fail under 70% on `internal/`).

**Curated resources:**
- `testing` package — https://pkg.go.dev/testing
- *Test-driven development* tour — https://go.dev/doc/tutorial/add-a-test
- testcontainers-go — https://golang.testcontainers.org/
- `golangci-lint` — https://golangci-lint.run/
- pre-commit — https://pre-commit.com/
- Dave Cheney, *Prefer table driven tests* — https://dave.cheney.net/2019/05/07/prefer-table-driven-tests
- `testify` — https://github.com/stretchr/testify  ·  `mockery` — https://vektra.github.io/mockery/

**Hints:** keep integration tests behind `//go:build integration` so the default
`go test ./...` is unit-only and fast. Reuse one container across a package's
integration tests with `TestMain` to cut startup cost.

---

## 6. Capstone contribution

Pulse now has a layered test suite (unit + integration + e2e), lint config, and
pre-commit hooks. You can refactor any service with confidence. M12 wires these
into CI so every push is gated automatically.

---

## 7. Self-check — you should now be able to…

- [ ] Write table-driven tests with subtests and good failure messages.
- [ ] Build hand-written fakes and know when a mock library is warranted.
- [ ] Write hermetic integration tests with testcontainers behind a build tag.
- [ ] Configure `golangci-lint` and pre-commit as quality gates.
- [ ] Treat coverage as a signal and avoid testing implementation details.
