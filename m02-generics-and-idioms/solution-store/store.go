// Package store provides a generic, concurrency-safe in-memory key-value store.
// It replaces the per-entity InMemoryRepo hand-written in M1.
package store

import "sync"

type Store[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewStore[K comparable, V any]() *Store[K, V] {
	return &Store[K, V]{m: make(map[K]V)}
}

func (s *Store[K, V]) Put(key K, val V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = val
}

// Get returns the value and whether it was present (comma-ok idiom — no need for
// a per-V sentinel).
func (s *Store[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	return v, ok
}

func (s *Store[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

func (s *Store[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// All returns a snapshot slice of all values (safe to iterate after the call).
func (s *Store[K, V]) All() []V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]V, 0, len(s.m))
	for _, v := range s.m {
		out = append(out, v)
	}
	return out
}
