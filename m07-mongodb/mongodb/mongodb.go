// Package mongodb is YOUR implementation target for M07 (§4 fill-in-the-blank).
//
// Goal: make `go test ./mongodb/... ./profile/...` pass (when PULSE_MONGO_URI
// is set; it skips cleanly otherwise) and `go run ./cmd/demo` work.
// Reference answer key: ../solution-mongodb/ (and §4 in ../M07-mongodb.md).
// Try it yourself before peeking.
//
// Build, in THIS file (mongodb.go) — §4:
//
//   - `Connect(ctx context.Context, uri string) (*mongo.Client, error)`:
//     dial Mongo with mongo.Connect + options.Client().ApplyURI(uri), then
//     verify connectivity with a Ping so a bad URI fails fast. On a ping
//     failure, Disconnect and return the wrapped error. Wrap errors with %w
//     ("connect: %w", "ping: %w").
//
// The profile package's MongoRepo (which you also implement) relies on a live
// client from this Connect to build collections and indexes.
//
// Delete this comment block as you implement. The package will not compile
// until Connect exists.
package mongodb

// TODO(§4): implement Connect using go.mongodb.org/mongo-driver/mongo and
// .../mongo/options. Ping after Connect; Disconnect on ping failure.
