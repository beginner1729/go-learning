# M01 — Interfaces, Composition & Idiomatic Errors (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `customer/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2). |
| `errorx/` | **You write here.** Stub + the real tests (§5 "Implement it yourself"). |
| `cmd/demo/` | End-to-end demo wired to *your* packages. Won't build until they exist. |
| `solution-customer/`, `solution-errorx/` | Answer key. Builds & passes on its own. |
| `solution-cmd/demo/` | The working demo wired to the answer key. |

Lesson + reasoning: `../M01-interfaces-and-errors.md`.

## Workflow

1. Read the task comments at the top of `customer/customer.go`,
   `customer/validate.go`, `errorx/errorx.go`.
2. Implement until the tests go green:
   ```sh
   go test ./customer/... ./errorx/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.)
3. Run the end-to-end demo:
   ```sh
   go run ./cmd/demo
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/demo
   go test ./solution-customer/... ./solution-errorx/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green) as a reference baseline.
