// Package notify is a small unit-under-test: a Notifier that depends on a Sender
// interface, so tests can substitute a hand-written fake.
package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Customer struct {
	Email string
	Name  string
}

type Message struct {
	To   string
	Body string
}

// Sender is the dependency boundary we fake in tests.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

type Notifier struct{ sender Sender }

func New(sender Sender) *Notifier { return &Notifier{sender: sender} }

// Welcome validates the customer, then sends a welcome message.
func (n *Notifier) Welcome(ctx context.Context, c Customer) error {
	if !strings.Contains(c.Email, "@") {
		return fmt.Errorf("invalid email %q", c.Email)
	}
	return n.sender.Send(ctx, Message{
		To:   c.Email,
		Body: fmt.Sprintf("Welcome to Pulse, %s!", c.Name),
	})
}

// SendWithRetry retries a transient send up to maxAttempts times.
func SendWithRetry(ctx context.Context, s Sender, m Message, maxAttempts int) error {
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = s.Send(ctx, m); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond): // tiny backoff; keep tests fast
		}
	}
	return errors.New("exhausted retries: " + err.Error())
}
