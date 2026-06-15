# Module 14 — Capstone: Assemble, Harden & Ship Pulse

> **This module ties M1–M13 into the running Pulse system** and proves the full
> CXM flow end-to-end: a REST call creates a customer, emits a durable event, and
> a notification service consumes it and "sends" a welcome message — observably,
> idempotently, gracefully.

---

## 0. Setup & run

```bash
cd go-cxm-course/m14-capstone
docker run -d --name pulse-nats -p 4222:4222 nats:2-alpine -js   # if not already running
export PULSE_NATS_URL="nats://localhost:4222"

go mod tidy
go vet ./...
go test ./...          # end-to-end test: REST create -> JetStream -> notification sent
go run ./cmd/pulse     # runs the whole vertical slice in one process and prints the flow
```

Layout (the integrated vertical slice — the full multi-service repo is the
"implement it yourself" deliverable):

```
m14-capstone/
  go.mod
  domain/               # Customer + CustomerEvent
  api/                  # REST handler: POST /v1/customers -> store + publish event
  events/               # JetStream publish + durable, idempotent consumer (M9)
  notify/               # dispatcher-backed sender (M3) wired to the consumer
  cmd/pulse/            # boots API + consumer in one process; drives the flow
  pulse_e2e_test.go     # asserts the event flow works end-to-end
```

---

## 1. The capstone definition (recap)

**Pulse** — a Customer Experience Management backend demonstrating the full stack:

| Capability | Built in | Status |
|---|---|---|
| Idiomatic domain + error taxonomy | M1 | ✅ |
| Generics toolkit + functional options | M2 | ✅ |
| Concurrency: worker-pool dispatcher | M3 | ✅ |
| REST `customer-api` (`net/http`+`chi`) | M4 | ✅ |
| gRPC `profile-svc` (unary+streaming) | M5 | ✅ |
| PostgreSQL (customers, campaigns, tx) | M6 | ✅ |
| MongoDB (profiles, event log) | M7 | ✅ |
| NATS core (pub/sub, queue, req-reply) | M8 | ✅ |
| JetStream (durable, idempotent, DLQ) | M9 | ✅ |
| Enterprise repo layout | M10 | ✅ |
| Tests + lint + quality gates | M11 | ✅ |
| CI/CD + Docker + k8s | M12 | ✅ |
| Observability + config + shutdown | M13 | ✅ |

**The flagship flow (what this module runs):**

```
POST /v1/customers ──► customer-api ──► save (Postgres)
                              │
                              └─► publish cxm.customer.created (JetStream, dedup MsgID)
                                          │
                                          ▼  durable, at-least-once
                              notification-svc consumer (idempotent)
                                          │
                                          └─► dispatcher (M3) ──► Sender ──► "welcome sent"
                                                                   │ on exhaustion ─► DLQ
```

---

## 2. How the modules compose

- **Boundaries (M1):** every service depends on **interfaces** (`Repository`,
  `Sender`, `Repo`), so infrastructure (Postgres/Mongo/NATS) is swappable and
  testable. Errors carry **codes** (`errorx`) mapped to HTTP (M4) and gRPC (M5).
- **The dispatcher (M3)** is the notification engine; in the capstone its input
  channel is fed by the **JetStream consumer (M9)** instead of a test loop.
- **Idempotency (M9)** guarantees a redelivered event doesn't double-send — the
  consumer checks a dedup store before invoking the dispatcher.
- **Observability (M13)** wraps handlers and records `notifications_sent_total`,
  so the flow is measurable; **graceful shutdown** drains HTTP, the consumer, the
  dispatcher, and flushes telemetry in order.
- **Config (M10/M13)** injects all endpoints; **CI/CD (M12)** builds and ships it.

---

## 3. Hands-on: read & run the vertical slice

The code in `m14-capstone/` is the integrated slice. Walk it in this order:

1. `domain/` — `Customer`, `CustomerEvent` (the contract on the wire).
2. `api/handler.go` — `POST /v1/customers`: validate → store → **publish event**.
   Note the handler returns `201` to the client *before* the notification is sent
   — that's the point of event-driven decoupling.
3. `events/stream.go` — stream/consumer setup, publish-with-dedup, and the
   idempotent consumer that feeds the dispatcher.
4. `notify/` — a `Sender` wrapped by the M3 dispatcher.
5. `cmd/pulse/main.go` — wires it all, drives 5 creates (one duplicate), prints
   the resulting metrics, and shuts down gracefully.

**Run it** (`go run ./cmd/pulse`) and you'll see: 5 HTTP creates, 5 events
published (the duplicate suppressed by MsgID dedup), and 5 welcome "sends" — proof
the whole chain works.

