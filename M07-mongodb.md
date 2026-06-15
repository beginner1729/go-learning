# Module 7 — MongoDB for Profiles & Events

> **Capstone contribution:** the document side of Pulse — flexible **customer
> profiles** and an append-only **event log**, backing the `profile-svc` from M5.
> The gRPC surface stays identical; only the repo implementation changes.

---

## 0. Setup & run

```bash
cd go-cxm-course/m07-mongodb
docker run -d --name pulse-mongo -p 27018:27017 mongo:7
export PULSE_MONGO_URI="mongodb://localhost:27018"

go mod tidy
go vet ./...
go test ./...        # integration tests SKIP if PULSE_MONGO_URI is unset
go run ./cmd/demo
```

Layout:

```
m07-mongodb/
  go.mod                 # module cxm/m07
  profile/               # Profile + Event domain, Mongo repository, indexes
  mongodb/               # client construction + ping
  cmd/demo/
```

---

## 1. Learning objectives

By the end you will be able to:

- Connect to MongoDB with the official Go driver and manage a client lifecycle.
- Model documents idiomatically (BSON tags, embedded vs referenced data).
- Perform CRUD, **upserts**, filtered queries, and **aggregations** from Go.
- Create **indexes** (including unique) and understand why they matter.
- Articulate **when to choose document vs relational** storage with CXM examples.

---

## 2. Concepts

### 2.1 When document beats relational (the CXM decision)

| Use **Postgres** (M6) when… | Use **MongoDB** when… |
|---|---|
| Data is uniform & relational (customers, campaigns, enrollments). | Schema varies per record (profile *traits* differ by customer). |
| You need multi-row transactions, FKs, joins. | You append high-volume, schema-flexible records (an **event log**). |
| Strong consistency on related rows. | You read whole documents by key and rarely join. |

In Pulse: **customers/campaigns → Postgres**; **profiles (arbitrary traits) and
the event stream → MongoDB**. A profile is naturally a document: a bag of
attributes that evolves without migrations. Events are an append-only, high-write
log — a classic document/time-series fit.

> **Anti-pattern:** using Mongo as a relational DB (lots of cross-collection
> "joins" via `$lookup`, multi-document transactions everywhere). If you're doing
> that, you wanted Postgres. Model documents around your *access pattern*: store
> together what you read together.

### 2.2 The driver & client lifecycle

