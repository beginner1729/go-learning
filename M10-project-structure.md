# Module 10 — Enterprise Project Structure & Modules

> **Capstone contribution:** unifies everything from M1–M9 into **one coherent
> Pulse repository** with a standard layout, shared `internal/` packages, a Go
> workspace, and clear service boundaries. This is the skeleton the capstone fills.

---

## 0. Setup & run

```bash
cd go-cxm-course/m10-project-structure/pulse
go work sync      # ties the modules together (if using a workspace)
go build ./...    # everything compiles
go test ./...
```

Layout produced in this module (the canonical Pulse repo):

```
pulse/
  go.mod                      # module github.com/acme/pulse
  cmd/                        # one dir per binary (main packages only)
    customer-api/main.go
    profile-svc/main.go
    notification-svc/main.go
  internal/                   # private packages — cannot be imported externally
    customer/                 # domain + service (M1, M4, M6)
    profile/                  # domain + service (M5, M7)
    notification/             # dispatcher (M3) + consumer (M9)
    platform/                 # shared infra: db, mongo, nats, httpx, config, log
      postgres/  mongo/  nats/  httpx/  config/  observability/
    errorx/                   # error codes (M1)
  api/
    proto/profile/v1/         # protobuf contracts (M5)
    openapi/customer.yaml     # REST contract (optional)
  migrations/                 # goose SQL (M6)
  deploy/                     # Dockerfiles, compose, k8s (M12)
  Makefile  .golangci.yml  README.md
```

---

## 1. Learning objectives

By the end you will be able to:

- Apply the de-facto **standard Go project layout** and justify each directory.
- Use **`internal/`** to enforce package privacy and **`cmd/`** for binaries.
- Decide **monorepo vs multi-repo** and use **Go workspaces** (`go.work`).
- Structure code by **domain** (not by technical layer) and keep dependencies
  pointing inward.
- Manage modules, versions, and dependencies cleanly.

---

## 2. Concepts

### 2.1 The standard layout (and what's myth)

There is no *official* layout, but the community converged on conventions:

- **`cmd/<binary>/main.go`** — each executable gets a directory; `main` packages
  are thin (parse config, wire dependencies, start). No business logic here.
- **`internal/`** — the compiler **enforces** that packages under `internal/` can
  only be imported by code rooted at the parent of `internal/`. This is your
  encapsulation boundary: external repos *cannot* import your guts. Put almost
  everything here.
