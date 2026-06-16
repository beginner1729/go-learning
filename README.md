# Go for Production Backends — A CXM-Themed Intensive

An advanced, self-paced Go course that takes you from "I know basic syntax" to
building and maintaining **enterprise-grade Go backend services**, themed around a
Customer Experience Management (CXM) platform called **Pulse**.

> Start with **[00-ROADMAP.md](00-ROADMAP.md)** for the full plan, capstone
> definition, and week-by-week sequencing.

## How to use this course

Each module is **two things**:
1. A lesson: `Mxx-*.md` — objectives, concepts + idioms + pitfalls, hands-on
   exercises with solutions, fill-in-the-blanks, an open "implement it yourself"
   task with curated links, the capstone contribution, and a self-check.
2. A **self-contained runnable Go module**: `mxx-*/` — compiles, tests, and runs on
   its own (`go test ./...`, `go run ./cmd/...`).

Work the lesson, do the exercises, then run/extend the code.

## Modules

| # | Lesson | Code | Needs |
|---|--------|------|-------|
| 0 | [Getting started: build a small Go project](M00-getting-started.md) | `m00-getting-started/` | — |
| 1 | [Interfaces & errors](M01-interfaces-and-errors.md) | `m01-interfaces-and-errors/` | — |
| 2 | [Generics & idioms](M02-generics-and-idioms.md) | `m02-generics-and-idioms/` | — |
| 3 | [Concurrency](M03-concurrency.md) | `m03-concurrency/` | — |
| 4 | [REST API](M04-rest-api.md) | `m04-rest-api/` | — |
| 5 | [gRPC](M05-grpc.md) | `m05-grpc/` | protoc (codegen committed) |
| 6 | [PostgreSQL](M06-postgresql.md) | `m06-postgresql/` | Docker (Postgres) |
| 7 | [MongoDB](M07-mongodb.md) | `m07-mongodb/` | Docker (Mongo) |
| 8 | [NATS Core](M08-nats-core.md) | `m08-nats-core/` | Docker (NATS) |
| 9 | [JetStream](M09-jetstream.md) | `m09-jetstream/` | Docker (NATS -js) |
| 10 | [Project structure](M10-project-structure.md) | `m10-project-structure/pulse/` | — |
| 11 | [Testing & quality](M11-testing.md) | `m11-testing/` | — |
| 12 | [CI/CD & Docker](M12-cicd-containerization.md) | `m12-cicd/` | Docker |
| 13 | [Observability](M13-observability.md) | `m13-observability/` | — |
| 14 | [Capstone](M14-capstone.md) | `m14-capstone/` | Docker (NATS -js) |

## Infrastructure (for the DB/messaging modules)

```bash
# Postgres (M6)
docker run -d --name pulse-pg -e POSTGRES_PASSWORD=pulse -e POSTGRES_USER=pulse \
  -e POSTGRES_DB=pulse -p 5433:5432 postgres:16-alpine
export PULSE_PG_DSN="postgres://pulse:pulse@localhost:5433/pulse?sslmode=disable"

# Mongo (M7)
docker run -d --name pulse-mongo -p 27018:27017 mongo:7
export PULSE_MONGO_URI="mongodb://localhost:27018"

# NATS + JetStream (M8, M9, M14)
docker run -d --name pulse-nats -p 4222:4222 nats:2-alpine -js
export PULSE_NATS_URL="nats://localhost:4222"
```

All integration tests **skip cleanly** when the relevant env var is unset, so
`go test ./...` works everywhere; set the vars to exercise the real-infra paths.

## What you'll have built

By M14: the **Pulse** CXM backend — REST + gRPC APIs, Postgres + Mongo storage,
a durable event-driven notification flow over NATS/JetStream, and a full
enterprise repo (structure, tests, lint, CI/CD, Docker, k8s, observability,
graceful shutdown). The flagship flow — `POST /v1/customers` → durable event →
idempotent consumer → welcome notification — runs and is tested end-to-end.

Go version: **1.24** (uses `slog`, enhanced `net/http` routing, modern `jetstream` API).
