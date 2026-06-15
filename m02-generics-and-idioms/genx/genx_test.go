package genx

import (
	"sort"
	"testing"
)

func TestMapFilter(t *testing.T) {
	in := []int{1, 2, 3, 4}
	doubled := Map(in, func(x int) int { return x * 2 })
	evens := Filter(in, func(x int) bool { return x%2 == 0 })
	if doubled[3] != 8 || len(evens) != 2 {
		t.Fatalf("Map/Filter wrong: %v %v", doubled, evens)
	}
}

func TestSet(t *testing.T) {
	s := NewSet("a", "b", "a")
	if s.Len() != 2 || !s.Has("a") {
		t.Fatalf("set wrong: len=%d", s.Len())
	}
	s.Remove("a")
	if s.Has("a") {
		t.Fatal("remove failed")
	}
}

func TestPaginate(t *testing.T) {
	items := make([]int, 25)
	for i := range items {
		items[i] = i
	}
	p := Paginate(items, 2, 10)
	if p.Total != 25 || p.TotalPages != 3 || len(p.Items) != 10 || !p.HasNext {
		t.Fatalf("page meta wrong: %+v", p)
	}
	if p.Items[0] != 10 {
		t.Fatalf("page 2 should start at 10, got %d", p.Items[0])
	}
	// Out-of-range page returns empty window, not a panic.
	last := Paginate(items, 99, 10)
	if len(last.Items) != 0 || last.HasNext {
		t.Fatalf("oob page wrong: %+v", last)
	}
}

func TestKeysValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	ks := Keys(m)
	sort.Strings(ks)
	if len(ks) != 2 || ks[0] != "a" {
		t.Fatalf("keys wrong: %v", ks)
	}
	if len(Values(m)) != 2 {
		t.Fatalf("values wrong")
	}
}
