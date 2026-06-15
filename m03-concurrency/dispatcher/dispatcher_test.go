package dispatcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// flakySender fails the first `failTimes` attempts per ID, then succeeds.
// IDs containing "dead" always fail (to exercise the dead-letter path).
type flakySender struct {
	mu        sync.Mutex
	attempts  map[string]int
	failTimes int
}

func (s *flakySender) Send(_ context.Context, n Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[n.ID]++
	if n.CustomerID == "dead" {
		return errors.New("permanent failure")
	}
	if s.attempts[n.ID] <= s.failTimes {
		return errors.New("transient failure")
	}
	return nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDispatcher_RetriesThenSucceeds(t *testing.T) {
	sender := &flakySender{attempts: map[string]int{}, failTimes: 1}
	d := New(sender,
		WithWorkers(4),
		WithMaxAttempts(3),
		WithBackoff(time.Millisecond),
		WithLogger(quietLogger()),
	)

	in := make(chan Notification)
	// Drain dead letters so the channel never blocks.
	var deadCount int64
	doneDrain := make(chan struct{})
	go func() {
		for range d.DeadLetters() {
			atomic.AddInt64(&deadCount, 1)
		}
		close(doneDrain)
	}()

	go func() {
		for i := 0; i < 10; i++ {
			in <- Notification{ID: string(rune('A' + i)), CustomerID: "c1", Channel: "email"}
		}
		close(in)
	}()

	d.Run(context.Background(), in)
	<-doneDrain

	if got := d.Metrics.Sent.Load(); got != 10 {
		t.Fatalf("sent = %d, want 10", got)
	}
	if deadCount != 0 {
		t.Fatalf("dead = %d, want 0", deadCount)
	}
	if got := d.Metrics.Failed.Load(); got != 10 { // one transient failure each
		t.Fatalf("failed attempts = %d, want 10", got)
	}
}

func TestDispatcher_DeadLetters(t *testing.T) {
	sender := &flakySender{attempts: map[string]int{}, failTimes: 0}
	d := New(sender, WithWorkers(2), WithMaxAttempts(3),
		WithBackoff(time.Millisecond), WithLogger(quietLogger()))

	in := make(chan Notification)
	var dead int64
	done := make(chan struct{})
	go func() {
		for range d.DeadLetters() {
			atomic.AddInt64(&dead, 1)
		}
		close(done)
	}()
	go func() {
		for i := 0; i < 5; i++ {
			in <- Notification{ID: string(rune('A' + i)), CustomerID: "dead"}
		}
		close(in)
	}()

	d.Run(context.Background(), in)
	<-done

	if dead != 5 {
		t.Fatalf("dead-lettered = %d, want 5", dead)
	}
	if d.Metrics.Sent.Load() != 0 {
		t.Fatalf("sent should be 0")
	}
}

func TestDispatcher_GracefulCancel(t *testing.T) {
	sender := &flakySender{attempts: map[string]int{}, failTimes: 0}
	d := New(sender, WithWorkers(2), WithMaxAttempts(2),
		WithBackoff(50*time.Millisecond), WithLogger(quietLogger()))

	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan Notification)
	go func() { for range d.DeadLetters() { } }()
	go func() {
		// keep offering work; cancellation should make Run return promptly
		for i := 0; ; i++ {
			select {
			case in <- Notification{ID: "x", CustomerID: "dead"}:
			case <-ctx.Done():
				close(in)
				return
			}
		}
	}()

	cancel()
	doneCh := make(chan struct{})
	go func() { d.Run(ctx, in); close(doneCh) }()
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return promptly after cancel")
	}
}
