package config

import (
	"testing"
	"time"
)

func TestDefaultsAndOverrides(t *testing.T) {
	c := New(":8080")
	if c.ReadTimeout != 15*time.Second {
		t.Fatalf("default read timeout wrong: %v", c.ReadTimeout)
	}

	c2 := New(":9090", WithReadTimeout(5*time.Second), WithShutdownTimeout(time.Minute))
	if c2.ReadTimeout != 5*time.Second || c2.ShutdownTimeout != time.Minute {
		t.Fatalf("overrides not applied: %+v", c2)
	}
	if c2.WriteTimeout != 15*time.Second {
		t.Fatalf("untouched option lost its default: %v", c2.WriteTimeout)
	}
}
