// Package pipeline provides reusable concurrency primitives: a bounded worker
// pool with cancellation + first-error semantics, and a fan-in merge.
package pipeline

import (
	"context"
	"sync"
)

// Run processes items with `workers` concurrent goroutines. It preserves output
// order, stops early on the first error (cancelling the rest), and returns that
// error. A hand-rolled equivalent of golang.org/x/sync/errgroup for teaching.
func Run[T, R any](ctx context.Context, workers int, items []T,
	work func(context.Context, T) (R, error)) ([]R, error) {

	if workers < 1 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		idx int
		val R
	}
	jobs := make(chan int)
	out := make(chan result)
	errCh := make(chan error, 1) // first error wins; later ones dropped

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				v, err := work(ctx, items[i])
				if err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
				select {
				case out <- result{i, v}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i := range items {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() { wg.Wait(); close(out) }()

	results := make([]R, len(items))
	for r := range out {
		results[r.idx] = r.val
	}
	select {
	case err := <-errCh:
		return nil, err
	default:
		return results, nil
	}
}

// Merge fans multiple input channels into one, closing the output after every
// input is drained.
func Merge[T any](chans ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	wg.Add(len(chans))
	for _, c := range chans {
		go func(c <-chan T) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(c)
	}
	go func() { wg.Wait(); close(out) }()
	return out
}