**The e2e test** (`pulse_e2e_test.go`) asserts the same thing programmatically:
create via `httptest`, then wait until the notification side records the send.

---

## 4. Fill-in-the-blank (the wiring)

The heart of the capstone is connecting the consumer to the dispatcher. Complete it.

```go
// On each delivered, not-yet-seen event, hand it to the dispatcher and ack.
func (c *Consumer) handle(ctx context.Context, msg jetstream.Msg) {
    var e domain.CustomerEvent
    if err := json.Unmarshal(msg.Data(), &e); err != nil {
        _ = msg.Term()
        return
    }
    if c.dedup.Seen(e.ID) {
        _ = msg./* ___1: ack a duplicate without re-sending ___ */()
        return
    }
    if err := c.send(e); err != nil {
        _ = msg./* ___2: request redelivery ___ */()
        return
    }
    c.dedup.Mark(e.ID)
    _ = msg./* ___3 ___ */()
}
```

<details>
<summary>Answers</summary>

```go
_ = msg.Ack()  // 1 — duplicate already processed; ack so it stops redelivering
_ = msg.Nak()  // 2 — transient failure; retry
_ = msg.Ack()  // 3 — success
```

</details>

---

## 5. Implement it yourself — finish Pulse

Take the vertical slice to the full multi-service system in the M10 `pulse/` repo:

1. **Persistence:** replace in-memory stores with the **Postgres** customer repo
   (M6) and **Mongo** profile/event repos (M7). Run migrations on startup.
2. **gRPC:** stand up `profile-svc` (M5) and have `customer-api` seed a profile via
   gRPC on `customer.created` (cross-service call, traced end-to-end).
3. **Three processes:** split `customer-api`, `profile-svc`, `notification-svc` into
   the three `cmd/` binaries; run them with **docker-compose** (M12) alongside
   Postgres + Mongo + NATS.
4. **Reliability:** idempotent consumer backed by a **durable** dedup store
   (a `processed_events` table), retries, and a **DLQ drainer** that alerts.
5. **Ops:** add `/metrics` + tracing (M13) to each service; gate CI on lint +
   race tests + coverage (M11/M12); ship images on tags.
6. **Harden:** auth on REST (M4) and gRPC (M5), input validation, PII redaction in
   logs and responses; load-test the dispatcher and tune worker counts.
7. **Runbook:** write `README.md` + a `RUNBOOK.md` (how to deploy, roll back, read
   the dashboards, drain the DLQ, interpret the key metrics/alerts).

**Curated resources (capstone-level):**
- *Let's Go Further* (Alex Edwards) — the canonical idiomatic Go API book —
  https://lets-go-further.alexedwards.net/
- *Designing Data-Intensive Applications* (Kleppmann) — the systems bible.
- *Designing Event-Driven Systems* (Stopford, free) —
  https://www.confluent.io/designing-event-driven-systems/
- microservices.io patterns (saga, outbox, CQRS) — https://microservices.io/patterns/
- The **transactional outbox** pattern (publish events atomically with DB writes) —
  https://microservices.io/patterns/data/transactional-outbox.html
- Go project: `service-weaver`, `encore` (for contrast with hand-rolled) — optional.

> **Advanced challenge — the outbox pattern:** right now we save the customer and
> publish the event in two steps; a crash between them loses the event. Implement
> the **transactional outbox**: in the same Postgres transaction, insert the
> customer *and* an `outbox` row; a separate poller publishes outbox rows to
> JetStream and marks them sent. This gives you atomic "save-and-emit" — the
> production-grade fix, and a great final exercise.

---

## 6. Definition of Done — the course success criteria

You've completed Pulse (and the course) when you can:

- [ ] Design and build idiomatic, concurrent Go backend services from scratch.
- [ ] Expose **both REST and gRPC** and integrate **Postgres + Mongo**.
- [ ] Design and implement event-driven flows with **NATS Core + JetStream**
      (durable, idempotent, with retries + DLQ).
- [ ] Stand up and maintain an **enterprise-grade repo** end-to-end (structure,
      tests, CI/CD, containerization, observability).
- [ ] Explain the *why* behind idiomatic Go choices and avoid common anti-patterns.

If every box across all 14 modules' self-checks is ticked — you're there. You can
now build production Go backends independently, in CXM or any domain.

---

## 7. Self-check — you should now be able to…

- [ ] Trace a single request/event across the whole system (REST → JetStream → notify).
- [ ] Explain where each module's code lives in the assembled repo and why.
- [ ] Reason about failure modes (lost events, duplicates, partial failures) and the
      patterns that address them (idempotency, DLQ, outbox, graceful shutdown).
- [ ] Operate the system: read its metrics/logs/traces and run the DLQ/runbook procedures.
