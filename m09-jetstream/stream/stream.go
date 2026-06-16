// Package stream is YOUR implementation target for M09. It sets up the CXM
// JetStream stream + durable consumer and provides a durable, idempotent,
// retrying, dead-lettering notification consumer.
//
// Goal: make `go test ./events/... ./stream/...` build, then pass (or skip
// cleanly when PULSE_NATS_URL is unset — the integration tests need a running
// NATS JetStream). The tests in stream_test.go define the exact API you must
// provide. Reference answer key: ../solution-stream/ (and §3–§5 in
// ../M09-jetstream.md). Try it yourself before peeking.
//
// Use the new jetstream API: github.com/nats-io/nats.go and
// github.com/nats-io/nats.go/jetstream. Build, in THIS file (stream.go):
//
//   - Connect(url string) (*nats.Conn, jetstream.JetStream, error) — dial NATS
//     (nats.Connect with infinite reconnects) and return jetstream.New(nc).
//
//   - EnsureStream(ctx, js) (jetstream.Stream, error) — CreateOrUpdateStream
//     (idempotent) named events.StreamName over events.SubjectCustomerAll,
//     LimitsPolicy, MaxAge 24h.
//
//   - Publish(ctx, js, e events.CustomerEvent) error — JSON-marshal e and
//     js.Publish to events.SubjectCustomerCreated WithMsgID(e.ID) so duplicate
//     publishes are deduped by JetStream.
//
//   - DedupStore — a concurrency-safe set of processed event IDs (mutex + map).
//     Provide NewDedupStore() *DedupStore, Seen(id string) bool, Mark(id string).
//
//   - Consumer — the durable, idempotent consumer. Fields: the JetStream
//     handle, a *DedupStore, a maxDeliver int, and a send func(events.CustomerEvent) error.
//     Provide:
//     NewConsumer(js, dedup *DedupStore, maxDeliver int, send func(events.CustomerEvent) error) *Consumer
//     (c *Consumer) Start(ctx, stream jetstream.Stream, durable string) (func(), error)
//     Start creates a durable AckExplicit consumer (MaxDeliver, AckWait,
//     FilterSubject = events.SubjectCustomerCreated), begins Consume-ing, and
//     returns the ConsumeContext's Stop func.
//
//   - The per-message handler (e.g. handle): unmarshal — Term() on parse error;
//     if NumDelivered exceeds maxDeliver, republish to events.SubjectDLQ and
//     Term(); if the ID was already Seen, Ack (idempotent skip); call send — on
//     failure Nak to retry, but on the final attempt DLQ + Term; on success
//     Mark the ID and Ack.
//
// The tests exercise durable delivery of 10 events, retry-then-succeed, and
// dead-lettering of a poison message — so your ack/nak/term + dedup logic must
// match those behaviours.
//
// Delete this comment block as you implement. The package will not compile
// until Connect, EnsureStream, Publish, DedupStore, NewDedupStore, Consumer,
// and NewConsumer exist.
package stream

// TODO(M09): implement Connect, EnsureStream, Publish, DedupStore,
// NewDedupStore, Consumer, NewConsumer, and the message handler.
