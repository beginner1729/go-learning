// Package dispatcher is YOUR implementation target for M03 §5 ("Implement it
// yourself"): Pulse's notification dispatcher. A bounded worker pool that sends
// notifications with retry/backoff, dead-letters permanent failures, exposes
// atomic metrics, and drains gracefully on cancel.
//
// Goal: make `go test ./dispatcher/...` pass (run it with -race) and
// `go run ./cmd/demo` work. The tests in dispatcher_test.go define the exact
// API you must provide. Reference answer key: ../solution-dispatcher/ (and §5
// in ../M03-concurrency.md). Try it yourself before peeking.
//
// Build here:
//
//   - Notification struct — the unit of work: ID, CustomerID, Channel, Body
//     (all string).
//
//   - Sender interface { Send(ctx context.Context, n Notification) error }.
//     A returned error means "retryable failure" for this exercise.
//
//   - Metrics struct of race-free counters readable at any time, using
//     sync/atomic: Sent, Failed (individual attempts that errored),
//     DeadLettered (notifications that exhausted all retries). atomic.Int64.
//
//   - Functional options (the M2 pattern): type Option func(*Dispatcher), with
//     WithWorkers(n int) Option
//     WithMaxAttempts(n int) Option
//     WithBackoff(d time.Duration) Option
//     WithLogger(l *slog.Logger) Option
//
//   - New(sender Sender, opts ...Option) *Dispatcher — sensible defaults
//     (e.g. 4 workers, 3 max attempts, ~10ms base backoff, slog.Default()),
//     a buffered dead-letter channel, then apply opts.
//
//   - (*Dispatcher).DeadLetters() <-chan Notification — exposes the
//     dead-letter channel for a drainer to read. It must be drained or it can
//     fill and block.
//
//   - (*Dispatcher).Run(ctx context.Context, in <-chan Notification) — start
//     `workers` goroutines reading from `in`. Each notification is retried up
//     to maxAttempts with EXPONENTIAL backoff (base * 2^(attempt-1)) inside a
//     select that also watches ctx.Done() (backoff must stay cancellable). On
//     success bump Metrics.Sent; on each errored attempt bump Metrics.Failed;
//     on exhausting retries bump Metrics.DeadLettered and push onto the
//     dead-letter channel. Run blocks until all workers exit — either because
//     `in` was closed (drain in-flight work, clean exit) OR ctx was cancelled
//     (graceful stop) — then closes the dead-letter channel exactly once.
//
// What the tests pin down:
//   - TestDispatcher_RetriesThenSucceeds: with one transient failure per ID,
//     10 inputs => Sent==10, Failed==10, 0 dead-lettered.
//   - TestDispatcher_DeadLetters: always-failing sends => 5 inputs all
//     dead-lettered, Sent==0.
//   - TestDispatcher_GracefulCancel: after cancel, Run returns promptly
//     (well under 2s) even while work is still being offered.
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package dispatcher

// TODO(§5): implement Notification, Sender, Metrics, Option, the With* options,
// New, Dispatcher, DeadLetters, and Run.
