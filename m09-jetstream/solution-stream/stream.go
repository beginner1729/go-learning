// Package stream sets up the CXM JetStream stream + durable consumer and
// provides a durable, idempotent, retrying, dead-lettering notification consumer.
package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	events "cxm/m09/solution-events"
)

// Connect dials NATS and returns a JetStream context.
func Connect(url string) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(url, nats.MaxReconnects(-1), nats.ReconnectWait(time.Second))
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream: %w", err)
	}
	return nc, js, nil
}

// EnsureStream creates/updates the CXM_EVENTS stream (idempotent).
func EnsureStream(ctx context.Context, js jetstream.JetStream) (jetstream.Stream, error) {
	return js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      events.StreamName,
		Subjects:  []string{events.SubjectCustomerAll},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    24 * time.Hour,
	})
}

// Publish sends an event with a dedup MsgID (exactly-once publishing).
func Publish(ctx context.Context, js jetstream.JetStream, e events.CustomerEvent) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	_, err = js.Publish(ctx, events.SubjectCustomerCreated, data, jetstream.WithMsgID(e.ID))
	return err
}

// DedupStore records processed event IDs. In production this is Postgres/Redis;
// here it's an in-memory set so idempotency is demonstrable and race-free.
type DedupStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewDedupStore() *DedupStore { return &DedupStore{seen: map[string]struct{}{}} }

func (d *DedupStore) Seen(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.seen[id]
	return ok
}
func (d *DedupStore) Mark(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[id] = struct{}{}
}

// Consumer is the durable, idempotent notification consumer.
type Consumer struct {
	js         jetstream.JetStream
	dedup      *DedupStore
	maxDeliver int
	send       func(events.CustomerEvent) error
}

func NewConsumer(js jetstream.JetStream, dedup *DedupStore, maxDeliver int,
	send func(events.CustomerEvent) error) *Consumer {
	return &Consumer{js: js, dedup: dedup, maxDeliver: maxDeliver, send: send}
}

// Start creates the durable consumer and begins consuming. Stop with the returned func.
func (c *Consumer) Start(ctx context.Context, stream jetstream.Stream, durable string) (func(), error) {
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    c.maxDeliver,
		AckWait:       5 * time.Second,
		FilterSubject: events.SubjectCustomerCreated,
	})
	if err != nil {
		return nil, fmt.Errorf("consumer: %w", err)
	}
	cc, err := cons.Consume(func(msg jetstream.Msg) { c.handle(ctx, msg) })
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	return cc.Stop, nil
}

func (c *Consumer) handle(ctx context.Context, msg jetstream.Msg) {
	var e events.CustomerEvent
	if err := json.Unmarshal(msg.Data(), &e); err != nil {
		_ = msg.Term() // unparseable -> never redeliver
		return
	}

	md, _ := msg.Metadata()
	if md != nil && int(md.NumDelivered) > c.maxDeliver {
		_, _ = c.js.Publish(ctx, events.SubjectDLQ, msg.Data())
		_ = msg.Term()
		return
	}

	// Idempotency: a redelivered (already-processed) event is acked, not re-sent.
	if c.dedup.Seen(e.ID) {
		_ = msg.Ack()
		return
	}

	if err := c.send(e); err != nil {
		// On the final attempt, dead-letter instead of looping.
		if md != nil && int(md.NumDelivered) >= c.maxDeliver {
			_, _ = c.js.Publish(ctx, events.SubjectDLQ, msg.Data())
			_ = msg.Term()
			return
		}
		_ = msg.Nak()
		return
	}

	c.dedup.Mark(e.ID) // mark only after a successful send
	_ = msg.Ack()
}
