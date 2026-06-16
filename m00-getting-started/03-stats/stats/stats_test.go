package stats

import (
	"errors"
	"testing"
)

// Table-driven test: one slice of cases, one loop, a subtest per case.
// This is the idiomatic Go testing style used throughout the course.
func TestSummarize(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want Summary
	}{
		{"single", []int{5}, Summary{Count: 1, Sum: 5, Min: 5, Max: 5}},
		{"many", []int{3, 1, 4, 1, 5}, Summary{Count: 5, Sum: 14, Min: 1, Max: 5}},
		{"negatives", []int{-2, 0, 2}, Summary{Count: 3, Sum: 0, Min: -2, Max: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Summarize(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Summarize(%v) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSummarize_Empty(t *testing.T) {
	if _, err := Summarize(nil); !errors.Is(err, ErrEmpty) {
		t.Fatalf("want ErrEmpty, got %v", err)
	}
}
