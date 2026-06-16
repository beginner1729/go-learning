# M02 — Generics & Idiomatic Project Idioms (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `genx/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2). |
| `store/` | **You write here.** Stub + the real tests (§4 fill-in-the-blank). |
| `config/` | **You write here.** Stub + the real tests (§2.3 functional options). |
| `cmd/demo/` | End-to-end demo wired to *your* packages. Won't build until they exist. |
| `solution-genx/`, `solution-store/`, `solution-config/` | Answer key. Builds & passes on its own. |
| `solution-cmd/demo/` | The working demo wired to the answer key. |

Lesson + reasoning: `../M02-generics-and-idioms.md`.

## Workflow

1. Read the task comments at the top of `genx/genx.go`, `store/store.go`,
   `config/config.go`.
2. Implement until the tests go green:
   ```sh
   go test ./genx/... ./store/... ./config/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.)
   The store is concurrency-safe; check it with `go test -race ./store/...`.
3. Run the end-to-end demo:
   ```sh
   go run ./cmd/demo
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/demo
   go test ./solution-genx/... ./solution-store/... ./solution-config/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green) as a reference baseline.
