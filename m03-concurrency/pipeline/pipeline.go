// Package pipeline is YOUR implementation target for M03 Exercises 3.1 & 3.2.
//
// Goal: make `go test ./pipeline/...` pass. The tests in pipeline_test.go
// define the exact API you must provide. Reference answer key:
// ../solution-pipeline/ (and §3.1 / §3.2 in ../M03-concurrency.md). Try it
// yourself before peeking.
//
// Build here:
//
//   - Run[T, R any](ctx context.Context, workers int, items []T,
//     work func(context.Context, T) (R, error)) ([]R, error) — Exercise 3.1.
//
//     A bounded worker pool: process `items` with `workers` concurrent
//     goroutines, PRESERVING output order (index results so completion order
//     doesn't matter), stop early on the FIRST error (cancelling the rest via
//     a derived context.WithCancel + defer cancel()), and return that error.
//     Treat workers < 1 as 1. The hand-rolled equivalent of
//     golang.org/x/sync/errgroup.
//
//     What the tests pin down:
//
//   - TestRun_OrderPreservedAndConcurrent: results come back in input order
//     (got[i] == i*2) and real concurrency happens (max in-flight >= 2).
//
//   - TestRun_FirstErrorCancels: the returned error matches the sentinel
//     the work func returned (errors.Is).
//
//     Idioms: a `jobs` channel of indices fed by one goroutine; an `out`
//     channel of {idx, val} drained into a pre-sized results slice; an
//     errCh with cap 1 written via select/default so only the first error
//     wins and no worker blocks; a closer goroutine doing wg.Wait(); close(out).
//
//   - Merge[T any](chans ...<-chan T) <-chan T — Exercise 3.2.
//
//     Fan-in: forward every value from every input channel onto one output
//     channel, and close the output EXACTLY ONCE after all inputs are drained.
//     One goroutine per input + a WaitGroup + a closer goroutine.
//
//     What the test pins down:
//
//   - TestMerge: every value from every input arrives (sum == 15).
//
// Delete this comment block as you implement. The package will not compile
// until Run and Merge exist.
package pipeline

// TODO(3.1): implement Run[T, R].
// TODO(3.2): implement Merge[T].
