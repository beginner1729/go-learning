// Package events is YOUR implementation target for M09. It holds the shared
// stream/subject names and the event payload that both the publisher and the
// durable consumer agree on.
//
// Goal: make `go test ./events/... ./stream/...` build, then pass (or skip
// cleanly when PULSE_NATS_URL is unset). The tests in ../stream/stream_test.go
// reference this API. Reference answer key: ../solution-events/ (and §3–§5 in
// ../M09-jetstream.md). Try it yourself before peeking.
//
// Build, in THIS file (events.go):
//
//   - Exported string constants — the wire contract shared by every service:
//     StreamName             = "CXM_EVENTS"
//     SubjectCustomerCreated = "cxm.customer.created"
//     SubjectCustomerAll     = "cxm.customer.>"
//     SubjectDLQ             = "cxm.customer.dlq"
//     SubjectCustomerAll is the stream's wildcard subject; the durable consumer
//     filters on SubjectCustomerCreated; dead letters are republished to
//     SubjectDLQ (which is captured by the same wildcard stream).
//
//   - `CustomerEvent` struct, JSON-tagged so it round-trips through JetStream:
//     ID         string    `json:"id"`
//     Email      string    `json:"email"`
//     Name       string    `json:"name"`
//     OccurredAt time.Time `json:"occurred_at"`
//     The tests build CustomerEvent values and read e.ID; stream.Publish uses
//     e.ID as the JetStream dedup MsgID.
//
// Delete this comment block as you implement. The package will not compile
// until the constants and CustomerEvent exist.
package events

// TODO(M09): implement StreamName, SubjectCustomerCreated, SubjectCustomerAll,
// SubjectDLQ, and the CustomerEvent struct.
