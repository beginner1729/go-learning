// Package notify is the send side: a Sender abstraction and a counter so the
// e2e test and demo can observe how many welcome messages were delivered.
package notify

import (
	"context"
	"fmt"
	"sync/atomic"

	"cxm/m14/domain"
)

type Sender interface {
	Send(ctx context.Context, e domain.CustomerEvent) error
}

// LoggingSender prints and counts sends (stands in for email/SMS).
type LoggingSender struct {
	Sent atomic.Int64
	out  func(string)
}

func NewLoggingSender(out func(string)) *LoggingSender {
	return &LoggingSender{out: out}
}

func (s *LoggingSender) Send(_ context.Context, e domain.CustomerEvent) error {
	s.Sent.Add(1)
	if s.out != nil {
		s.out(fmt.Sprintf("welcome sent -> %s (%s)", e.Name, e.Email))
	}
	return nil
}
