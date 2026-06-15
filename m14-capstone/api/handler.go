// Package api is the REST surface: POST /v1/customers stores the customer and
// publishes a durable cxm.customer.created event (event-driven decoupling).
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"cxm/m14/domain"
)

// EventPublisher is the boundary the handler depends on (events.Publisher satisfies it).
type EventPublisher interface {
	PublishCustomerCreated(ctx context.Context, e domain.CustomerEvent) error
}

type Handler struct {
	pub EventPublisher

	mu   sync.Mutex
	byID map[string]domain.Customer
}

func NewHandler(pub EventPublisher) *Handler {
	return &Handler{pub: pub, byID: map[string]domain.Customer{}}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/customers", h.create)
	return mux
}

type createReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("VALIDATION", "invalid JSON"))
		return
	}
	if !strings.Contains(req.Email, "@") || strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errBody("VALIDATION", "email and name required"))
		return
	}

	c := domain.Customer{
		ID:        "cus_" + randHex(),
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: time.Now().UTC(),
	}
	h.mu.Lock()
	h.byID[c.ID] = c
	h.mu.Unlock()

	// Publish the domain event. We respond 201 regardless of downstream
	// notification — that's the decoupling: the client doesn't wait for the email.
	_ = h.pub.PublishCustomerCreated(r.Context(), domain.CustomerEvent{
		ID: c.ID, Email: c.Email, Name: c.Name, OccurredAt: c.CreatedAt,
	})

	writeJSON(w, http.StatusCreated, c)
}

func randHex() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func errBody(code, msg string) map[string]any {
	return map[string]any{"error": map[string]string{"code": code, "message": msg}}
}
