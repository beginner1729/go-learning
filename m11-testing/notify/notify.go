// Package notify is YOUR implementation target for M11 (Testing & Quality Gates).
//
// Goal: make `go test ./notify/...` pass. The tests in notify_test.go define the
// exact API you must provide — they are the spec. Reference answer key:
// ../solution-notify/ (and §3 "Hands-on exercises" + §4 "Fill-in-the-blank" in
// ../M11-testing.md). Try it yourself before peeking.
//
// This package is the unit-under-test: a Notifier that depends on a Sender
// interface, so tests can substitute a hand-written fake. Build, in THIS file
// (notify.go):
//
//   - `Customer` struct with fields `Email string` and `Name string`.
//
//   - `Message` struct with fields `To string` and `Body string`.
//
//   - `Sender` interface — the dependency boundary the tests fake:
//     Send(ctx context.Context, m Message) error
//
//   - `Notifier` struct holding a Sender, plus a constructor
//     `New(sender Sender) *Notifier`.
//
//   - Method `(*Notifier) Welcome(ctx context.Context, c Customer) error`:
//     validate the customer first — if c.Email does not contain "@", return a
//     non-nil error WITHOUT calling Sender. Otherwise call sender.Send with a
//     Message addressed To: c.Email and a welcome Body (e.g.
//     fmt.Sprintf("Welcome to Pulse, %s!", c.Name)), returning the Sender's
//     error verbatim. (Tests assert: valid customer => 1 send, no error; a
//     Sender failure => 0 recorded sends, error; invalid email => 0 sends,
//     error, Sender never touched.)
//
//   - Function `SendWithRetry(ctx context.Context, s Sender, m Message,
//     maxAttempts int) error`: call s.Send up to maxAttempts times; return nil
//     on the first success. Between attempts, honor ctx cancellation (return
//     ctx.Err() if ctx is Done) and apply a tiny backoff (e.g.
//     time.After(time.Millisecond) — keep tests fast). If all attempts fail,
//     return a non-nil error. (Tests assert: 0 prior failures => ok; 2 prior
//     failures with maxAttempts 3 => ok; 5 prior failures with maxAttempts 3 =>
//     error.)
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package notify

// TODO: implement Customer, Message, Sender, Notifier, New, (*Notifier).Welcome,
// and SendWithRetry.
