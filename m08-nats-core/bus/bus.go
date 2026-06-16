// Package bus is YOUR implementation target for M08 (NATS Core).
//
// Goal: make `go test ./bus/...` pass and `go run ./cmd/demo` work. The tests
// in bus_test.go define the exact API you must provide. They are NATS
// integration tests: they SKIP cleanly when PULSE_NATS_URL is unset, so run a
// local NATS server and export PULSE_NATS_URL to actually exercise them (see
// §0 of ../M08-nats-core.md). Reference answer key: ../solution-bus/ (and the
// §3/§4 details in ../M08-nats-core.md). Try it yourself before peeking.
//
// Build a thin, typed wrapper over a *nats.Conn. In THIS file (bus.go):
//
//   - `Bus` struct that holds a *nats.Conn (github.com/nats-io/nats.go).
//
//   - `Connect(url, name string) (*Bus, error)` — dials NATS with sensible
//     reconnect settings: nats.Name(name), nats.MaxReconnects(-1),
//     nats.ReconnectWait(time.Second). Wrap dial errors with %w.
//
//   - `(*Bus) Conn() *nats.Conn` — expose the underlying connection.
//
//   - `(*Bus) Drain() error` — graceful shutdown (preferred over Close).
//
//   - `(*Bus) Flush() error` — block until the server processed prior
//     publishes (the tests call this before asserting).
//
//   - `(*Bus) Publish(subject string, v any) error` — JSON-encode v and send
//     it to subject. Wrap marshal errors with %w.
//
//   - `Subscribe[T any](b *Bus, subject string, fn func(T)) (*nats.Subscription, error)`
//     — a generic FUNCTION (Go methods can't take type parameters). Async
//     subscribe; decode each message into a fresh T and call fn. Drop messages
//     that fail to decode.
//
//   - `QueueSubscribe[T any](b *Bus, subject, queue string, fn func(T)) (*nats.Subscription, error)`
//     — like Subscribe but joins a queue group: each message goes to exactly
//     one member of the group (competing consumers / load balancing).
//
//   - `(*Bus) Request(subject string, payload []byte, timeout time.Duration) ([]byte, error)`
//     — synchronous request-reply with a timeout. Always pass a timeout.
//
//   - `(*Bus) ServeReply(subject string, handler func([]byte) []byte) (*nats.Subscription, error)`
//     — register a responder that replies via m.Respond(handler(m.Data)).
//
// What the tests check (bus_test.go):
//   - TestPubSubTyped: Subscribe + Publish round-trips a CustomerEvent.
//   - TestQueueGroupExactlyOnce: 3 QueueSubscribe workers in group "g" receive
//     30 published messages exactly once total, distributed across workers.
//   - TestRequestReply: ServeReply + Request returns the responder's bytes.
//
// Delete this comment block as you implement. The package will not compile
// until the Bus type and the functions the tests reference exist.
package bus

// TODO(M08): implement Bus, Connect, Conn, Drain, Flush, Publish,
// Subscribe[T], QueueSubscribe[T], Request, ServeReply.