The official driver is `go.mongodb.org/mongo-driver/mongo`. Create **one** client
and share it (it's a pool):

```go
client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
if err != nil { ... }
if err := client.Ping(ctx, nil); err != nil { ... } // fail fast
defer client.Disconnect(ctx)

coll := client.Database("pulse").Collection("profiles")
```

### 2.3 Modeling with BSON

Documents map to Go structs via `bson` tags (like `json` tags but for BSON):

```go
type Profile struct {
    CustomerID string            `bson:"_id"`        // use the natural key as _id
    Traits     map[string]string `bson:"traits"`     // flexible attributes
    UpdatedAt  time.Time         `bson:"updated_at"`
}
```

Using `customer_id` as `_id` makes lookups primary-key fast and enforces
uniqueness for free. `omitempty` on optional fields keeps documents lean.

### 2.4 CRUD + upsert

Upsert (insert-or-update) is the bread-and-butter for profiles:

```go
filter := bson.M{"_id": p.CustomerID}
update := bson.M{"$set": bson.M{"traits": p.Traits, "updated_at": p.UpdatedAt}}
opts := options.Update().SetUpsert(true)
_, err := coll.UpdateOne(ctx, filter, update, opts)
```

Read one:

```go
var p Profile
err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
if errors.Is(err, mongo.ErrNoDocuments) {
    return Profile{}, ErrNotFound // map driver sentinel to domain
}
```

Query many + decode all:

```go
cur, err := coll.Find(ctx, bson.M{"customer_id": id}, options.Find().SetSort(bson.D{{"occurred_at", 1}}))
if err != nil { ... }
defer cur.Close(ctx)
var events []Event
if err := cur.All(ctx, &events); err != nil { ... } // decode the whole cursor
```

### 2.5 Indexes

Without indexes, queries do full collection scans. Create them at startup
(idempotent):

```go
_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
    Keys:    bson.D{{Key: "customer_id", Value: 1}, {Key: "occurred_at", Value: -1}},
    Options: options.Index().SetName("by_customer_time"),
})
```

A **unique** index enforces constraints (e.g. one event ID once):
`options.Index().SetUnique(true)`. Index the fields you filter/sort on.

### 2.6 Aggregation (brief)

For analytics-style reads (e.g. event counts per type), use the aggregation
pipeline:

```go
pipeline := mongo.Pipeline{
    {{"$match", bson.M{"customer_id": id}}},
    {{"$group", bson.M{"_id": "$type", "count": bson.M{"$sum": 1}}}},
}
cur, err := coll.Aggregate(ctx, pipeline)
```

#### Pitfalls

- **Forgetting `cur.Close(ctx)`** → cursor/connection leak.
- **Unbounded `Find`** → loading a huge collection into memory; paginate/limit.
- **Not mapping `mongo.ErrNoDocuments`** → leaks driver errors upstream.
- **No indexes** → silent full scans that only hurt at scale.
- **Over-embedding** → documents that grow unboundedly (e.g. embedding *all* events
  inside the profile doc) hit the 16MB document cap. Keep the event log a separate
  collection.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Upsert a profile and read it back

**Task:** Implement `Upsert(ctx, Profile) (Profile, error)` and `Get(ctx, id)
(Profile, error)` mapping `ErrNoDocuments` → `ErrNotFound`.

<details>
<summary>Reference solution</summary>

```go
func (r *MongoRepo) Upsert(ctx context.Context, p Profile) (Profile, error) {
    p.UpdatedAt = time.Now().UTC()
    _, err := r.profiles.UpdateOne(ctx,
        bson.M{"_id": p.CustomerID},
        bson.M{"$set": bson.M{"traits": p.Traits, "updated_at": p.UpdatedAt}},
        options.Update().SetUpsert(true),
    )
    if err != nil {
        return Profile{}, fmt.Errorf("upsert profile: %w", err)
    }
    return p, nil
}

func (r *MongoRepo) Get(ctx context.Context, id string) (Profile, error) {
    var p Profile
    err := r.profiles.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
    if errors.Is(err, mongo.ErrNoDocuments) {
        return Profile{}, ErrNotFound
    }
    if err != nil {
        return Profile{}, fmt.Errorf("get profile: %w", err)
    }
    return p, nil
}
```

**Reasoning:** Upsert avoids a read-modify-write race and a separate "exists"
check. `_id` as the customer ID makes both operations primary-key efficient.

</details>

### Exercise 3.2 — Append + list events

**Task:** Implement `AppendEvent(ctx, Event)` and `Events(ctx, customerID)`
sorted by time ascending.

<details>
<summary>Reference solution</summary>

```go
func (r *MongoRepo) AppendEvent(ctx context.Context, e Event) error {
    _, err := r.events.InsertOne(ctx, e)
    if err != nil {
        return fmt.Errorf("append event: %w", err)
    }
    return nil
}

func (r *MongoRepo) Events(ctx context.Context, customerID string) ([]Event, error) {
    cur, err := r.events.Find(ctx,
        bson.M{"customer_id": customerID},
        options.Find().SetSort(bson.D{{Key: "occurred_at", Value: 1}}),
    )
    if err != nil {
        return nil, fmt.Errorf("find events: %w", err)
    }
    defer cur.Close(ctx)
    var out []Event
    if err := cur.All(ctx, &out); err != nil {
        return nil, fmt.Errorf("decode events: %w", err)
    }
    return out, nil
}
```

**Reasoning:** Events are append-only `InsertOne`s into a dedicated collection.
The compound index `(customer_id, occurred_at)` makes this query index-covered.

</details>

---

## 4. Fill-in-the-blank

Complete client construction and the index setup.

```go
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, fmt.Errorf("connect: %w", err)
    }
    if err := client./* ___1: verify connectivity ___ */(ctx, nil); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }
    return client, nil
}

func ensureIndexes(ctx context.Context, events *mongo.Collection) error {
    _, err := events.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{
            {Key: "customer_id", Value: 1},
            {Key: "occurred_at", Value: /* ___2: descending ___ */},
        },
    })
    return err
}
```

<details>
<summary>Answers</summary>

```go
if err := client.Ping(ctx, nil); err != nil { ... } // 1
...
{Key: "occurred_at", Value: -1},                     // 2 (-1 = descending)
```

</details>

---

## 5. Implement it yourself

**Problem:** Make M7's `MongoRepo` satisfy the **M5 `profile.Repo` interface**, then
swap it into the gRPC `profile-svc` so the service is now Mongo-backed end-to-end
(start the gRPC server, call `UpsertProfile`/`ListEvents`, see documents in Mongo).
Add:
- An **aggregation** endpoint: event counts grouped by `type` for a customer.
- A **TTL index** on events so raw events auto-expire after N days (retention).
- A small **load test**: insert 10k events, time an indexed vs non-indexed query.

**Curated resources:**
- Go driver docs — https://www.mongodb.com/docs/drivers/go/current/
- Quick start — https://www.mongodb.com/docs/drivers/go/current/quick-start/
- BSON & struct tags — https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
- Indexes — https://www.mongodb.com/docs/drivers/go/current/fundamentals/indexes/
- Aggregation — https://www.mongodb.com/docs/drivers/go/current/fundamentals/aggregation/
- Data modeling — https://www.mongodb.com/docs/manual/core/data-model-design/
- TTL indexes — https://www.mongodb.com/docs/manual/core/index-ttl/

**Hints:** the M5 `Repo` interface returns `[]profile.Event`; reuse your BSON
`Event` with a mapping function. TTL index: `SetExpireAfterSeconds` on a date field.

---

## 6. Capstone contribution

`profile-svc` is now backed by MongoDB for profiles + events, with indexes for
fast lookups. Pulse now uses **both** datastores deliberately: Postgres for the
relational core, Mongo for flexible/append-only data — and you can justify each.
M8 introduces the messaging backbone that ties the services together.

---

## 7. Self-check — you should now be able to…

- [ ] Decide document vs relational for a given CXM dataset and justify it.
- [ ] Manage a Mongo client lifecycle and map `ErrNoDocuments` to a domain error.
- [ ] Do CRUD, upserts, filtered/sorted queries, and a basic aggregation from Go.
- [ ] Create indexes (incl. unique/compound/TTL) and explain their purpose.
- [ ] Keep document modeling aligned to access patterns (store-together-read-together).
