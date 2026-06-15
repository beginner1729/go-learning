// Package customer is the domain package (organized by feature): it owns the
// type, the service, the repository interface, and the HTTP handler.
package customer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/acme/pulse/internal/errorx"
)

type ID string
type Email string

var (
	ErrNotFound = errors.New("customer: not found")
	ErrConflict = errors.New("customer: already exists")
)

type Customer struct {
	ID        ID
	Email     Email
	Name      string
	CreatedAt time.Time
}

func (c Customer) Validate() error {
	if !strings.Contains(string(c.Email), "@") {
		return errors.New("email must contain @")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name must not be empty")
	}
	return nil
}

// Repository is the storage contract (implemented by platform/postgres in the capstone).
type Repository interface {
	Create(ctx context.Context, c Customer) (Customer, error)
	ByID(ctx context.Context, id ID) (Customer, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, c Customer) (Customer, error) {
	if err := c.Validate(); err != nil {
		return Customer{}, errorx.WithCode(err, errorx.CodeValidation)
	}
	return s.repo.Create(ctx, c)
}

func (s *Service) Get(ctx context.Context, id ID) (Customer, error) {
	c, err := s.repo.ByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Customer{}, errorx.WithCode(err, errorx.CodeNotFound)
	}
	return c, err
}

// InMemoryRepo is the default wiring; swap for postgres.CustomerRepo in prod.
type InMemoryRepo struct {
	mu  sync.RWMutex
	m   map[ID]Customer
	seq int
}

func NewInMemoryRepo() *InMemoryRepo { return &InMemoryRepo{m: map[ID]Customer{}} }

func (r *InMemoryRepo) Create(_ context.Context, c Customer) (Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.m {
		if e.Email == c.Email {
			return Customer{}, ErrConflict
		}
	}
	r.seq++
	c.ID = ID("cus_" + itoa(r.seq))
	c.CreatedAt = time.Now().UTC()
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Handler exposes the service over HTTP (thin DTO mapping).
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/customers", h.create)
	mux.HandleFunc("GET /v1/customers/{id}", h.get)
	return mux
}

type createReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
type resp struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, errorx.WithCode(err, errorx.CodeValidation))
		return
	}
	c, err := h.svc.Create(r.Context(), Customer{Email: Email(req.Email), Name: req.Name})
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp{string(c.ID), string(c.Email), c.Name})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), ID(r.PathValue("id")))
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp{string(c.ID), string(c.Email), c.Name})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respondErr(w http.ResponseWriter, err error) {
	status := errorx.HTTPStatus(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal server error"
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{
		"code": errorx.Code(err), "message": msg,
	}})
}
