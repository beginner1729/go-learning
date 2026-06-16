// Package stats computes simple summary statistics over a slice of ints.
// It shows a package that returns both a value and an error.
package stats

import "errors"

// ErrEmpty is a sentinel error returned when there is nothing to summarize.
var ErrEmpty = errors.New("stats: no values")

// Summary holds the results of one pass over the data.
type Summary struct {
	Count int
	Sum   int
	Min   int
	Max   int
}

// Summarize returns count/sum/min/max for xs, or ErrEmpty if xs is empty.
func Summarize(xs []int) (Summary, error) {
	if len(xs) == 0 {
		return Summary{}, ErrEmpty
	}
	s := Summary{Count: len(xs), Min: xs[0], Max: xs[0]}
	for _, x := range xs {
		s.Sum += x
		if x < s.Min {
			s.Min = x
		}
		if x > s.Max {
			s.Max = x
		}
	}
	return s, nil
}
