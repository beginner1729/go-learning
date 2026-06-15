# Module 4 — REST APIs with `net/http` + `chi`

> **Capstone contribution:** Pulse's public **`customer-api`** — a versioned REST
> service with routing, middleware (request ID, logging, recovery, auth),
> JSON handling, request validation, and the domain→HTTP error mapping from M1.

---

## 0. Setup & run

```bash
cd go-cxm-course/m04-rest-api
go mod tidy
go vet ./...
go test ./...
go run ./cmd/api      # serves on :8080
# in another shell:
curl -s localhost:8080/healthz
curl -s -XPOST localhost:8080/v1/customers -H 'Authorization: Bearer dev-token' \
  -H 'Content-Type: application/json' -d '{"email":"ada@pulse.dev","name":"Ada"}' | jq
```

Layout:

```
m04-rest-api/
  go.mod                 # module cxm/m04
  customer/              # domain (carried from M1, trimmed)
  httpapi/               # handlers, middleware, router, DTOs, error mapping
  cmd/api/               # main: wires router + http.Server with graceful shutdown
```

---

## 1. Learning objectives

By the end you will be able to:

- Build an idiomatic HTTP service with `net/http` and the `chi` router.
- Write composable **middleware** (request ID, structured logging, panic recovery, auth).
- Decode/validate JSON safely and return consistent, structured error responses.
- Version an API and structure handlers as testable units (`httptest`).
- Run an `http.Server` with timeouts and **graceful shutdown**.

---

## 2. Concepts

### 2.1 `http.Handler`, `http.HandlerFunc`, and why `chi`

Everything in Go HTTP is the `http.Handler` interface:

```go
type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

`chi` is a router that is **100% `http.Handler`-compatible** — its middleware and
handlers are stdlib types, so there's no framework lock-in. We use it for clean
path params, sub-routers, and a solid middleware stack. (Go 1.22's stdlib router
now supports `GET /v1/customers/{id}` patterns too; we note that, but `chi`'s
middleware ergonomics win for a real service.)

**Why not a big framework (Gin/Echo/Fiber)?** They're fine, but they introduce
their own context types and idioms. For an enterprise codebase, staying on
stdlib `http.Handler` keeps every middleware/library in the ecosystem compatible.
This is a deliberate, conservative choice.

### 2.2 Middleware = handler decorators

Middleware is a function `func(http.Handler) http.Handler`. It wraps the next
handler, doing work before/after. This is the M1 decorator pattern applied to HTTP.

```go
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Request-ID")
        if id == "" {
            id = newID()
        }
        ctx := context.WithValue(r.Context(), ridKey{}, id) // request-scoped value
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Order matters: recovery outermost (so it catches panics from everything), then
request ID, then logging, then auth. `chi` applies them top-down.

#### Pitfall: writing to the body before the status code

`w.WriteHeader(status)` must come **before** any `w.Write`. The first `Write`
implicitly sends `200`. So set headers → `WriteHeader` → `Write`. Our JSON helper
enforces this.

### 2.3 JSON: decoding safely, encoding consistently

Decoding pitfalls and the idiomatic guard rails:

```go
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
    var v T
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields() // reject typos / unexpected fields
    if err := dec.Decode(&v); err != nil {
        return v, fmt.Errorf("decode body: %w", err)
    }
    // Ensure there's no trailing garbage / second JSON value.
    if dec.More() {
        return v, errors.New("body must contain a single JSON object")
    }
    return v, nil
}
```

Also cap the body size with `http.MaxBytesReader` to prevent memory-exhaustion.
Encode through one helper so every response is consistent:

```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
```

#### DTOs vs domain types