- **`pkg/`** — for code you *intend* others to import. Most apps don't need it;
  prefer `internal/` until you actually publish a library. (The `pkg/` dir is
  somewhat controversial — don't cargo-cult it.)
- **`api/`** — interface contracts: `.proto`, OpenAPI specs.
- **`migrations/`, `deploy/`, `configs/`** — self-explanatory.

> **Anti-pattern: layered packages** (`controllers/`, `services/`, `models/`,
> `repositories/`). This scatters one feature across four packages and creates
> import cycles. **Organize by domain** (`customer/`, `profile/`,
> `notification/`), each containing its handler+service+repo. This is "package by
> feature," and it's how idiomatic Go scales.

### 2.2 Dependencies point inward

Domain packages (`internal/customer`) define interfaces; infrastructure
(`internal/platform/postgres`) implements them. The domain knows nothing about
Postgres, NATS, or HTTP. This is the M1 lesson at repo scale: the HTTP handler
depends on a `customer.Service`, the service depends on a `customer.Repository`
interface, and the Postgres repo satisfies it. You can test the domain with fakes
and swap infrastructure without touching business logic.

```
cmd/customer-api ─► internal/customer (service, handler)
                         │ depends on
                         ▼
                 customer.Repository (interface)
                         ▲ implemented by
                         │
        internal/platform/postgres.CustomerRepo
```

### 2.3 `internal/platform` — the shared kernel

Cross-cutting infrastructure shared by all services lives in one place:
`config` (load env), `log` (slog setup), `postgres`/`mongo`/`nats` (connection
constructors), `httpx` (middleware, JSON helpers), `observability` (metrics,
tracing). Services import these; they don't reimplement connection logic.

### 2.4 Monorepo vs multi-repo & Go workspaces

- **Monorepo** (one repo, possibly multiple modules): atomic cross-service
  changes, shared tooling, easy refactors. Most teams start here. Pulse is a
  monorepo with a single module (simplest) — multiple binaries under `cmd/`.
- **Multi-repo**: independent release cadence and ownership; cost is cross-repo
  coordination and version skew.
- **Go workspaces (`go.work`)**: when you *do* split into multiple modules in one
  repo (e.g. a separately-versioned client SDK), `go.work` lets them resolve each
  other locally without `replace` directives:

```
go work init ./pulse ./pulse-sdk
go work sync
```

For Pulse we use a **single module, multiple binaries** — the least-friction
choice until scale demands otherwise.

### 2.5 Module hygiene

- `go mod tidy` after every dependency change; commit `go.sum`.
- Pin tool versions (e.g. `golangci-lint`, `goose`) in a `tools.go` or the
  Makefile, not just "whatever's on the dev's machine."
- Use **semantic import versioning** for v2+ modules (`/v2` in the path).
- Keep one `go.mod` per module at its root; don't nest modules accidentally
  (a stray `go.mod` in a subdir creates a separate module).

#### Pitfalls

- **Business logic in `main`** → untestable, unreusable. Keep `main` thin.
- **A `utils`/`common` junk-drawer package** → becomes a dependency magnet and
  import-cycle source. Name packages for what they *do*.
- **Importing `internal/` from another repo** → won't compile (by design).
- **Circular imports** → Go forbids them; they signal bad boundaries. Break the
  cycle by introducing an interface in the consumer.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Lay out the Pulse repo skeleton

**Task:** Create the directory tree above with placeholder packages so
`go build ./...` succeeds, and a thin `cmd/customer-api/main.go` that wires a
config + logger + (stub) service.

<details>
<summary>Reference solution</summary>

See `m10-project-structure/pulse/` in this module — it contains the compiling
skeleton: `internal/platform/config`, `internal/platform/log`, a domain
`internal/customer` with an interface + in-memory impl, and three thin `cmd/`
mains. Key idea — `main` only wires:

```go
func main() {
    cfg := config.Load()
    logger := log.New(cfg.LogLevel)
    repo := customer.NewInMemoryRepo()
    svc := customer.NewService(repo)
    srv := httpx.NewServer(cfg.HTTPAddr, customer.NewHandler(svc, logger))
    srv.Run() // blocks; handles graceful shutdown internally
}
```

**Reasoning:** every dependency is constructed at the top and injected downward
(manual dependency injection — Go rarely needs a DI framework). Swapping
`NewInMemoryRepo` for `postgres.NewCustomerRepo(pool)` is a one-line change.

</details>

### Exercise 3.2 — Enforce a boundary with `internal/`

**Task:** Prove the `internal/` rule: try to import `internal/customer` from a
sibling throwaway module and observe the compile error; then move the importer
under the module root and watch it compile.

<details>
<summary>Reference solution</summary>

Importing `github.com/acme/pulse/internal/customer` from a module *not* rooted at
`github.com/acme/pulse` fails with:

```
use of internal package github.com/acme/pulse/internal/customer not allowed
```

Move the consumer to `github.com/acme/pulse/cmd/...` (same root) and it compiles.
**Reasoning:** `internal/` is compiler-enforced encapsulation — your public API is
exactly what's *outside* `internal/`, which for a service binary is nothing. That's
the point.

</details>

---

## 4. Fill-in-the-blank

Complete the config loader (the shared-kernel pattern).

```go
package config

type Config struct {
    HTTPAddr   string
    GRPCAddr   string
    PostgresDSN string
    MongoURI    string
    NATSURL     string
    LogLevel    string
}

func Load() Config {
    return Config{
        HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
        GRPCAddr:    getenv("GRPC_ADDR", ":9090"),
        PostgresDSN: getenv("PULSE_PG_DSN", ""),
        MongoURI:    getenv("PULSE_MONGO_URI", ""),
        NATSURL:     getenv("PULSE_NATS_URL", "nats://localhost:4222"),
        LogLevel:    getenv("LOG_LEVEL", "info"),
    }
}

func getenv(key, def string) string {
    if v := os./* ___1 ___ */(key); v != "" {
        return v
    }
    return /* ___2 ___ */
}
```

<details>
<summary>Answers</summary>

```go
if v := os.Getenv(key); v != "" { return v } // 1
return def                                    // 2
```

(In M13 we replace this with a typed config lib; the shape stays the same.)

</details>

---

## 5. Implement it yourself

**Problem:** Migrate your M1–M9 code into the unified `pulse/` layout:
- Move each module's domain into `internal/<domain>/`.
- Extract connection constructors into `internal/platform/{postgres,mongo,nats}`.
- Make the three `cmd/` mains wire real dependencies from `config`.
- Get `go build ./...` and `go test ./...` green across the whole repo.
- Write a `Makefile` with `build`, `test`, `lint`, `migrate`, `run-*` targets.

This *is* the start of the capstone repo; M11–M14 build on it.

**Curated resources:**
- *Standard Go Project Layout* (community) — https://github.com/golang-standards/project-layout
- *Style guideline for Go packages* — https://rakyll.org/style-packages/
- Russ Cox, *Go modules reference* — https://go.dev/ref/mod
- Go workspaces tutorial — https://go.dev/doc/tutorial/workspaces
- Ben Johnson, *Standard Package Layout* — https://www.gobeyond.dev/standard-package-layout/
- *Organising Database Access* (Alex Edwards) — https://www.alexedwards.net/blog/organising-database-access

**Hints:** start by moving one domain (customer) end-to-end; get it green; repeat.
Resist a `models` package — keep types next to the logic that owns them.

---

## 6. Capstone contribution

Pulse now has its real home: a single-module monorepo, domain-organized
`internal/` packages, a shared `platform` kernel, thin `cmd/` binaries, and
contracts under `api/`. Every later module commits into this structure.

---

## 7. Self-check — you should now be able to…

- [ ] Lay out a Go service repo and justify `cmd/`, `internal/`, `api/`.
- [ ] Explain and demonstrate the `internal/` import rule.
- [ ] Organize by domain/feature and keep dependencies pointing inward.
- [ ] Decide monorepo vs multi-repo and use `go.work` when splitting modules.
- [ ] Keep `main` thin with manual dependency injection.
