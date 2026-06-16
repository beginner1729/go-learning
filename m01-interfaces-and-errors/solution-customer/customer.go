// Package customer defines Pulse's core customer domain: value types, the
// Customer entity, the repository contract, and an in-memory implementation.
package customer

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CustomerID and Email are string-based value types. Using distinct types
// (instead of bare string) makes signatures self-documenting and prevents
// argument-order bugs: ByID(someEmail) won't compile.
type CustomerID string
type Email string

// ErrNotFound is a sentinel: an expected, branchable condition with no extra
// data. Callers check it with errors.Is(err, ErrNotFound).
var ErrNotFound = errors.New("customer: not found")

// AuditFields is embedded into every entity that needs created/updated stamps.
// Embedding (not a named field) promotes these fields onto the entity.
type AuditFields struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Customer is the core entity.
type Customer struct {
	ID    CustomerID
	Email Email
	Name  string
	AuditFields
}

// CustomerRepository is the contract consumers depend on. It is defined here for
// course convenience, but the idiom is to declare it in the package that *uses*
// it. It says nothing about Postgres/Mongo — only what operations exist.
type CustomerRepository interface {
	Create(ctx context.Context, c Customer) (Customer, error)
	ByID(ctx context.Context, id CustomerID) (Customer, error)
}

// InMemoryRepo is a concrete, concurrency-safe implementation handy for tests
// and demos. Reads dominate, so we use an RWMutex.
type InMemoryRepo struct {
	mu sync.RWMutex
	m  map[CustomerID]Customer
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{m: make(map[CustomerID]Customer)}
}

func (r *InMemoryRepo) Create(_ context.Context, c Customer) (Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	r.m[c.ID] = c
	return c, nil
}

func (r *InMemoryRepo) ByID(_ context.Context, id CustomerID) (Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[id]
	if !ok {
		return Customer{}, ErrNotFound
	}
	return c, nil
}
