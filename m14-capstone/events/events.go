// Package events wires JetStream: stream setup, dedup publish, and a durable,
// idempotent consumer that feeds a send function (the dispatcher).
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"cxm/m14/domain"
)

type Publisher struct{ js jetstream.JetStream }

func Connect(url string) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(url, nats.MaxReconnects(-1), nats.ReconnectWait(time.Second))
	if err != nil {
		return nil, nil, fmt.Errorf("connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream: %w", err)
	}
	return nc, js, nil
}

func EnsureStream(ctx context.Context, js jetstream.JetStream) (jetstream.Stream, error) {
	return js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      domain.StreamName,
		Subjects:  []string{domain.SubjectCustomerAll},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    24 * time.Hour,
	})
}

func NewPublisher(js jetstream.JetStream) *Publisher { return &Publisher{js: js} }

// PublishCustomerCreated publishes with the customer ID as the dedup MsgID, so a
// retried publish stores the event only once.
func (p *Publisher) PublishCustomerCreated(ctx context.Context, e domain.CustomerEvent) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	_, err = p.js.Publish(ctx, domain.SubjectCustomerCreated, data, jetstream.WithMsgID(e.ID))
	return err
}

// DedupStore tracks processed event IDs (in-memory here; durable in production).
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
	send       func(domain.CustomerEvent) error
}

func NewConsumer(js jetstream.JetStream, dedup *DedupStore, maxDeliver int,
	send func(domain.CustomerEvent) error) *Consumer {
	return &Consumer{js: js, dedup: dedup, maxDeliver: maxDeliver, send: send}
}

// Start creates the durable consumer and begins consuming. Returns a stop func.
func (c *Consumer) Start(ctx context.Context, stream jetstream.Stream, durable string) (func(), error) {
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    c.maxDeliver,
		AckWait:       5 * time.Second,
		FilterSubject: domain.SubjectCustomerCreated,
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
	var e domain.CustomerEvent
	if err := json.Unmarshal(msg.Data(), &e); err != nil {
		_ = msg.Term()
		return
	}
	if c.dedup.Seen(e.ID) {
		_ = msg.Ack()
		return
	}
	if err := c.send(e); err != nil {
		md, _ := msg.Metadata()
		if md != nil && int(md.NumDelivered) >= c.maxDeliver {
			_, _ = c.js.Publish(ctx, domain.SubjectDLQ, msg.Data())
			_ = msg.Term()
			return
		}
		_ = msg.Nak()
		return
	}
	c.dedup.Mark(e.ID)
	_ = msg.Ack()
}