**Never serialize your domain struct directly.** Define request/response **DTOs**
in the HTTP layer. This (a) decouples your wire format from internal types,
(b) lets you control exactly which fields leak — crucial for **PII** (don't echo
internal IDs, don't leak fields), and (c) keeps JSON tags out of your domain.

### 2.4 Validation

Validate at the edge, return a structured 400. Two idioms:
- **Hand-rolled** `Validate()` on the DTO (explicit, no deps) — what we do here,
  reusing M1's `ValidationError`.
- **`go-playground/validator`** with struct tags (`validate:"required,email"`) for
  larger APIs. We mention it; hand-rolled is clearer for learning.

### 2.5 Error responses & the M1 bridge

Map domain errors to HTTP via `errorx` (M1): one `respondError` helper inspects
the error's code and emits `{ "error": { "code": "...", "message": "..." } }`.
This is why we built `errorx` first.

### 2.6 The server: timeouts + graceful shutdown

A bare `http.ListenAndServe` has **no timeouts** — a slow client can hold a
connection forever (Slowloris). Always configure an `http.Server`:

```go
srv := &http.Server{
    Addr:              addr,
    Handler:           router,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

Graceful shutdown: catch SIGINT/SIGTERM, call `srv.Shutdown(ctx)` to stop
accepting new connections and let in-flight requests finish within a deadline.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — A recovery middleware that returns 500 JSON

**Task:** Write middleware that recovers from a panic in any downstream handler,
logs it with the request ID, and returns a clean `500` JSON body (never a leaked
stack trace to the client).

<details>
<summary>Reference solution</summary>

```go
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if rec := recover(); rec != nil {
                    log.Error("panic recovered",
                        "err", rec,
                        "request_id", RequestIDFrom(r.Context()),
                        "stack", string(debug.Stack()))
                    writeJSON(w, http.StatusInternalServerError, errorBody{
                        Error: errorDetail{Code: "INTERNAL", Message: "internal server error"},
                    })
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

**Reasoning:** The stack goes to logs, not the client. It's a closure over `log`
(middleware-with-config idiom). Placed outermost so it catches everything below.

</details>

### Exercise 3.2 — Create-customer handler with validation

**Task:** `POST /v1/customers` decodes a `createCustomerRequest` DTO, validates it,
calls the service, and returns `201` with a `customerResponse` DTO — or the right
error status via `errorx`.

<details>
<summary>Reference solution</summary>

```go
type createCustomerRequest struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

func (r createCustomerRequest) toDomain() customer.Customer {
    return customer.Customer{Email: customer.Email(r.Email), Name: r.Name}
}

type customerResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    Name      string `json:"name"`
    CreatedAt string `json:"created_at"`
}

func (h *Handler) createCustomer(w http.ResponseWriter, r *http.Request) {
    req, err := decodeJSON[createCustomerRequest](w, r)
    if err != nil {
        respondError(w, errorx.WithCode(err, errorx.CodeValidation))
        return
    }
    c := req.toDomain()
    if err := c.Validate(); err != nil {
        respondError(w, errorx.WithCode(err, errorx.CodeValidation))
        return
    }
    created, err := h.svc.Create(r.Context(), c)
    if err != nil {
        respondError(w, err) // service already tagged the error with a code
        return
    }
    writeJSON(w, http.StatusCreated, toResponse(created))
}
```

**Reasoning:** DTO in, DTO out — the domain `Customer` never touches the wire. All
error paths funnel through `respondError`, which uses `errorx.HTTPStatus`. The
handler is thin; business logic lives in `h.svc`.

</details>

---

## 4. Fill-in-the-blank

Complete the auth middleware and the route mounting.

```go
func BearerAuth(token string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
            if got == "" || got != token {
                respondError(w, errorx.WithCode(errors.New("unauthorized"), /* ___1 ___ */))
                return
            }
            /* ___2: call the next handler ___ */
        })
    }
}

func (h *Handler) Routes() http.Handler {
    r := chi.NewRouter()
    r.Use(Recover(h.log), RequestID, Logging(h.log))
    r.Get("/healthz", h.health)
    r.Route("/v1", func(r chi.Router) {
        r.Use(BearerAuth(h.token))           // protect everything under /v1
        r.Post("/customers", h.createCustomer)
        r.Get("/customers/{id}", h./* ___3 ___ */)
    })
    return r
}
```

<details>
<summary>Answers</summary>

```go
// 1: a 401 maps from a dedicated code; add CodeUnauthorized to errorx, or reuse
//    a mapping. Here we add errorx.CodeUnauthorized -> 401.
respondError(w, errorx.WithCode(errors.New("unauthorized"), errorx.CodeUnauthorized))

// 2:
next.ServeHTTP(w, r)

// 3:
r.Get("/customers/{id}", h.getCustomer)
```

(We extend `errorx` with `CodeUnauthorized -> 401`; the runnable code includes it.)

</details>

---

## 5. Implement it yourself

**Problem:** Extend `customer-api` with:
- `GET /v1/customers` — paginated list using M2's `genx.Paginate`, returning a
  `{ items, page, total_pages, has_next }` envelope. Read `?page=` & `?size=`.
- `PATCH /v1/customers/{id}` — partial update (only provided fields change). Think
  about how to distinguish "field absent" from "field set to empty" (`*string`).
- A **PII-aware** response mode: add a middleware/flag that redacts email to
  `a***@pulse.dev` unless the caller has an `X-Scope: pii` header.
- Table-driven `httptest` tests for each.

**Curated resources:**
- `net/http` docs — https://pkg.go.dev/net/http
- `chi` — https://github.com/go-chi/chi  · examples: https://github.com/go-chi/chi/tree/master/_examples
- Go 1.22 routing enhancements — https://go.dev/blog/routing-enhancements
- *How I write HTTP services in Go* (Mat Ryer) — https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/
- `httptest` — https://pkg.go.dev/net/http/httptest
- `go-playground/validator` — https://github.com/go-playground/validator

**Hints:** for PATCH use a DTO of pointers (`Name *string`); nil means "leave
alone". For pagination, validate/clamp query params before calling `Paginate`.

---

## 6. Capstone contribution

Pulse now has a running, versioned, authenticated REST `customer-api` with
middleware, validation, JSON DTOs, error mapping, and graceful shutdown. M5 adds
the gRPC side; M6 swaps the in-memory service for a Postgres-backed one behind the
same handler interface (the payoff of M1's repository abstraction).

---

## 7. Self-check — you should now be able to…

- [ ] Explain why staying on `http.Handler` keeps the ecosystem composable.
- [ ] Write ordered middleware (recovery → request ID → logging → auth).
- [ ] Decode JSON defensively and return consistent structured errors.
- [ ] Keep domain types off the wire using DTOs (and why that matters for PII).
- [ ] Run an `http.Server` with timeouts and graceful shutdown.
