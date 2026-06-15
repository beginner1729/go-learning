// Command pulse runs the capstone vertical slice in one process:
// REST create -> JetStream durable event -> idempotent consumer -> welcome send.
// Run: go run ./cmd/pulse   (requires PULSE_NATS_URL)
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"time"

	"cxm/m14/api"
	"cxm/m14/domain"
	"cxm/m14/events"
	"cxm/m14/notify"
)

func main() {
	url := os.Getenv("PULSE_NATS_URL")
	if url == "" {
		log.Fatal("PULSE_NATS_URL is required")
	}
	ctx := context.Background()

	nc, js, err := events.Connect(url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer nc.Drain()

	_ = js.DeleteStream(ctx, domain.StreamName) // fresh for the demo
	stream, err := events.EnsureStream(ctx, js)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	// Notification side: dedup + consumer feeding a logging sender.
	sender := notify.NewLoggingSender(func(s string) { fmt.Println("  " + s) })
	dedup := events.NewDedupStore()
	consumer := events.NewConsumer(js, dedup, 5, func(e domain.CustomerEvent) error {
		return sender.Send(ctx, e)
	})
	stop, err := consumer.Start(ctx, stream, "notify-workers")
	if err != nil {
		log.Fatalf("consumer: %v", err)
	}
	defer stop()

	// REST side: handler publishing events.
	handler := api.NewHandler(events.NewPublisher(js)).Routes()

	fmt.Println("creating 5 customers via REST -> durable events -> welcome sends...")
	bodies := []string{
		`{"email":"ada@pulse.dev","name":"Ada"}`,
		`{"email":"bob@pulse.dev","name":"Bob"}`,
		`{"email":"cara@pulse.dev","name":"Cara"}`,
		`{"email":"dan@pulse.dev","name":"Dan"}`,
		`{"email":"eve@pulse.dev","name":"Eve"}`,
	}
	for _, b := range bodies {
		req := httptest.NewRequest("POST", "/v1/customers", bytes.NewBufferString(b))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 201 {
			log.Fatalf("create failed: %d %s", rec.Code, rec.Body)
		}
	}

	// Wait for the async notification chain to drain.
	deadline := time.Now().Add(5 * time.Second)
	for sender.Sent.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	info, _ := stream.Info(ctx)
	fmt.Printf("\n=== flow complete ===\n")
	fmt.Printf("welcome notifications sent: %d\n", sender.Sent.Load())
	fmt.Printf("events stored in stream:    %d\n", info.State.Msgs)
}
