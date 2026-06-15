# Go for Production Backends — A CXM-Themed Intensive

> From "I know basic Go syntax" → "I can design, build, and maintain
> enterprise-grade Go backend services independently."

This course is **topic-based modules that progressively assemble one capstone
service**: a Customer Experience Management (CXM) backend called **Pulse**.
Every module tells you exactly which piece of Pulse it contributes.

---

## The Capstone: "Pulse" — a CXM backend

Pulse manages **customers**, their **profiles**, the **journeys/campaigns** they
move through, the **events** they generate, and the **messages/notifications**
sent to them. It is the canonical shape of a modern enterprise backend, so every
pattern transfers to any domain.

### Capstone scope (what you will have built by the end)

A multi-service Go system:

| Concern | Choice | Why |
|---|---|---|
| External API | **REST** (`net/http` + `chi`) | Public clients, browsers, partners |
| Internal API | **gRPC** | Typed, fast service-to-service calls |
| Relational data | **PostgreSQL** (`pgx` + `sqlc` + `goose`) | Customers, campaigns, journeys — structured, transactional |
| Document data | **MongoDB** | Customer profiles + event records — flexible schema |
| Messaging | **NATS Core + JetStream** | Event-driven notification flows, durable delivery |
| Repo/ops | modules + `internal/`, table tests, `golangci-lint`, GitHub Actions, multi-stage Docker, Prometheus, OpenTelemetry, structured logging, graceful shutdown | Enterprise maintainability |

### Service decomposition

```
                         ┌─────────────────┐
   REST (public) ──────► │  customer-api   │ ── gRPC ──┐
                         └────────┬────────┘           │
                                  │ events             ▼
                                  ▼              ┌─────────────┐
                         ┌─────────────────┐     │ profile-svc │  (MongoDB)
                         │  NATS JetStream │     └─────────────┘
                         └────────┬────────┘
                                  │ subscribe
                                  ▼
                         ┌─────────────────┐
                         │ notification-svc│ ── sends messages, records delivery
                         └─────────────────┘
   PostgreSQL  ◄── customers / campaigns / journeys
   MongoDB     ◄── profiles / event log
```

### Core feature list

1. CRUD customers + campaigns + journeys (Postgres, REST).
2. Rich customer profiles + an append-only event log (Mongo).
3. Emit a domain event on every meaningful change (e.g. `customer.created`,
   `journey.step.entered`) onto JetStream.
4. A notification service that consumes events, decides what to send,
   sends it (simulated email/SMS), and records delivery — idempotently,
   with retries and a dead-letter path.
5. gRPC between `customer-api` and `profile-svc`.
6. AuthN/AuthZ, validation, PII handling.
7. Full ops: tests, lint, CI, Docker, metrics, traces, logs, graceful shutdown.

---

## Module list

### Week 1 — Go language depth (the foundation everything else rests on)
- **M1. Interfaces, composition & idiomatic error handling** → defines Pulse's
  domain model + error taxonomy.
- **M2. Generics & idiomatic project idioms** → reusable helpers (result sets,
  paginated responses, typed IDs).
- **M3. Concurrency in depth** → the notification dispatcher: worker pools,
  fan-in/fan-out, `context`, graceful cancellation.

### Week 2 — Backend servers
- **M4. REST with `net/http` + `chi`** → the public `customer-api`.
- **M5. gRPC** → internal `customer-api` ⇄ `profile-svc`.

### Week 3 — Databases
- **M6. PostgreSQL** (`pgx`, `sqlc`, `goose`, repository pattern, tx, testing)
  → persistence for customers/campaigns/journeys.
- **M7. MongoDB** → profiles + event log; relational-vs-document decision making.

### Week 4 — Messaging / event-driven
- **M8. NATS Core** → pub/sub, queue groups, request-reply between services.
- **M9. JetStream & event-driven patterns** → durable streams, consumers,
  idempotency, retries, DLQ → the notification flow.

### Week 5 — Enterprise repo & ops
- **M10. Project structure & modules** → unify everything into a clean repo.
- **M11. Testing & quality gates** → table tests, integration tests with
  testcontainers, mocks, coverage, `golangci-lint`, pre-commit.
- **M12. CI/CD & containerization** → GitHub Actions, multi-stage Docker,
  releases, k8s basics.
- **M13. Observability & config** → structured logs, Prometheus metrics,
  OpenTelemetry traces, config/secrets, graceful shutdown.

### Week 6 — Capstone
- **M14. Assemble, harden, and ship Pulse** → wire all services end-to-end,
  run the full event flow, write the runbook.

---

## How each module is structured

1. **Learning objectives**
2. **Concept explanations** with idiomatic Go + the *why* + pitfalls/anti-patterns
3. **Hands-on exercises** with full reference solutions + reasoning
4. **Fill-in-the-blank** exercises (with answers)
5. **"Implement it yourself"** open problems + curated resources
6. **Capstone contribution**
7. **Self-check** ("you should now be able to…")

## Tooling choices (justified once, used throughout)

- **`chi`** for routing: stdlib-compatible (`http.Handler`), tiny, idiomatic — no
  framework lock-in. (Std `net/http` 1.22+ routing is also taught.)
- **`pgx`** over `database/sql`: best-in-class Postgres driver, native types,
  pooling. **`sqlc`** generates type-safe Go from SQL — no ORM magic. **`goose`**
  for migrations: simple, file-based.
- **`grpc-go`** + **`buf`** for protobuf: `buf` is the modern build/lint/breaking-
  change tool for protos.
- **`nats.go`**: the official NATS client; JetStream built in.
- **`slog`** (stdlib, Go 1.21+) for logging; **`prometheus/client_golang`** for
  metrics; **`go.opentelemetry.io/otel`** for traces.
- **`testify`** for assertions, **`testcontainers-go`** for real-DB integration tests.

Go version: **1.22+** assumed (uses `slog`, enhanced `net/http` routing, range-over-func is optional).

---

## Suggested daily cadence (intensive)

- **Morning:** read concepts + do hands-on exercises.
- **Afternoon:** fill-in-the-blanks + start the "implement it yourself" task.
- **End of day:** integrate the day's code into the capstone repo; run tests/lint.
- **Weekend (optional):** refactor, re-read pitfalls, extend the capstone.

Proceed module by module. Each builds on the last. Let's start with **M1**.
