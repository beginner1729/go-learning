// Package customer carries the domain from M1 (trimmed) plus a thin service.
package customer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ID string
type Email string

var ErrNotFound = errors.New("customer: not found")
var ErrConflict = errors.New("customer: already exists")

type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s", e.Field, e.Msg)
}

type AuditFields struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Customer struct {
	ID    ID
	Email Email
	Name  string
	AuditFields
}

func (c Customer) Validate() error {
	return errors.Join(validateEmail(c.Email), validateName(c.Name))
}

func validateEmail(e Email) error {
	if !strings.Contains(string(e), "@") {
		return &ValidationError{Field: "email", Msg: "must contain @"}
	}
	return nil
}

func validateName(n string) error {
	if strings.TrimSpace(n) == "" {
		return &ValidationError{Field: "name", Msg: "must not be empty"}
	}
	return nil
}

// Repository is the contract the service depends on (swapped for Postgres in M6).
type Repository interface {
	Create(ctx context.Context, c Customer) (Customer, error)
	ByID(ctx context.Context, id ID) (Customer, error)
	List(ctx context.Context) ([]Customer, error)
}

// InMemoryRepo is the M4 implementation.
type InMemoryRepo struct {
	mu  sync.RWMutex
	m   map[ID]Customer
	seq int
}

func NewInMemoryRepo() *InMemoryRepo { return &InMemoryRepo{m: make(map[ID]Customer)} }

func (r *InMemoryRepo) Create(_ context.Context, c Customer) (Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.m {
		if existing.Email == c.Email {
			return Customer{}, ErrConflict
		}
	}
	r.seq++
	c.ID = ID(fmt.Sprintf("cus_%04d", r.seq))
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	r.m[c.ID] = c
	return c, nil
}

func (r *InMemoryRepo) ByID(_ context.Context, id ID) (Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[id]
	if !ok {
		return Customer{}, ErrNotFound
	}
	return c, nil
}

func (r *InMemoryRepo) List(_ context.Context) ([]Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Customer, 0, len(r.m))
	for _, c := range r.m {
		out = append(out, c)
	}
	return out, nil
}

// Service holds business logic and tags errors with stable codes via errorx.
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, c Customer) (Customer, error) {
	return s.repo.Create(ctx, c)
}
func (s *Service) Get(ctx context.Context, id ID) (Customer, error) {
	return s.repo.ByID(ctx, id)
}
func (s *Service) List(ctx context.Context) ([]Customer, error) {
	return s.repo.List(ctx)
}
