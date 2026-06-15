package notify_test

import (
	"context"
	"errors"
	"testing"

	"cxm/m11/notify"
)

// fakeSender is a hand-written test double implementing notify.Sender.
type fakeSender struct {
	sent      []notify.Message
	failTimes int // fail this many calls before succeeding
	calls     int
}

func (f *fakeSender) Send(_ context.Context, m notify.Message) error {
	f.calls++
	if f.calls <= f.failTimes {
		return errors.New("transient send failure")
	}
	f.sent = append(f.sent, m)
	return nil
}

func newFixture(t *testing.T) (*notify.Notifier, *fakeSender) {
	t.Helper()
	f := &fakeSender{}
	return notify.New(f), f
}

func TestNotifier_Welcome(t *testing.T) {
	tests := []struct {
		name      string
		cust      notify.Customer
		failTimes int
		wantErr   bool
		wantSends int
	}{
		{"ok", notify.Customer{Email: "a@b.com", Name: "Ada"}, 0, false, 1},
		{"send fails", notify.Customer{Email: "a@b.com", Name: "Ada"}, 1, true, 0},
		{"invalid email rejected before send", notify.Customer{Email: "x", Name: "Ada"}, 0, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeSender{failTimes: tt.failTimes}
			n := notify.New(f)
			err := n.Welcome(context.Background(), tt.cust)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if len(f.sent) != tt.wantSends {
				t.Fatalf("sends = %d, want %d", len(f.sent), tt.wantSends)
			}
		})
	}
}

func TestSendWithRetry(t *testing.T) {
	cases := []struct {
		name    string
		fails   int
		wantErr bool
	}{
		{"succeeds first try", 0, false},
		{"succeeds after retries", 2, false},
		{"exhausts retries", 5, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeSender{failTimes: tt.fails}
			err := notify.SendWithRetry(context.Background(), f, notify.Message{To: "a@b.com"}, 3)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFixtureHelper(t *testing.T) {
	n, f := newFixture(t)
	if err := n.Welcome(context.Background(), notify.Customer{Email: "z@b.com", Name: "Z"}); err != nil {
		t.Fatal(err)
	}
	if len(f.sent) != 1 || f.sent[0].To != "z@b.com" {
		t.Fatalf("unexpected sends: %+v", f.sent)
	}
}
