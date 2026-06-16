# M04 — REST APIs with `net/http` + `chi` (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `customer/` | **You write here.** Stub for the trimmed domain + thin service. No tests of its own — exercised through `httpapi`. |
| `httpapi/` | **You write here.** Stub + the real handler tests (Exercises 3.1, 3.2, §4 & §5). |
| `cmd/api/` | The runnable server wired to *your* packages. Won't build until they exist. |
| `solution-customer/`, `solution-httpapi/` | Answer key. Builds & passes on its own. |
| `solution-cmd/api/` | The working server wired to the answer key. |

Lesson + reasoning: `../M04-rest-api.md`.

## Workflow

1. Read the task comments at the top of `customer/customer.go` and
   `httpapi/httpapi.go`.
2. Implement until the tests go green:
   ```sh
   go test ./customer/... ./httpapi/...
   ```
   (Before you start they fail to *compile* — the undefined symbols list the API.)
3. Run the server and poke it:
   ```sh
   go run ./cmd/api
   # in another shell:
   curl -s -H 'Authorization: Bearer dev-token' \
     -d '{"email":"ada@pulse.dev","name":"Ada"}' localhost:8080/v1/customers
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/api
   go test ./solution-customer/... ./solution-httpapi/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green) as a reference baseline.
