# Module 8 — NATS Core: pub/sub, queue groups, request-reply

> **Capstone contribution:** the messaging backbone connecting Pulse services.
> Core NATS gives us fire-and-forget **event publishing**, **queue-group** load
> balancing across notification workers, and **request-reply** for a synchronous
> internal lookup. (Durability/replay comes in M9 with JetStream.)

---

## 0. Setup & run

```bash
cd go-cxm-course/m08-nats-core
docker run -d --name pulse-nats -p 4222:4222 nats:2-alpine -js
export PULSE_NATS_URL="nats://localhost:4222"

go mod tidy
go vet ./...
go test ./...        # SKIP if PULSE_NATS_URL unset
go run ./cmd/demo
```

Layout:

```
m08-nats-core/
  go.mod                 # module cxm/m08
  events/                # event types + JSON codec + subject naming
  bus/                   # thin NATS wrapper: Publish, Subscribe, QueueSubscribe, Request
  cmd/demo/              # publisher + queue-group workers + request-reply
```

---

## 1. Learning objectives

By the end you will be able to:

- Explain NATS's messaging model and the **at-most-once** semantics of core NATS.
- Publish and subscribe to **subjects**, including wildcard subscriptions.
- Use **queue groups** to load-balance messages across worker instances.
- Implement **request-reply** for synchronous RPC-style calls over NATS.
- Design a sane **subject naming** scheme for a CXM event taxonomy.
- Know when core NATS is enough vs when you need JetStream (M9).

---

## 2. Concepts

### 2.1 The NATS model

NATS is a lightweight pub/sub messaging system. Publishers send to a **subject**
(a dotted string like `cxm.customer.created`); subscribers express interest in
subjects. Core NATS is **at-most-once**: if no subscriber is listening, the
message is dropped. It's blazing fast and great for ephemeral signaling, but it
does **not** persist messages — that's JetStream's job (M9).

```go
nc, err := nats.Connect(url,
    nats.Name("pulse-customer-api"),
    nats.MaxReconnects(-1),               // reconnect forever
    nats.ReconnectWait(time.Second),
)
defer nc.Drain() // flush pending + unsubscribe gracefully (preferred over Close)
```

`Drain()` vs `Close()`: **`Drain` is the graceful shutdown** — it processes
in-flight messages and pending publishes before disconnecting. Always prefer it.

### 2.2 Subjects & wildcards

Subjects are hierarchical, dot-separated tokens. Two wildcards:
- `*` matches exactly one token: `cxm.customer.*` → `cxm.customer.created`,
  `cxm.customer.updated`.
- `>` matches one or more trailing tokens: `cxm.>` → everything under `cxm`.

**Design a taxonomy up front.** Pulse uses `cxm.<entity>.<action>`:

```
cxm.customer.created
cxm.customer.updated
cxm.journey.step.entered
cxm.notification.requested
```

Good subject design makes consumers' filters obvious and lets you add new event
types without touching existing subscribers.

### 2.3 Publish / Subscribe

```go
// publish: payload is bytes — we JSON-encode our event structs.
data, _ := json.Marshal(evt)
nc.Publish("cxm.customer.created", data)

// async subscribe: handler runs in a NATS-managed goroutine.
sub, _ := nc.Subscribe("cxm.customer.*", func(m *nats.Msg) {
    var evt CustomerEvent
    _ = json.Unmarshal(m.Data, &evt)
    handle(evt)
})
defer sub.Unsubscribe()
```

#### Pitfall: slow consumers

A subscriber's handler must keep up. If it's slow, NATS buffers up to a limit then
drops messages ("slow consumer"). For heavy work, hand off to a worker pool
(M3!) instead of doing it inline in the handler. For guaranteed delivery, use
JetStream.

### 2.4 Queue groups — load balancing

If multiple subscribers join the **same queue group** on a subject, NATS delivers
each message to **exactly one** member (round-robin-ish). This is how you scale
horizontally: run N notification workers in queue group `notify-workers`, and each
message is handled once across the fleet.

```go
// Every instance of notification-svc uses the same queue name.
nc.QueueSubscribe("cxm.notification.requested", "notify-workers", handler)
```

Without a queue group, *every* subscriber gets *every* message (fan-out). With
one, the group collectively gets each message once (competing consumers). This
distinction is central to event-driven design.

### 2.5 Request-reply

NATS has built-in RPC: the requester publishes with a temporary reply subject and
waits; a responder sends back to that subject.

```go
// responder
nc.Subscribe("cxm.profile.lookup", func(m *nats.Msg) {
    p := lookup(string(m.Data))
    resp, _ := json.Marshal(p)
    m.Respond(resp) // reply to m.Reply
})

// requester (with timeout — always!)
msg, err := nc.Request("cxm.profile.lookup", []byte("cus_1"), 2*time.Second)
if err != nil { /* nats.ErrTimeout if no responder answered in time */ }
```

Use request-reply for quick internal lookups where you want a synchronous answer
but not a full gRPC channel. (For typed, streaming, contract-driven calls, gRPC
from M5 is still the better tool — pick per use case.)

#### Pitfalls

