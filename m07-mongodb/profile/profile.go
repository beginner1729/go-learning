// Package profile is YOUR implementation target for M07 (Exercises 3.1 & 3.2).
//
// Goal: make `go test ./mongodb/... ./profile/...` pass (when PULSE_MONGO_URI
// is set; it skips cleanly otherwise) and `go run ./cmd/demo` work.
// The tests in profile_test.go define the exact API you must provide.
// Reference answer key: ../solution-profile/ (and §3.1/§3.2 in
// ../M07-mongodb.md). Try it yourself before peeking.
//
// This package holds the document domain (Profile + Event) and a MongoDB
// repository, shaped to satisfy the M5 profile.Repo gRPC interface.
//
// Build, in THIS file (profile.go):
//
//   - `ErrNotFound` — a sentinel error (var, errors.New). Get maps the driver's
//     mongo.ErrNoDocuments to this so callers can branch with errors.Is.
//
//   - `Profile` struct: CustomerID string (bson:"_id"), Traits map[string]string
//     (bson:"traits"), UpdatedAt time.Time (bson:"updated_at"). The customer id
//     is the document _id so reads/writes are primary-key efficient.
//
//   - `Event` struct: ID string (bson:"_id"), CustomerID string
//     (bson:"customer_id"), Type string (bson:"type"), OccurredAt time.Time
//     (bson:"occurred_at").
//
//   - `MongoRepo` holding two *mongo.Collection handles (profiles, events).
//
//   - `NewMongoRepo(ctx, client *mongo.Client, dbName string) (*MongoRepo, error)`:
//     grab the db, bind the "profiles" and "events" collections, and create a
//     compound index on events (customer_id asc, occurred_at desc).
//
//   - `Upsert(ctx, Profile) (Profile, error)` — Exercise 3.1: stamp UpdatedAt
//     with time.Now().UTC(), UpdateOne with $set + SetUpsert(true), return the
//     stamped Profile. A second Upsert for the same id replaces traits in place.
//
//   - `Get(ctx, customerID string) (Profile, error)` — Exercise 3.1: FindOne by
//     _id; map mongo.ErrNoDocuments to ErrNotFound; wrap other errors with %w.
//
//   - `AppendEvent(ctx, Event) error` — Exercise 3.2: InsertOne into events.
//
//   - `Events(ctx, customerID string) ([]Event, error)` — Exercise 3.2: Find by
//     customer_id sorted by occurred_at ascending; cur.All into the slice.
//
//   - `EventCountsByType(ctx, customerID string) (map[string]int, error)` — §5
//     aggregation: $match on customer_id then $group by $type counting events.
//
// What the tests expect (profile_test.go):
//   - Two Upserts for "cus_1" leave traits = {tier:"platinum", vip:"true"}.
//   - Get("ghost") returns an error that errors.Is(err, ErrNotFound).
//   - After appending e1..e3 for "cus_1", Events returns 3 events with got[0].ID
//     == "e1" (ascending by OccurredAt).
//   - EventCountsByType("cus_1") == {page_view:2, purchase:1}.
//
// Delete this comment block as you implement. The package will not compile
// until the types and methods the tests reference exist.
package profile

// TODO(3.1/3.2/§5): implement ErrNotFound, Profile, Event, MongoRepo,
// NewMongoRepo, Upsert, Get, AppendEvent, Events, EventCountsByType using
// go.mongodb.org/mongo-driver (mongo, bson, options).
