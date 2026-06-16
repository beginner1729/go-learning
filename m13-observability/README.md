# M13 — Observability & Config (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real `obs` package — that's the point.

## Layout

| Path | What it is |
|---|---|
| `obs/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2). |
| `cmd/server/` | Instrumented HTTP server wired to *your* `obs` package. Won't build until it exists. |
| `solution-obs/` | Answer key. Builds & passes on its own. |
| `solution-cmd/server/` | The working server wired to the answer key. |

Lesson + reasoning: `../M13-observability.md`.

## Workflow

1. Read the task comment at the top of `obs/obs.go`.
2. Implement until the tests go green:
   ```sh
   go test ./obs/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.)
3. Run the instrumented server:
   ```sh
   go run ./cmd/server          # :8080 (app) + /metrics; traces to stdout
   # curl localhost:8080/hello?name=Ada
   # curl -s localhost:8080/metrics | grep http_requests
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/server
   go test ./solution-obs/...
   ```

Tip: `go test ./...` runs everything — your package (red until done) plus the
solution (always green) as a reference baseline.
