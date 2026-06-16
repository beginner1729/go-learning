// Package store is YOUR implementation target for M02 §4 (fill-in-the-blank).
//
// Goal: make `go test ./store/...` (run it with -race) pass and `go run ./cmd/demo`
// work. The tests in store_test.go define the exact API. Reference answer key:
// ../solution-store/ (and §4 of ../M02-generics-and-idioms.md).
//
// Build here — a generic, concurrency-safe in-memory key-value store that
// replaces M1's hand-written InMemoryRepo:
//
//   - `Store[K comparable, V any]` — a struct holding a `sync.RWMutex` and a
//     `map[K]V`. The map must be initialized via the constructor (writing to a
//     nil map panics).
//
//   - `NewStore[K comparable, V any]() *Store[K, V]` — returns a ready store
//     with an allocated map.
//
//   - `Put(key K, val V)` — write under a write lock (Lock/Unlock).
//
//   - `Get(key K) (V, bool)` — read under a read lock (RLock/RUnlock), returning
//     the comma-ok pair (value, present) — no per-V sentinel needed.
//
//   - `Delete(key K)` — remove under a write lock.
//
//   - `Len() int` — count under a read lock.
//
//   - `All() []V` — return a snapshot slice of all values (safe to iterate after
//     the call), taken under a read lock.
//
// Use `defer` to release each lock. Concurrent Put/Get from many goroutines must
// be race-free.
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package store

// TODO(§4): implement Store[K,V], NewStore, Put, Get, Delete, Len, All.
