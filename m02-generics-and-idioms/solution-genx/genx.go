// Package genx holds small, reusable generic helpers used across Pulse:
// a Set, slice/map utilities, and pagination.
package genx

// Map applies f to each element, returning a new slice of the result type.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// Filter returns the elements for which keep returns true.
func Filter[T any](s []T, keep func(T) bool) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Keys returns the keys of a map in unspecified order.
func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Values returns the values of a map in unspecified order.
func Values[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// Set is a generic set backed by a map with zero-byte values.
type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{m: make(map[T]struct{}, len(items))}
	for _, it := range items {
		s.Add(it)
	}
	return s
}

func (s *Set[T]) Add(v T)    { s.m[v] = struct{}{} }
func (s *Set[T]) Remove(v T) { delete(s.m, v) }

func (s *Set[T]) Has(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *Set[T]) Len() int { return len(s.m) }

func (s *Set[T]) Items() []T {
	out := make([]T, 0, len(s.m))
	for v := range s.m {
		out = append(out, v)
	}
	return out
}

// Page is a paginated window over a slice plus metadata. It is internal plumbing;
// the REST layer maps it to a JSON DTO rather than exposing Page[T] directly.
type Page[T any] struct {
	Items      []T
	Page       int
	Size       int
	Total      int
	TotalPages int
	HasNext    bool
}

// Paginate returns the requested page window, clamping invalid inputs.
func Paginate[T any](items []T, page, size int) Page[T] {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	total := len(items)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	totalPages := (total + size - 1) / size
	return Page[T]{
		Items:      items[start:end],
		Page:       page,
		Size:       size,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
	}
}
