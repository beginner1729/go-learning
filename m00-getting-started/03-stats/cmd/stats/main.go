// Command stats summarizes the integers passed as arguments and exits non-zero
// on bad input — a tiny but complete CLI.
//
// Run: go run ./cmd/stats 3 1 4 1 5
package main

import (
	"fmt"
	"os"
	"strconv"

	"cxm/m00/stats/stats"
)

func main() {
	args := os.Args[1:] // os.Args[0] is the program name; skip it.
	xs := make([]int, 0, len(args))
	for _, a := range args {
		n, err := strconv.Atoi(a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %q: not an integer\n", a)
			continue
		}
		xs = append(xs, n)
	}

	s, err := stats.Summarize(xs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("count=%d sum=%d min=%d max=%d\n", s.Count, s.Sum, s.Min, s.Max)
}
