package bus_test

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bus "cxm/m08/solution-bus"
	events "cxm/m08/solution-events"
)

func mustBus(t *testing.T, name string) *bus.Bus {
	t.Helper()
	url := os.Getenv("PULSE_NATS_URL")
	if url == "" {
		t.Skip("PULSE_NATS_URL not set; skipping NATS integration test")
	}
	b, err := bus.Connect(url, name)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = b.Drain() })
	return b
}

func TestPubSubTyped(t *testing.T) {
	b := mustBus(t, "test-pubsub")
	got := make(chan events.CustomerEvent, 1)
	sub, err := bus.Subscribe(b, events.SubjectCustomerCreated, func(e events.CustomerEvent) {
		got <- e
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	_ = b.Publish(events.SubjectCustomerCreated, events.CustomerEvent{ID: "cus_1", Email: "a@b.com"})
	_ = b.Flush()

	select {
	case e := <-got:
		if e.ID != "cus_1" {
			t.Fatalf("got %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestQueueGroupExactlyOnce(t *testing.T) {
	b := mustBus(t, "test-queue")

	var total atomic.Int64
	var mu sync.Mutex
	perWorker := map[int]int{}

	for w := 0; w < 3; w++ {
		w := w
		sub, err := bus.QueueSubscribe(b, "cxm.work", "g", func(_ events.CustomerEvent) {
			mu.Lock()
			perWorker[w]++
			mu.Unlock()
			total.Add(1)
		})
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Unsubscribe()
	}

	for i := 0; i < 30; i++ {
		_ = b.Publish("cxm.work", events.CustomerEvent{ID: "x"})
	}
	_ = b.Flush()

	deadline := time.After(2 * time.Second)
	for total.Load() < 30 {
		select {
		case <-deadline:
			t.Fatalf("only %d/30 handled", total.Load())
		case <-time.After(10 * time.Millisecond):
		}
	}
	if total.Load() != 30 {
		t.Fatalf("exactly-once violated: %d", total.Load())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(perWorker) < 2 {
		t.Fatalf("expected load distribution, got %v", perWorker)
	}
}

func TestRequestReply(t *testing.T) {
	b := mustBus(t, "test-reqrep")
	sub, err := b.ServeReply(events.SubjectProfileLookup, func(req []byte) []byte {
		return []byte("profile-for-" + string(req))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	resp, err := b.Request(events.SubjectProfileLookup, []byte("cus_1"), 2*time.Second)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if string(resp) != "profile-for-cus_1" {
		t.Fatalf("got %q", resp)
	}
}
