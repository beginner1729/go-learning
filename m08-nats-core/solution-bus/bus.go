// Package bus is a thin, typed wrapper over a NATS connection.
package bus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Bus struct{ nc *nats.Conn }

// Connect dials NATS with sensible reconnect settings.
func Connect(url, name string) (*Bus, error) {
	nc, err := nats.Connect(url,
		nats.Name(name),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Bus{nc: nc}, nil
}

func (b *Bus) Conn() *nats.Conn { return b.nc }

// Drain flushes and unsubscribes gracefully (preferred over Close).
func (b *Bus) Drain() error { return b.nc.Drain() }

// Flush blocks until the server has processed prior publishes (useful in tests).
func (b *Bus) Flush() error { return b.nc.Flush() }

// Publish JSON-encodes v and sends it to subject.
func (b *Bus) Publish(subject string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return b.nc.Publish(subject, data)
}

// Subscribe decodes each message into a fresh T and calls fn. It's a generic
// function (Go methods can't take type parameters).
func Subscribe[T any](b *Bus, subject string, fn func(T)) (*nats.Subscription, error) {
	return b.nc.Subscribe(subject, func(m *nats.Msg) {
		var v T
		if err := json.Unmarshal(m.Data, &v); err != nil {
			return
		}
		fn(v)
	})
}

// QueueSubscribe is like Subscribe but joins a queue group (competing consumers):
// each message goes to exactly one member of the group.
func QueueSubscribe[T any](b *Bus, subject, queue string, fn func(T)) (*nats.Subscription, error) {
	return b.nc.QueueSubscribe(subject, queue, func(m *nats.Msg) {
		var v T
		if err := json.Unmarshal(m.Data, &v); err != nil {
			return
		}
		fn(v)
	})
}

// Request performs a synchronous request-reply with a timeout.
func (b *Bus) Request(subject string, payload []byte, timeout time.Duration) ([]byte, error) {
	msg, err := b.nc.Request(subject, payload, timeout)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", subject, err)
	}
	return msg.Data, nil
}

// ServeReply registers a responder for request-reply on subject.
func (b *Bus) ServeReply(subject string, handler func([]byte) []byte) (*nats.Subscription, error) {
	return b.nc.Subscribe(subject, func(m *nats.Msg) {
		_ = m.Respond(handler(m.Data))
	})
}
