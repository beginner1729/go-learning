# Module 9 — JetStream & Event-Driven Patterns

> **Capstone contribution:** makes Pulse's notification flow **durable and
> reliable** — a JetStream stream persists `cxm.customer.*` events, a durable
> consumer feeds the M3 dispatcher with **at-least-once** delivery,
> **idempotency**, **redelivery/retries**, and a **dead-letter** path.

---

## 0. Setup & run

```bash
cd go-cxm-course/m09-jetstream
docker run -d --name pulse-nats -p 4222:4222 nats:2-alpine -js   # -js enables JetStream
export PULSE_NATS_URL="nats://localhost:4222"

go mod tidy
go vet ./...
go test ./...        # SKIP if PULSE_NATS_URL unset
go run ./cmd/demo
```

Layout:

```
m09-jetstream/
  go.mod                 # module cxm/m09
  events/                # event types + subjects
  stream/                # stream/consumer setup, publish, consume, idempotency, DLQ
  cmd/demo/              # durable consumer + simulated failures + replay
```

---

## 1. Learning objectives

By the end you will be able to:

- Explain JetStream's **persistence** and **at-least-once** delivery vs core NATS.
- Create **streams** and **durable consumers** with retention/ack policies.
- Publish with **dedup IDs** and consume with **ack/nak/term** and redelivery.
- Implement **idempotent** consumers (handle duplicates safely).
- Build a **dead-letter** path for messages that exceed max delivery attempts.
- Choose **choreography vs orchestration** and apply event-driven patterns.

---

## 2. Concepts

### 2.1 Why JetStream

Core NATS (M8) is fire-and-forget. For a notification system you need **durability**
(survive restarts), **replay** (reprocess history), and **delivery guarantees**.
JetStream adds a persistence layer on top of NATS subjects:

- A **stream** captures and stores messages published to a set of subjects.
- A **consumer** is a stateful view over the stream that tracks which messages a
  client has acknowledged. Delivery is **at-least-once**: a message is redelivered
  until you `Ack` it (or it's terminated/expires).

> At-least-once ⇒ **duplicates are possible** (a redelivery after your handler
> succeeded but crashed before acking). Therefore consumers **must be idempotent**.
> This is the single most important event-driven discipline.

### 2.2 Streams

```go
js, _ := jetstream.New(nc)
stream, _ := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
    Name:      "CXM_EVENTS",
    Subjects:  []string{"cxm.customer.>"},
    Retention: jetstream.LimitsPolicy, // keep up to limits (age/size/count)
    MaxAge:    24 * time.Hour,
})
```

Retention policies: **Limits** (keep until age/size/count limit — good default),
**Interest** (keep while consumers haven't acked), **WorkQueue** (each message
consumed once across all consumers, then deleted — great for task queues).

### 2.3 Publishing with dedup

JetStream can deduplicate publishes within a window using a message ID:

```go
_, err := js.Publish(ctx, "cxm.customer.created", data,
    jetstream.WithMsgID(evt.ID)) // same ID within the dedup window = stored once
```

This gives **exactly-once *publishing*** (no dup stored even if your publisher
retries). It does **not** make consumption exactly-once — that's on you (idempotency).

### 2.4 Durable consumers & ack semantics

A **durable** consumer remembers its position across restarts (by name):

```go
cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
    Durable:       "notify-workers",
    AckPolicy:     jetstream.AckExplicitPolicy, // we must Ack each message
    MaxDeliver:    5,                            // give up after 5 attempts
    AckWait:       30 * time.Second,             // redeliver if not acked in time
    FilterSubject: "cxm.customer.created",
})

cc, _ := cons.Consume(func(msg jetstream.Msg) {
    if err := handle(msg); err != nil {
        _ = msg.Nak() // negative ack -> redeliver sooner
        return
    }
    _ = msg.Ack()     // success -> never redelivered
})
defer cc.Stop()
```

The three responses:
- **`Ack`** — done; remove from redelivery.
- **`Nak`** — failed, retry (optionally `NakWithDelay` for backoff).
- **`Term`** — permanent failure; stop redelivering (send to your DLQ first).

`msg.Metadata()` tells you `NumDelivered` — use it to decide when to dead-letter.

### 2.5 Idempotency patterns

Because delivery is at-least-once, design handlers so processing the same event
twice is harmless:

1. **Natural idempotency:** the operation is inherently safe to repeat (an upsert
   keyed by event ID; setting a flag).
2. **Dedup store:** record processed event IDs (a Mongo/Postgres `processed_events`
   table or a Redis set with TTL); skip if already seen.
3. **Conditional writes:** "insert if not exists" / compare-and-set on a version.

In Pulse, the notification consumer records `sent:<eventID>` so a redelivery
doesn't send a second welcome email.

### 2.6 Dead-letter handling

When `NumDelivered >= MaxDeliver`, JetStream stops (or you `Term`). Capture those
so they're not silently lost:

```go
if msg.Metadata().NumDelivered >= maxDeliver {
    _ = js.Publish(ctx, "cxm.dlq.customer.created", msg.Data()) // park it
    _ = msg.Term()  // stop redelivery of the original
    return
}
```

A separate process/alert drains the DLQ for inspection/replay.

### 2.7 Architecture: choreography vs orchestration

- **Choreography:** services react to events independently; no central
  coordinator. `customer.created` → `notification-svc` reacts, `profile-svc`
  reacts, etc. Loosely coupled, scales well, but the end-to-end flow is implicit
  (harder to trace). Pulse is primarily choreographed.
- **Orchestration:** a central workflow service tells each step what to do next
  (a "saga orchestrator"). Explicit, easier to reason about complex flows, but a
  coupling/bottleneck point.

Rule of thumb: choreography for simple reactive flows; orchestration when you need
a multi-step transaction with compensation (a **saga**) — e.g. "create customer →
provision account → send welcome → on failure, roll back." Mentioning the saga
pattern here; the capstone uses choreography with idempotency + DLQ.

#### Pitfalls

- **Non-idempotent consumer** → duplicate side effects (double emails). The #1 bug.
- **Acking before the work is done** → message lost if you then crash. Ack *after* success.
- **No `MaxDeliver`** → a poison message redelivers forever. Always cap + DLQ.
- **Blocking forever in the handler** → exceeds `AckWait`, triggering redelivery
  while still processing. Keep handlers bounded; offload heavy work.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Create stream + durable consumer

**Task:** Create a `CXM_EVENTS` stream over `cxm.customer.>` and a durable
explicit-ack consumer filtered to `cxm.customer.created` with `MaxDeliver=5`.

<details>
<summary>Reference solution</summary>

```go
func Setup(ctx context.Context, js jetstream.JetStream) (jetstream.Consumer, error) {
    stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
        Name:      "CXM_EVENTS",
        Subjects:  []string{"cxm.customer.>"},
        Retention: jetstream.LimitsPolicy,
        MaxAge:    24 * time.Hour,
    })
    if err != nil {
        return nil, fmt.Errorf("stream: %w", err)
    }
    return stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
        Durable:       "notify-workers",
        AckPolicy:     jetstream.AckExplicitPolicy,
        MaxDeliver:    5,
        AckWait:       30 * time.Second,
        FilterSubject: "cxm.customer.created",
    })
}
```

**Reasoning:** `CreateOrUpdate` is idempotent — safe to call on every startup.
The durable name makes the consumer survive restarts and lets multiple workers
share the same consumer (competing consumers, like a queue group but durable).

</details>

### Exercise 3.2 — Idempotent handler with a dedup set

**Task:** Wrap a handler so it skips events whose ID was already processed
(in-memory set for the exercise), acks duplicates, and only "sends" once.

<details>
<summary>Reference solution</summary>

```go
type IdempotentHandler struct {
    mu   sync.Mutex
    seen map[string]struct{}
    send func(events.CustomerEvent) error
}

func (h *IdempotentHandler) Handle(msg jetstream.Msg) {
    var e events.CustomerEvent
    if err := json.Unmarshal(msg.Data(), &e); err != nil {
        _ = msg.Term() // unparseable -> don't redeliver forever
        return
    }
    h.mu.Lock()
    _, dup := h.seen[e.ID]
    h.mu.Unlock()
    if dup {
        _ = msg.Ack() // already processed; ack and move on
        return
    }
    if err := h.send(e); err != nil {
        _ = msg.Nak() // retry later
        return
    }
    h.mu.Lock()
    h.seen[e.ID] = struct{}{}
    h.mu.Unlock()
    _ = msg.Ack()
}
```

**Reasoning:** The dedup check makes redelivery safe. We record the ID only
*after* a successful send, so a crash mid-send leads to a retry (not a skip).
In production the `seen` set is a durable store (Postgres/Redis) with TTL.

</details>

---

## 4. Fill-in-the-blank

Complete the dead-letter logic.

```go
func (c *Consumer) handle(ctx context.Context, msg jetstream.Msg) {
    md, _ := msg.Metadata()
    if md.NumDelivered >= c.maxDeliver {
        _ = c.js.Publish(ctx, c.dlqSubject, msg.Data()) // park for inspection
        _ = msg./* ___1: stop redelivery permanently ___ */()
        return
    }
    if err := c.process(msg); err != nil {
        _ = msg./* ___2: request redelivery ___ */()
        return
    }
    _ = msg./* ___3: success ___ */()
}
```

<details>
<summary>Answers</summary>

```go
_ = msg.Term()  // 1 — permanent stop (we've DLQ'd it)
_ = msg.Nak()   // 2 — retry
_ = msg.Ack()   // 3 — done
```

</details>

---

## 5. Implement it yourself

**Problem:** Complete Pulse's durable notification flow:
- `customer-api` publishes `cxm.customer.created` to JetStream **with a dedup
  MsgID** (the customer/event ID).
- `notification-svc` runs a **durable** consumer that feeds events into the **M3
  dispatcher**, is **idempotent** (dedup store), retries via `Nak`, and
  **dead-letters** after `MaxDeliver`.
- Persist the dedup/`processed_events` set in **Postgres or Mongo** (from M6/M7) so
  idempotency survives restarts.
- Demonstrate **replay**: create a *new* durable consumer with `DeliverAllPolicy`
  and watch it reprocess history (idempotency means no duplicate sends).
- Add a **DLQ drainer** that logs/alerts on dead-lettered events.

**Curated resources:**
- JetStream concepts — https://docs.nats.io/nats-concepts/jetstream
- New `jetstream` Go API — https://pkg.go.dev/github.com/nats-io/nats.go/jetstream
- Consumers — https://docs.nats.io/nats-concepts/jetstream/consumers
- Idempotency & dedup — https://docs.nats.io/using-nats/developer/develop_jetstream/model_deep_dive
- Saga pattern — https://microservices.io/patterns/data/saga.html
- *Designing Event-Driven Systems* (Ben Stopford, free e-book) —
  https://www.confluent.io/designing-event-driven-systems/

**Hints:** the new API is `jetstream.New(nc)`; `cons.Consume(handler)` returns a
`ConsumeContext` you `Stop()` on shutdown. For replay, use a different durable
name + `DeliverPolicy: jetstream.DeliverAllPolicy`.

---

## 6. Capstone contribution

Pulse's event flow is now **production-grade**: durable persistence, at-least-once
delivery, idempotent consumption, retries with redelivery, and a dead-letter path.
The notification pipeline (M3 dispatcher) is fed by a durable JetStream consumer.
This completes the core technical stack — Weeks 5–6 make it an enterprise-grade,
shippable repository.

---

## 7. Self-check — you should now be able to…

- [ ] Explain at-least-once delivery and why consumers must be idempotent.
- [ ] Create streams and durable consumers with the right retention/ack policies.
- [ ] Use Ack/Nak/Term and `NumDelivered` to drive retries and dead-lettering.
- [ ] Implement an idempotent consumer with a dedup store.
- [ ] Choose choreography vs orchestration and describe the saga pattern.
