// Command demo shows JetStream durability + idempotency + replay, wired to
// YOUR packages. It won't build until you implement ./events and ./stream.
// Run (needs PULSE_NATS_URL): go run ./cmd/demo
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"cxm/m09/events"
	"cxm/m09/stream"
)

func main() {
	url := os.Getenv("PULSE_NATS_URL")
	if url == "" {
		log.Fatal("PULSE_NATS_URL is required")
	}
	ctx := context.Background()

	nc, js, err := stream.Connect(url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer nc.Drain()

	// Fresh stream for a clean demo.
	_ = js.DeleteStream(ctx, events.StreamName)
	st, err := stream.EnsureStream(ctx, js)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	var sends atomic.Int64
	dedup := stream.NewDedupStore()
	cons := stream.NewConsumer(js, dedup, 5, func(e events.CustomerEvent) error {
		sends.Add(1)
		fmt.Printf("  sent welcome -> %s (%s)\n", e.Name, e.Email)
		return nil
	})
	stop, err := cons.Start(ctx, st, "notify-workers")
	if err != nil {
		log.Fatalf("start: %v", err)
	}

	fmt.Println("publishing 5 events (one duplicated MsgID to show dedup)...")
	for i := 1; i <= 5; i++ {
		_ = stream.Publish(ctx, js, events.CustomerEvent{
			ID: fmt.Sprintf("cus_%02d", i), Email: fmt.Sprintf("u%d@pulse.dev", i),
			Name: fmt.Sprintf("User %d", i), OccurredAt: time.Now(),
		})
	}
	// Duplicate publish (same MsgID cus_03) -> stored once by JetStream dedup.
	_ = stream.Publish(ctx, js, events.CustomerEvent{ID: "cus_03", Email: "u3@pulse.dev", Name: "User 3"})

	time.Sleep(1 * time.Second)
	fmt.Printf("after first consumer: %d sends (expected 5, dup suppressed)\n\n", sends.Load())
	stop()

	// Replay: a NEW durable consumer (DeliverAll) re-reads ALL history. Because it
	// shares the dedup store, idempotency means zero duplicate sends.
	fmt.Println("starting REPLAY consumer (DeliverAll) sharing the dedup store...")
	var replaySends atomic.Int64
	replay := stream.NewConsumer(js, dedup, 5, func(e events.CustomerEvent) error {
		replaySends.Add(1) // should stay 0: every ID already marked
		return nil
	})
	rstop, err := replay.Start(ctx, st, "replay-audit")
	if err != nil {
		log.Fatalf("replay: %v", err)
	}
	defer rstop()

	time.Sleep(1 * time.Second)
	info, _ := st.Info(ctx)
	fmt.Printf("replay reprocessed history; duplicate sends=%d (idempotent)\n", replaySends.Load())
	fmt.Printf("stream stored messages: %d (5 unique; duplicate cus_03 suppressed by MsgID dedup)\n", info.State.Msgs)
}
