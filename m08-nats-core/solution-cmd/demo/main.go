// Command demo shows core NATS: a publisher, a queue group of workers consuming
// customer.created events, and a request-reply lookup.
// This is the reference (answer-key) demo wired to the solution packages.
// Run: go run ./solution-cmd/demo
package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	bus "cxm/m08/solution-bus"
	events "cxm/m08/solution-events"
)

func main() {
	url := os.Getenv("PULSE_NATS_URL")
	if url == "" {
		log.Fatal("PULSE_NATS_URL is required")
	}

	b, err := bus.Connect(url, "pulse-demo")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer b.Drain()

	// Two notification workers in one queue group: each event handled once.
	var handled atomic.Int64
	var mu sync.Mutex
	byWorker := map[int]int{}
	for w := 1; w <= 2; w++ {
		w := w
		_, _ = bus.QueueSubscribe(b, events.SubjectCustomerCreated, "notify-workers",
			func(e events.CustomerEvent) {
				mu.Lock()
				byWorker[w]++
				mu.Unlock()
				handled.Add(1)
				fmt.Printf("[worker %d] welcome notification -> %s (%s)\n", w, e.Name, e.Email)
			})
	}

	// Request-reply responder (stands in for profile-svc).
	_, _ = b.ServeReply(events.SubjectProfileLookup, func(req []byte) []byte {
		return []byte(`{"customer":"` + string(req) + `","tier":"gold"}`)
	})

	// Publish a batch of customer.created events.
	for i := 1; i <= 6; i++ {
		_ = b.Publish(events.SubjectCustomerCreated, events.CustomerEvent{
			ID:         fmt.Sprintf("cus_%02d", i),
			Email:      fmt.Sprintf("user%d@pulse.dev", i),
			Name:       fmt.Sprintf("User %d", i),
			OccurredAt: time.Now(),
		})
	}
	_ = b.Flush()

	// Synchronous lookup.
	resp, err := b.Request(events.SubjectProfileLookup, []byte("cus_01"), 2*time.Second)
	if err != nil {
		log.Fatalf("lookup: %v", err)
	}

	time.Sleep(300 * time.Millisecond) // let async handlers run
	fmt.Printf("\nlookup reply: %s\n", resp)
	fmt.Printf("events handled=%d distributed across workers=%v\n", handled.Load(), byWorker)
}
