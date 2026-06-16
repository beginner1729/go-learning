// Package genx is YOUR implementation target for M02 Exercises 3.1 & 3.2.
//
// Goal: make `go test ./genx/...` pass and `go run ./cmd/demo` work. The tests
// in genx_test.go define the exact API. Reference answer key: ../solution-genx/
// (and §3 of ../M02-generics-and-idioms.md). Try it yourself before peeking.
//
// Build here — small, reusable generic helpers:
//
//   - `Map[T, U any](s []T, f func(T) U) []U` — apply f to each element,
//     returning a new slice of the result type.
//
//   - `Filter[T any](s []T, keep func(T) bool) []T` — return only elements for
//     which keep returns true.
//
//   - `Keys[K comparable, V any](m map[K]V) []K` — the map's keys (any order).
//
//   - `Values[K comparable, V any](m map[K]V) []V` — the map's values (any order).
//
//   - `Set[T comparable]` (Exercise 3.1) — a set backed by `map[T]struct{}`
//     (zero-byte values). Constructor `NewSet[T comparable](items ...T) *Set[T]`
//     is required because a nil map can't be written to. Methods:
//     Add(v T), Remove(v T), Has(v T) bool, Len() int, Items() []T.
//     NewSet must dedupe its inputs (NewSet("a","b","a") has Len 2).
//
//   - `Page[T any]` (Exercise 3.2) — a paginated window plus metadata:
//     Items []T, Page, Size, Total, TotalPages int, HasNext bool.
//
//   - `Paginate[T any](items []T, page, size int) Page[T]` — return the
//     requested window, clamping invalid inputs: page < 1 -> 1, size < 1 -> 20.
//     Clamp the start/end indices so an out-of-range page yields an empty slice
//     (no panic). TotalPages is ceil(total/size); HasNext is page < totalPages.
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package genx

// TODO(3.1/3.2): implement Map, Filter, Keys, Values, Set + NewSet, Page + Paginate.
