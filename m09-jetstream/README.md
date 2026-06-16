# M09 — JetStream & Event-Driven Patterns (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `events/` | **You write here.** Stub for the shared subjects + `CustomerEvent`. |
| `stream/` | **You write here.** Stub + the real tests (Exercises 3.1, 3.2, §4 DLQ, §5). |
| `cmd/demo/` | End-to-end demo wired to *your* packages. Won't build until they exist. |
| `solution-events/`, `solution-stream/` | Answer key. Builds & passes on its own. |
| `solution-cmd/demo/` | The working demo wired to the answer key. |

The `stream` tests are **JetStream integration tests**: they need a running
NATS JetStream and the `PULSE_NATS_URL` env var. When `PULSE_NATS_URL` is unset
they **skip cleanly** — that's expected, not a failure.

Lesson + reasoning: `../M09-jetstream.md`.

## Workflow

1. Read the task comments at the top of `events/events.go` and `stream/stream.go`.
2. Implement until the tests go green:
   ```sh
   go test ./events/... ./stream/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.
   Without `PULSE_NATS_URL` the integration tests skip rather than run.)
3. Run the end-to-end demo (needs JetStream):
   ```sh
   PULSE_NATS_URL=nats://localhost:4222 go run ./cmd/demo
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   PULSE_NATS_URL=nats://localhost:4222 go run ./solution-cmd/demo
   go test ./solution-events/... ./solution-stream/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (green, or skipped without `PULSE_NATS_URL`) as a reference baseline.
