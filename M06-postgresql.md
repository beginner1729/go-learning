# Module 6 — PostgreSQL with `pgx`, migrations & the repository pattern

> **Capstone contribution:** durable storage for Pulse's relational core —
> **customers** and **campaigns**. Connection pooling, file-based migrations
> (`goose`), transactions, and a Postgres-backed `Repository` that drops in behind
> the M4 service interface unchanged.

---

## 0. Setup & run

```bash
cd go-cxm-course/m06-postgresql

# Start Postgres (any one works):
docker run -d --name pulse-pg -e POSTGRES_PASSWORD=pulse -e POSTGRES_USER=pulse \
  -e POSTGRES_DB=pulse -p 5433:5432 postgres:16-alpine

export PULSE_PG_DSN="postgres://pulse:pulse@localhost:5433/pulse?sslmode=disable"

go mod tidy
go vet ./...
go test ./...          # integration tests SKIP automatically if PULSE_PG_DSN is unset
go run ./cmd/migrate   # applies migrations
go run ./cmd/demo      # creates + queries customers against real Postgres
```

Layout:

```
m06-postgresql/
  go.mod                 # module cxm/m06
  migrations/            # goose SQL migrations (0001_*.sql ...)
  customer/              # domain + Postgres Repository (pgxpool)
  db/                    # pool construction + migration runner
  cmd/migrate/           # apply migrations
  cmd/demo/              # end-to-end against Postgres
```

---

## 1. Learning objectives

By the end you will be able to:

- Connect with **`pgxpool`** and configure a connection pool sensibly.
- Manage schema with **versioned migrations** (`goose`).
- Implement the **repository pattern** with type-safe queries and proper error mapping.
- Use **transactions** correctly (the `BeginFunc`/closure idiom, rollback on error).
- Write **integration tests against a real database** that skip cleanly when none is configured.
- Decide relational-vs-document storage (Postgres here, Mongo in M7).

---

## 2. Concepts

### 2.1 `database/sql` vs `pgx` — and `sqlc`

- **`database/sql`** is the stdlib abstraction; works with any driver but lowest-
  common-denominator (no native Postgres types, slower).
- **`pgx`** is the best-in-class Postgres driver. Use **`pgxpool`** directly
  (not through `database/sql`) to get native types (arrays, JSONB, `numeric`),
  better performance, and `pgx.Rows` scanning helpers.
- **`sqlc`** generates type-safe Go from your SQL at build time — you write SQL,
  it writes the Go structs and methods. No ORM, no reflection. It's the
  enterprise sweet spot. We hand-write the query layer here so the module runs
  without the `sqlc` binary, and show the `sqlc.yaml` so you can adopt it.

> **Why not an ORM (GORM)?** ORMs hide SQL, generate surprising queries, and make
> performance debugging hard. The Go community strongly favors SQL-first
> (`sqlc`/`pgx`). You stay in control of every query.

### 2.2 The connection pool

```go
cfg, err := pgxpool.ParseConfig(dsn)
cfg.MaxConns = 10            // size to your DB's max_connections / replica count
cfg.MaxConnIdleTime = 5 * time.Minute
cfg.HealthCheckPeriod = time.Minute
pool, err := pgxpool.NewWithConfig(ctx, cfg)
defer pool.Close()
if err := pool.Ping(ctx); err != nil { ... } // fail fast at startup
```

Pitfalls: don't open a pool per request (open once, share it). Don't set
`MaxConns` higher than the DB allows across all instances. Always `defer rows.Close()`.

### 2.3 Migrations with `goose`

Migrations are versioned, ordered SQL files checked into the repo. Each has an
`Up` and `Down`:

```sql
-- migrations/0001_customers.sql
-- +goose Up
CREATE TABLE customers (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE customers;
```

Run them from code (embedded via `embed.FS`) or the `goose` CLI. Rules: migrations
are **append-only and immutable** once merged — never edit a shipped migration;
add a new one. The `UNIQUE(email)` constraint is what surfaces our `ErrConflict`.

### 2.4 The repository

