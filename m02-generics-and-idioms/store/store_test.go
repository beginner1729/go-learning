package store

import (
	"strconv"
	"sync"
	"testing"
)

func TestStorePutGetDelete(t *testing.T) {
	s := NewStore[string, int]()
	s.Put("a", 1)
	if v, ok := s.Get("a"); !ok || v != 1 {
		t.Fatalf("get a = %d,%v", v, ok)
	}
	s.Delete("a")
	if _, ok := s.Get("a"); ok {
		t.Fatal("delete failed")
	}
}

// Run with: go test -race ./store
func TestStoreConcurrent(t *testing.T) {
	s := NewStore[int, int]()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Put(i, i)
			_, _ = s.Get(i)
			_ = strconv.Itoa(i)
		}(i)
	}
	wg.Wait()
	if s.Len() != 100 {
		t.Fatalf("want 100, got %d", s.Len())
	}
}
