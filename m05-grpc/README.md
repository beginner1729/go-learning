# M05 — gRPC for Internal Services (exercise)

You implement the code; the tests tell you exactly what to build. Nothing here
compiles until you write the real packages — that's the point.

## Layout

| Path | What it is |
|---|---|
| `interceptor/` | **You write here.** Stub for the auth/log/recover unary interceptors (§2.6 & §4). No tests of its own — exercised via `profile/`. |
| `profile/` | **You write here.** Stub + the real tests (Exercises 3.1 & 3.2). |
| `cmd/server/` | The gRPC server wired to *your* packages. Won't build until they exist. |
| `proto/profile/v1/` | **Generated protobuf code — do not edit.** Shared verbatim by both your packages and the answer key (`*.pb.go` from `profile.proto`). |
| `solution-interceptor/`, `solution-profile/` | Answer key. Builds & passes on its own. |
| `solution-cmd/server/` | The working server wired to the answer key. |

Lesson + reasoning: `../M05-grpc.md`.

## Workflow

1. Read the task comments at the top of `interceptor/interceptor.go` and
   `profile/profile.go`.
2. Implement until the tests go green:
   ```sh
   go test ./interceptor/... ./profile/...
   ```
   (Before you start they fail to *compile* — the `undefined:` errors list the
   API the tests and the server expect.)
3. Run the server:
   ```sh
   go run ./cmd/server
   ```
4. Stuck? Compare with the reference and run it:
   ```sh
   go run ./solution-cmd/server
   go test ./solution-interceptor/... ./solution-profile/...
   ```

Note: `proto/profile/v1/` is generated (`protoc` output, committed) and imported
unchanged as `cxm/m05/proto/profile/v1` by both your code and the solution —
never rename or edit it.

Tip: `go test ./...` runs everything — your packages (red until done) plus the
solution (always green) as a reference baseline.