The repo implements the same interface the service depends on (M1's lesson pays
off — the HTTP handler doesn't change):

```go
func (r *PostgresRepo) ByID(ctx context.Context, id customer.ID) (customer.Customer, error) {
    const q = `SELECT id, email, name, created_at, updated_at FROM customers WHERE id = $1`
    row := r.pool.QueryRow(ctx, q, string(id))
    var c customer.Customer
    err := row.Scan(&c.ID, &c.Email, &c.Name, &c.CreatedAt, &c.UpdatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return customer.Customer{}, customer.ErrNotFound // map driver error to domain
    }
    if err != nil {
        return customer.Customer{}, fmt.Errorf("byID: %w", err)
    }
    return c, nil
}
```

**Error mapping is the repo's job:** translate `pgx.ErrNoRows` → `ErrNotFound`, and
a unique-violation (`SQLSTATE 23505`) → `ErrConflict`. Upstream layers stay
driver-agnostic.

```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) && pgErr.Code == "23505" {
    return customer.Customer{}, customer.ErrConflict
}
```

### 2.5 Transactions — the closure idiom

Wrap multi-statement work in a transaction with guaranteed rollback on any error
or panic:

```go
func (r *PostgresRepo) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx) // no-op if already committed; safety net on panic/early return
    if err := fn(tx); err != nil {
        return err
    }
    return tx.Commit(ctx)
}
```

The `defer Rollback` after a successful `Commit` is harmless (it returns
`ErrTxClosed`), and it guarantees you never leak an open transaction. Always set
appropriate isolation if you need it (`pgx.TxOptions{IsoLevel: pgx.Serializable}`).

#### Pitfalls

- **Forgetting to map driver errors** → leaks `pgx` types into your domain.
- **`QueryRow` then ignoring `Scan`'s error** — `ErrNoRows` only surfaces on `Scan`.
- **String-building SQL with user input** → SQL injection. Always use `$1, $2`
  placeholders (pgx never interpolates).
- **Not closing `rows`** → connection leak; pool exhaustion under load.
- **Long-held transactions** → lock contention; keep them short.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — `Create` with conflict mapping

**Task:** Implement `Create` inserting a customer, generating an ID, mapping a
unique-violation to `ErrConflict`, and returning the row with DB-populated timestamps.

<details>
<summary>Reference solution</summary>

```go
func (r *PostgresRepo) Create(ctx context.Context, c customer.Customer) (customer.Customer, error) {
    id := "cus_" + r.ids.New() // some ID generator
    const q = `
        INSERT INTO customers (id, email, name)
        VALUES ($1, $2, $3)
        RETURNING id, email, name, created_at, updated_at`
    row := r.pool.QueryRow(ctx, q, id, string(c.Email), c.Name)
    var out customer.Customer
    err := row.Scan(&out.ID, &out.Email, &out.Name, &out.CreatedAt, &out.UpdatedAt)
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" {
        return customer.Customer{}, customer.ErrConflict
    }
    if err != nil {
        return customer.Customer{}, fmt.Errorf("create customer: %w", err)
    }
    return out, nil
}
```

**Reasoning:** `RETURNING` gives the DB-generated `created_at/updated_at` in one
round-trip. Unique-violation code `23505` becomes the domain `ErrConflict` the
HTTP layer already maps to 409.

</details>

### Exercise 3.2 — Paginated `List` with `LIMIT/OFFSET`

**Task:** Implement `List(ctx, limit, offset)` returning customers ordered by
`created_at DESC`, scanning all rows, closing `rows`.

<details>
<summary>Reference solution</summary>

```go
func (r *PostgresRepo) List(ctx context.Context, limit, offset int) ([]customer.Customer, error) {
    const q = `
        SELECT id, email, name, created_at, updated_at
        FROM customers ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`
    rows, err := r.pool.Query(ctx, q, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("list query: %w", err)
    }
    defer rows.Close()

    var out []customer.Customer
    for rows.Next() {
        var c customer.Customer
        if err := rows.Scan(&c.ID, &c.Email, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
            return nil, fmt.Errorf("scan: %w", err)
        }
        out = append(out, c)
    }
    return out, rows.Err() // check iteration error AFTER the loop
}
```

**Reasoning:** `rows.Err()` after the loop catches errors that ended iteration
early — a commonly forgotten check. `pgx.CollectRows` + `RowToStructByName` can
replace this boilerplate; we show the explicit form first.

</details>

---

## 4. Fill-in-the-blank

Complete the pool constructor and the not-found mapping.

```go
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("parse dsn: %w", err)
    }
    cfg.MaxConns = 10
    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return nil, err
    }
    if err := pool./* ___1: verify connectivity ___ */(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping: %w", err)
    }
    return pool, nil
}

func mapErr(err error) error {
    if errors.Is(err, /* ___2: pgx no-rows sentinel ___ */) {
        return customer.ErrNotFound
    }
    return err
}
```

<details>
<summary>Answers</summary>

```go
if err := pool.Ping(ctx); err != nil { ... } // 1
...
if errors.Is(err, pgx.ErrNoRows) { ... }     // 2
```

</details>

---

## 5. Implement it yourself

**Problem:** Add the **campaigns** table and repository, plus a **transactional**
operation that exercises real tx semantics:
- `campaigns(id, name, status, created_at)` migration.
- A `EnrollCustomer(ctx, customerID, campaignID)` that, **in one transaction**,
  inserts an enrollment row and increments a `campaigns.enrolled_count` — rolling
  back if either fails (e.g. unknown customer FK).
- Adopt **`sqlc`**: write `sqlc.yaml`, move one query to a `.sql` file, generate,
  and compare ergonomics to the hand-written version.
- Add a **testcontainers-go** integration test that boots Postgres in-process
  (so tests need no external setup) — preview of M11.

**Curated resources:**
- `pgx` — https://github.com/jackc/pgx  ·  pool: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool
- `sqlc` — https://docs.sqlc.dev/  ·  with pgx: https://docs.sqlc.dev/en/latest/reference/database-drivers.html
- `goose` — https://github.com/pressly/goose
- `testcontainers-go` Postgres module — https://golang.testcontainers.org/modules/postgresql/
- Postgres error codes (SQLSTATE) — https://www.postgresql.org/docs/current/errcodes-appendix.html
- *Organising Database Access in Go* (Alex Edwards) — https://www.alexedwards.net/blog/organising-database-access

**Hints:** FK violation is SQLSTATE `23503`; map it to a validation/`ErrNotFound`.
Use the `inTx` helper; the closure does both writes and returns the first error.

---

## 6. Capstone contribution

Pulse's customers now persist in Postgres with migrations, pooling, and tx
support — behind the *same* `Repository` interface, so M4's HTTP handler is
untouched. Campaigns join the schema. M7 adds MongoDB for the data that doesn't
fit a relational table.

---

## 7. Self-check — you should now be able to…

- [ ] Build and configure a `pgxpool` and fail fast on a bad connection.
- [ ] Write and run versioned migrations; explain why they're immutable once merged.
- [ ] Implement a repository that maps driver errors to domain errors.
- [ ] Use the transaction closure idiom with guaranteed rollback.
- [ ] Write DB integration tests that skip cleanly without a database.
