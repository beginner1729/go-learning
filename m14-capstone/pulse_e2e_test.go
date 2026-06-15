package main_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cxm/m14/api"
	"cxm/m14/domain"
	"cxm/m14/events"
	"cxm/m14/notify"
)

// TestPulseEndToEnd proves the full flow: a REST create produces a JetStream
// event that the durable consumer turns into a welcome send.
func TestPulseEndToEnd(t *testing.T) {
	url := os.Getenv("PULSE_NATS_URL")
	if url == "" {
		t.Skip("PULSE_NATS_URL not set; skipping capstone e2e test")
	}
	ctx := context.Background()

	nc, js, err := events.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { nc.Drain() })

	_ = js.DeleteStream(ctx, domain.StreamName)
	stream, err := events.EnsureStream(ctx, js)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}

	sender := notify.NewLoggingSender(nil)
	dedup := events.NewDedupStore()
	consumer := events.NewConsumer(js, dedup, 5, func(e domain.CustomerEvent) error {
		return sender.Send(ctx, e)
	})
	stop, err := consumer.Start(ctx, stream, "notify-e2e")
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	t.Cleanup(stop)

	handler := api.NewHandler(events.NewPublisher(js)).Routes()

	// Create 3 customers; one duplicate publish is impossible via REST (new IDs),
	// so we assert exactly 3 sends.
	for _, body := range []string{
		`{"email":"a@pulse.dev","name":"A"}`,
		`{"email":"b@pulse.dev","name":"B"}`,
		`{"email":"c@pulse.dev","name":"C"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/customers", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status = %d (%s)", rec.Code, rec.Body)
		}
	}

	// Reject an invalid create (no event should be produced).
	bad := httptest.NewRequest(http.MethodPost, "/v1/customers", bytes.NewBufferString(`{"email":"nope"}`))
	badRec := httptest.NewRecorder()
	handler.ServeHTTP(badRec, bad)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d, want 400", badRec.Code)
	}

	// Wait for the async chain.
	deadline := time.Now().Add(8 * time.Second)
	for sender.Sent.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := sender.Sent.Load(); got != 3 {
		t.Fatalf("welcome sends = %d, want 3", got)
	}
}
