# M06 — PostgreSQL with `pgx`, migrations & the repository pattern (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

The integration tests need a real Postgres. They **skip cleanly** when
`PULSE_PG_DSN` is unset, so `go test` is green-by-skip without a database. Set
the DSN (see the lesson §0) to actually exercise them.

## Layout

| Path | What it is |
|---|---|
| `customer/` | **You write here.** Stub + the real tests (domain, Postgres repo, transactions). |
| `db/` | **You write here.** Stub (pool constructor + migration runner). |
| `cmd/migrate/`, `cmd/demo/` | CLIs wired to *your* packages. Won't build until they exist. |
| `migrations/` | **Shared infrastructure — do not touch.** Embedded goose SQL, imported unchanged by both your code and the solution. |
| `solution-customer/`, `solution-db/` | Answer key. Builds & passes (or skips) on its own. |
| `solution-cmd/migrate/`, `solution-cmd/demo/` | The working CLIs wired to the answer key. |

Lesson + reasoning: `../M06-postgresql.md`.

## Workflow

1. Read the task comments at the top of `customer/customer.go` and `db/db.go`.
2. Implement until the tests go green (or skip cleanly):
   ```sh
   go test ./customer/... ./db/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.
   Once they compile, they SKIP unless `PULSE_PG_DSN` is set.)
3. To run against a real database, start Postgres and export the DSN (lesson §0):
   ```sh
   export PULSE_PG_DSN="postgres://pulse:pulse@localhost:5433/pulse?sslmode=disable"
   go run ./cmd/migrate   # apply migrations
   go run ./cmd/demo      # create + query customers end-to-end
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/migrate
   go run ./solution-cmd/demo
   go test ./solution-customer/... ./solution-db/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green, skipping without a DB) as a reference baseline.
