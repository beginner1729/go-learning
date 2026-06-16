// Package dispatcher implements Pulse's notification dispatcher: a bounded
// worker pool that sends notifications with retry/backoff, dead-letters
// permanent failures, exposes atomic metrics, and drains gracefully on cancel.
package dispatcher

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Notification is the unit of work. In M9 these arrive from JetStream.
type Notification struct {
	ID         string
	CustomerID string
	Channel    string // "email", "sms", ...
	Body       string
}

// Sender abstracts the delivery mechanism (email/SMS/push). Returning an error
// means "retryable failure" for this teaching example.
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// Metrics are race-free counters readable at any time.
type Metrics struct {
	Sent         atomic.Int64
	Failed       atomic.Int64 // individual send attempts that errored
	DeadLettered atomic.Int64 // notifications that exhausted all retries
}

type Dispatcher struct {
	workers     int
	sender      Sender
	maxAttempts int
	baseBackoff time.Duration
	log         *slog.Logger

	Metrics Metrics
	dead    chan Notification
}

// Option configures a Dispatcher (functional options from M2).
type Option func(*Dispatcher)

func WithWorkers(n int) Option     { return func(d *Dispatcher) { d.workers = n } }
func WithMaxAttempts(n int) Option { return func(d *Dispatcher) { d.maxAttempts = n } }
func WithBackoff(d2 time.Duration) Option {
	return func(d *Dispatcher) { d.baseBackoff = d2 }
}
func WithLogger(l *slog.Logger) Option { return func(d *Dispatcher) { d.log = l } }

func New(sender Sender, opts ...Option) *Dispatcher {
	d := &Dispatcher{
		workers:     4,
		sender:      sender,
		maxAttempts: 3,
		baseBackoff: 10 * time.Millisecond,
		log:         slog.Default(),
		dead:        make(chan Notification, 64),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// DeadLetters yields notifications that exhausted their retries. Drain it
// (e.g. log/persist) or it can fill and block once full.
func (d *Dispatcher) DeadLetters() <-chan Notification { return d.dead }

// Run consumes from in until it is closed (then drains in-flight work) or until
// ctx is cancelled (graceful stop). It blocks until all workers exit, then
// closes the dead-letter channel.
func (d *Dispatcher) Run(ctx context.Context, in <-chan Notification) {
	var wg sync.WaitGroup
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			d.worker(ctx, in)
		}(i)
	}
	wg.Wait()
	close(d.dead)
}

func (d *Dispatcher) worker(ctx context.Context, in <-chan Notification) {
	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-in:
			if !ok {
				return // input drained: clean exit
			}
			d.process(ctx, n)
		}
	}
}

func (d *Dispatcher) process(ctx context.Context, n Notification) {
	for attempt := 0; attempt < d.maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff, but always cancellable.
			backoff := d.baseBackoff * (1 << (attempt - 1))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
		err := d.sender.Send(ctx, n)
		if err == nil {
			d.Metrics.Sent.Add(1)
			return
		}
		d.Metrics.Failed.Add(1)
		d.log.Debug("send failed", "id", n.ID, "attempt", attempt+1, "err", err)
	}
	// Exhausted retries -> dead-letter (respect cancellation while enqueuing).
	d.Metrics.DeadLettered.Add(1)
	select {
	case d.dead <- n:
	case <-ctx.Done():
	}
}
