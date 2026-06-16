package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	customer "cxm/m04/solution-customer"
)

type Handler struct {
	svc   *customer.Service
	log   *slog.Logger
	token string
}

func NewHandler(svc *customer.Service, log *slog.Logger, token string) *Handler {
	return &Handler{svc: svc, log: log, token: token}
}

// ----- DTOs (wire format; domain types never serialized directly) -----

type createCustomerRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type customerResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func toResponse(c customer.Customer, redactPII bool) customerResponse {
	email := string(c.Email)
	if redactPII {
		email = redactEmail(email)
	}
	return customerResponse{
		ID:        string(c.ID),
		Email:     email,
		Name:      c.Name,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}

func redactEmail(e string) string {
	at := strings.IndexByte(e, '@')
	if at <= 1 {
		return "***" + e[at:]
	}
	return e[:1] + "***" + e[at:]
}

type listResponse struct {
	Items      []customerResponse `json:"items"`
	Page       int                `json:"page"`
	TotalPages int                `json:"total_pages"`
	Total      int                `json:"total"`
	HasNext    bool               `json:"has_next"`
}

// ----- Router -----

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(Recover(h.log), RequestID, Logging(h.log))

	r.Get("/healthz", h.health)

	r.Route("/v1", func(r chi.Router) {
		r.Use(BearerAuth(h.token))
		r.Post("/customers", h.createCustomer)
		r.Get("/customers", h.listCustomers)
		r.Get("/customers/{id}", h.getCustomer)
	})
	return r
}

// ----- Handlers -----

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createCustomer(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[createCustomerRequest](w, r)
	if err != nil {
		respondError(w, withCode(err, CodeValidation))
		return
	}
	c := customer.Customer{Email: customer.Email(req.Email), Name: req.Name}
	if err := c.Validate(); err != nil {
		respondError(w, err) // ValidationError -> 400 via codeOf
		return
	}
	created, err := h.svc.Create(r.Context(), c)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(created, h.pii(r)))
}

func (h *Handler) getCustomer(w http.ResponseWriter, r *http.Request) {
	id := customer.ID(chi.URLParam(r, "id"))
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(c, h.pii(r)))
}

func (h *Handler) listCustomers(w http.ResponseWriter, r *http.Request) {
	all, err := h.svc.List(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	page := atoiDefault(r.URL.Query().Get("page"), 1)
	size := atoiDefault(r.URL.Query().Get("size"), 20)
	p := paginate(all, page, size)
	items := make([]customerResponse, len(p.items))
	for i, c := range p.items {
		items[i] = toResponse(c, h.pii(r))
	}
	writeJSON(w, http.StatusOK, listResponse{
		Items: items, Page: p.page, TotalPages: p.totalPages, Total: p.total, HasNext: p.hasNext,
	})
}

// pii reports whether the caller may see unredacted PII.
func (h *Handler) pii(r *http.Request) bool {
	return r.Header.Get("X-Scope") != "pii" // redact unless explicitly scoped
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// minimal local pagination (M2's genx.Paginate would be imported in the capstone)
type pageView struct {
	items      []customer.Customer
	page, size int
	total      int
	totalPages int
	hasNext    bool
}

func paginate(items []customer.Customer, page, size int) pageView {
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
	return pageView{items[start:end], page, size, total, totalPages, page < totalPages}
}
