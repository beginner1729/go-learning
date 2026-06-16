// Command demo runs the notification dispatcher with a simulated, flaky sender
// and prints the resulting metrics.
//
// This wires together YOUR package (./dispatcher). It will not compile until
// you have implemented it. Run it once it does:
//
//	go run ./cmd/demo
//
// Compare against the reference: go run ./solution-cmd/demo
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"cxm/m03/dispatcher"
)

// simSender simulates network latency and a ~25% transient failure rate.
type simSender struct {
	r  *rand.Rand
	mu sync.Mutex
}

func (s *simSender) Send(ctx context.Context, n dispatcher.Notification) error {
	select {
	case <-time.After(5 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}
	s.mu.Lock()
	fail := s.r.Float64() < 0.25
	s.mu.Unlock()
	if fail {
		return fmt.Errorf("simulated transient failure")
	}
	return nil
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	sender := &simSender{r: rand.New(rand.NewSource(42))}

	d := dispatcher.New(sender,
		dispatcher.WithWorkers(6),
		dispatcher.WithMaxAttempts(4),
		dispatcher.WithBackoff(5*time.Millisecond),
		dispatcher.WithLogger(log),
	)

	// Drain dead letters concurrently.
	go func() {
		for dl := range d.DeadLetters() {
			log.Warn("dead-lettered", "id", dl.ID, "customer", dl.CustomerID)
		}
	}()

	in := make(chan dispatcher.Notification)
	go func() {
		defer close(in)
		for i := 0; i < 50; i++ {
			in <- dispatcher.Notification{
				ID:         fmt.Sprintf("n%03d", i),
				CustomerID: fmt.Sprintf("c%03d", i%10),
				Channel:    "email",
				Body:       "Welcome to Pulse!",
			}
		}
	}()

	start := time.Now()
	d.Run(context.Background(), in)

	fmt.Printf("\n=== dispatch complete in %s ===\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("sent=%d  failed_attempts=%d  dead_lettered=%d\n",
		d.Metrics.Sent.Load(), d.Metrics.Failed.Load(), d.Metrics.DeadLettered.Load())
}
