# M07 — MongoDB: Documents, Indexes & Aggregation (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `mongodb/` | **You write here.** Stub for `Connect` (§4 fill-in-the-blank). |
| `profile/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2, §5 aggregation). |
| `cmd/demo/` | End-to-end demo wired to *your* packages. Won't build until they exist. |
| `solution-mongodb/`, `solution-profile/` | Answer key. Builds & passes on its own. |
| `solution-cmd/demo/` | The working demo wired to the answer key. |

The integration tests need a running MongoDB: set `PULSE_MONGO_URI` (e.g.
`mongodb://localhost:27017`). With it unset they **skip cleanly** — that's
expected, not a failure.

Lesson + reasoning: `../M07-mongodb.md`.

## Workflow

1. Read the task comments at the top of `mongodb/mongodb.go` and
   `profile/profile.go`.
2. Implement until the tests go green:
   ```sh
   go test ./mongodb/... ./profile/...
   ```
   (Before you start they fail to *compile* — undefined symbols list the API.
   Without `PULSE_MONGO_URI` they compile and skip; set it to actually run.)
3. Run the end-to-end demo against your Mongo:
   ```sh
   PULSE_MONGO_URI=mongodb://localhost:27017 go run ./cmd/demo
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   PULSE_MONGO_URI=mongodb://localhost:27017 go run ./solution-cmd/demo
   go test ./solution-mongodb/... ./solution-profile/...
   ```

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green) as a reference baseline.
