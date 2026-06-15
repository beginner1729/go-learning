package stream_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"cxm/m09/events"
	"cxm/m09/stream"
)

func mustJS(t *testing.T) (jetstream.JetStream, func()) {
	t.Helper()
	url := os.Getenv("PULSE_NATS_URL")
	if url == "" {
		t.Skip("PULSE_NATS_URL not set; skipping JetStream integration test")
	}
	nc, js, err := stream.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Fresh stream per test for isolation.
	_ = js.DeleteStream(context.Background(), events.StreamName)
	return js, func() { nc.Drain() }
}

func TestDurableDelivery(t *testing.T) {
	js, cleanup := mustJS(t)
	defer cleanup()
	ctx := context.Background()

	st, err := stream.EnsureStream(ctx, js)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	var sent atomic.Int64
	dedup := stream.NewDedupStore()
	cons := stream.NewConsumer(js, dedup, 5, func(e events.CustomerEvent) error {
		sent.Add(1)
		return nil
	})
	stop, err := cons.Start(ctx, st, "notify-test-1")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer stop()

	for i := 0; i < 10; i++ {
		if err := stream.Publish(ctx, js, events.CustomerEvent{
			ID: fmt.Sprintf("cus_%02d", i), Email: "x@y.com", Name: "X", OccurredAt: time.Now(),
		}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	waitFor(t, func() bool { return sent.Load() == 10 }, 5*time.Second, "10 sends")
}

func TestRetryThenSucceed(t *testing.T) {
	js, cleanup := mustJS(t)
	defer cleanup()
	ctx := context.Background()
	st, _ := stream.EnsureStream(ctx, js)

	var attempts sync.Map // id -> count
	var sent atomic.Int64
	dedup := stream.NewDedupStore()
	cons := stream.NewConsumer(js, dedup, 5, func(e events.CustomerEvent) error {
		v, _ := attempts.LoadOrStore(e.ID, new(atomic.Int64))
		n := v.(*atomic.Int64).Add(1)
		if n < 3 {
			return fmt.Errorf("transient")
		}
		sent.Add(1)
		return nil
	})
	stop, err := cons.Start(ctx, st, "notify-test-retry")
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	_ = stream.Publish(ctx, js, events.CustomerEvent{ID: "retry_1", Email: "a@b.com", Name: "R"})
	waitFor(t, func() bool { return sent.Load() == 1 }, 8*time.Second, "1 send after retries")
}

func TestDeadLetter(t *testing.T) {
	js, cleanup := mustJS(t)
	defer cleanup()
	ctx := context.Background()
	st, _ := stream.EnsureStream(ctx, js)

	// A separate consumer on the DLQ subject to observe parked messages.
	dlqCount := make(chan struct{}, 8)
	dlqCons, err := st.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable: "dlq-watch", AckPolicy: jetstream.AckExplicitPolicy,
		FilterSubject: events.SubjectDLQ,
	})
	if err != nil {
		t.Fatal(err)
	}
	dcc, _ := dlqCons.Consume(func(m jetstream.Msg) { m.Ack(); dlqCount <- struct{}{} })
	defer dcc.Stop()

	dedup := stream.NewDedupStore()
	cons := stream.NewConsumer(js, dedup, 3, func(e events.CustomerEvent) error {
		return fmt.Errorf("always fails") // poison message
	})
	stop, _ := cons.Start(ctx, st, "notify-test-dlq")
	defer stop()

	_ = stream.Publish(ctx, js, events.CustomerEvent{ID: "poison_1", Email: "a@b.com", Name: "P"})

	select {
	case <-dlqCount:
		// dead-lettered as expected
	case <-time.After(15 * time.Second):
		t.Fatal("message was not dead-lettered")
	}
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration, what string) {
	t.Helper()
	deadline := time.After(timeout)
	for !cond() {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		case <-time.After(20 * time.Millisecond):
		}
	}
}