- **No timeout on `Request`** → can block forever. Always pass one.
- **Doing heavy work in the subscription callback** → slow consumer drops.
- **Forgetting `Drain`** on shutdown → in-flight messages lost.
- **Treating core NATS as durable** → it isn't; restart loses undelivered messages.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — A typed event bus wrapper

**Task:** Wrap `*nats.Conn` in a `Bus` that publishes typed events (JSON-encoded)
to `cxm.<entity>.<action>` subjects and subscribes with a typed handler.

<details>
<summary>Reference solution</summary>

```go
type Bus struct{ nc *nats.Conn }

func (b *Bus) Publish(subject string, v any) error {
    data, err := json.Marshal(v)
    if err != nil {
        return fmt.Errorf("marshal: %w", err)
    }
    return b.nc.Publish(subject, data)
}

// Subscribe decodes each message into a fresh T and calls fn.
func Subscribe[T any](b *Bus, subject string, fn func(T)) (*nats.Subscription, error) {
    return b.nc.Subscribe(subject, func(m *nats.Msg) {
        var v T
        if err := json.Unmarshal(m.Data, &v); err != nil {
            return // in production: log + maybe dead-letter
        }
        fn(v)
    })
}
```

**Reasoning:** `Subscribe` is a generic *function* (not a method) because Go
methods can't have type parameters (M2). The codec lives in one place, so every
event is consistently JSON-encoded.

</details>

### Exercise 3.2 — Queue-group worker count

**Task:** Start 3 workers in one queue group, publish 9 messages, and assert each
message was handled exactly once and work was spread across workers.

<details>
<summary>Reference solution (sketch)</summary>

```go
var mu sync.Mutex
perWorker := map[int]int{}
var total atomic.Int64
for w := 0; w < 3; w++ {
    w := w
    nc.QueueSubscribe("cxm.work", "g", func(m *nats.Msg) {
        mu.Lock(); perWorker[w]++; mu.Unlock()
        total.Add(1)
    })
}
for i := 0; i < 9; i++ { nc.Publish("cxm.work", []byte("x")) }
nc.Flush()
// wait briefly, then: total == 9, and len(perWorker) > 1 (work was distributed)
```

**Reasoning:** `total == 9` proves *exactly-once across the group* (no duplication);
`len(perWorker) > 1` proves load balancing. `nc.Flush()` ensures publishes reached
the server before we assert.

</details>

---

## 4. Fill-in-the-blank

Complete the connect + request-reply responder.

```go
func Connect(url, name string) (*nats.Conn, error) {
    return nats.Connect(url,
        nats.Name(name),
        nats.MaxReconnects(-1),
        nats.ReconnectWait(time.Second),
    )
}

func serveLookup(nc *nats.Conn, lookup func(string) []byte) {
    nc.Subscribe("cxm.profile.lookup", func(m *nats.Msg) {
        result := lookup(string(m.Data))
        _ = m./* ___1: send the reply ___ */(result)
    })
}

func askLookup(nc *nats.Conn, id string) ([]byte, error) {
    msg, err := nc./* ___2: synchronous request ___ */("cxm.profile.lookup", []byte(id), 2*time.Second)
    if err != nil {
        return nil, err
    }
    return msg.Data, nil
}
```

<details>
<summary>Answers</summary>

```go
_ = m.Respond(result)                                              // 1
msg, err := nc.Request("cxm.profile.lookup", []byte(id), 2*time.Second) // 2
```

</details>

---

## 5. Implement it yourself

**Problem:** Wire Pulse's services with core NATS:
- When `customer-api` (M4) creates a customer, **publish** `cxm.customer.created`
  with the customer payload.
- Stand up a `notification-svc` that **queue-subscribes** to
  `cxm.customer.created` in group `notify-workers` and feeds each event into the
  **M3 dispatcher** (so the welcome notification actually "sends").
- Add a **request-reply** endpoint `cxm.profile.lookup` served by `profile-svc`.
- Demonstrate horizontal scaling: run 2 `notification-svc` instances; confirm each
  event is handled once across them.

This is the first time all your pieces talk to each other. (M9 makes the
event delivery durable.)

**Curated resources:**
- NATS concepts — https://docs.nats.io/nats-concepts/core-nats
- `nats.go` client — https://github.com/nats-io/nats.go
- Subjects & wildcards — https://docs.nats.io/nats-concepts/subjects
- Queue groups — https://docs.nats.io/nats-concepts/core-nats/queue
- Request-reply — https://docs.nats.io/nats-concepts/core-nats/reqreply

**Hints:** keep subscription handlers thin — enqueue onto the dispatcher's input
channel and return. Run two instances by `go run`-ing the service twice; both join
the same queue group.

---

## 6. Capstone contribution

Pulse services now communicate over NATS: `customer-api` emits domain events,
`notification-svc` consumes them via a queue group and dispatches notifications,
and a request-reply path exists for quick lookups. The architecture is now truly
event-driven — but delivery isn't yet durable. M9 fixes that with JetStream.

---

## 7. Self-check — you should now be able to…

- [ ] Explain core NATS's at-most-once model and when it's appropriate.
- [ ] Publish/subscribe with subjects and wildcards.
- [ ] Use queue groups to load-balance work across instances (exactly-once-in-group).
- [ ] Implement request-reply with a timeout.
- [ ] Design a clear subject taxonomy for an event-driven system.
