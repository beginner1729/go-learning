# M08 — NATS Core: pub/sub, queue groups, request-reply (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

The tests are NATS **integration** tests. They **skip cleanly** when
`PULSE_NATS_URL` is unset, so they're green-by-skip out of the box. To actually
exercise them, run a local NATS server and export the URL (see below).

## Layout

| Path | What it is |
|---|---|
| `events/` | **You write here.** Stub + the event taxonomy used by the bus tests (§2.2/§3). |
| `bus/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2). Tests need `PULSE_NATS_URL`; otherwise they skip. |
| `cmd/demo/` | End-to-end demo wired to *your* packages. Won't build until they exist. |
| `solution-events/`, `solution-bus/` | Answer key. Builds & passes (or skips) on its own. |
| `solution-cmd/demo/` | The working demo wired to the answer key. |

Lesson + reasoning: `../M08-nats-core.md`.

## Setup (for non-skipped tests + the demo)

```sh
docker run -d --name pulse-nats -p 4222:4222 nats:2-alpine -js
export PULSE_NATS_URL="nats://localhost:4222"
```

## Workflow

1. Read the task comments at the top of `events/events.go` and `bus/bus.go`.
2. Implement until the tests go green:
   ```sh
   go test ./bus/... ./events/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.
   With `PULSE_NATS_URL` unset they *skip*; set it to run them for real.)
3. Run the end-to-end demo:
   ```sh
   go run ./cmd/demo
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/demo
   go test ./solution-bus/... ./solution-events/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green, or green-by-skip) as a reference baseline.
