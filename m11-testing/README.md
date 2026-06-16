# M11 — Testing & Quality Gates (exercise)

You implement the code; the tests tell you exactly what to build. The `notify`
package won't compile until you write the real declarations — that's the point.

## Layout

| Path | What it is |
|---|---|
| `notify/` | **You write here.** Stub + the real tests (Exercises 3.1, 3.2 & §4). |
| `solution-notify/` | Answer key. Builds & passes on its own. |

Lesson + reasoning: `../M11-testing.md`.

## Workflow

1. Read the task comment at the top of `notify/notify.go`.
2. Implement until the tests go green:
   ```sh
   go test ./notify/...
   ```
   (Before you start they fail to *compile* — `undefined:` symbols list the API.)
3. Stuck? Compare with the reference and run it:
   ```sh
   go test ./solution-notify/...
   ```

Tip: `go test ./...` runs everything — your package (red until done) plus the
solution (always green) as a reference baseline.
