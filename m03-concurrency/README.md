# M03 — Concurrency in Depth (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `pipeline/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2). |
| `dispatcher/` | **You write here.** Stub + the real tests (§5 "Implement it yourself"). |
| `cmd/demo/` | End-to-end demo wired to *your* dispatcher. Won't build until it exists. |
| `solution-pipeline/`, `solution-dispatcher/` | Answer key. Builds & passes on its own. |
| `solution-cmd/demo/` | The working demo wired to the answer key. |

Lesson + reasoning: `../M03-concurrency.md`.

## Workflow

1. Read the task comments at the top of `pipeline/pipeline.go` and
   `dispatcher/dispatcher.go`.
2. Implement until the tests go green:
   ```sh
   go test ./pipeline/... ./dispatcher/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.)
   Concurrency code earns its keep under the race detector:
   ```sh
   go test -race ./pipeline/... ./dispatcher/...
   ```
3. Run the end-to-end demo:
   ```sh
   go run ./cmd/demo
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/demo
   go test ./solution-pipeline/... ./solution-dispatcher/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green) as a reference baseline.
