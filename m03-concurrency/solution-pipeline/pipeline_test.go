package pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestRun_OrderPreservedAndConcurrent(t *testing.T) {
	items := make([]int, 50)
	for i := range items {
		items[i] = i
	}
	var inFlightMax int64
	var inFlight int64
	got, err := Run(context.Background(), 8, items, func(_ context.Context, x int) (int, error) {
		cur := atomic.AddInt64(&inFlight, 1)
		for {
			old := atomic.LoadInt64(&inFlightMax)
			if cur <= old || atomic.CompareAndSwapInt64(&inFlightMax, old, cur) {
				break
			}
		}
		atomic.AddInt64(&inFlight, -1)
		return x * 2, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		if got[i] != i*2 {
			t.Fatalf("order broken at %d: got %d", i, got[i])
		}
	}
	if inFlightMax < 2 {
		t.Fatalf("expected real concurrency, max in-flight=%d", inFlightMax)
	}
}

func TestRun_FirstErrorCancels(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	sentinel := errors.New("boom")
	_, err := Run(context.Background(), 3, items, func(_ context.Context, x int) (int, error) {
		if x == 3 {
			return 0, sentinel
		}
		return x, nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel, got %v", err)
	}
}

func TestMerge(t *testing.T) {
	mk := func(vals ...int) <-chan int {
		c := make(chan int, len(vals))
		for _, v := range vals {
			c <- v
		}
		close(c)
		return c
	}
	sum := 0
	for v := range Merge(mk(1, 2), mk(3, 4), mk(5)) {
		sum += v
	}
	if sum != 15 {
		t.Fatalf("merge sum = %d, want 15", sum)
	}
}
