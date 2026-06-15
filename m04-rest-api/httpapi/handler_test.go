package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"cxm/m04/customer"
)

func newTestHandler() *Handler {
	svc := customer.NewService(customer.NewInMemoryRepo())
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(svc, log, "dev-token")
}

func do(h *Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, r)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func auth() map[string]string { return map[string]string{"Authorization": "Bearer dev-token"} }

func TestCreateAndGet(t *testing.T) {
	h := newTestHandler()

	rec := do(h, http.MethodPost, "/v1/customers", `{"email":"ada@pulse.dev","name":"Ada"}`, auth())
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body)
	}
	var created customerResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.ID == "" {
		t.Fatal("missing id")
	}
	// default response redacts PII
	if created.Email != "a***@pulse.dev" {
		t.Fatalf("expected redacted email, got %q", created.Email)
	}

	// With pii scope, email is unredacted.
	hdr := auth()
	hdr["X-Scope"] = "pii"
	rec = do(h, http.MethodGet, "/v1/customers/"+created.ID, "", hdr)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	var got customerResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Email != "ada@pulse.dev" {
		t.Fatalf("expected full email, got %q", got.Email)
	}
}

func TestValidationAndErrors(t *testing.T) {
	h := newTestHandler()
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{"no auth", http.MethodGet, "/v1/customers", "", nil, http.StatusUnauthorized, CodeUnauthorized},
		{"bad email", http.MethodPost, "/v1/customers", `{"email":"nope","name":"X"}`, auth(), http.StatusBadRequest, CodeValidation},
		{"unknown field", http.MethodPost, "/v1/customers", `{"email":"a@b.com","name":"X","x":1}`, auth(), http.StatusBadRequest, CodeValidation},
		{"not found", http.MethodGet, "/v1/customers/missing", "", auth(), http.StatusNotFound, CodeNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, tt.method, tt.path, tt.body, tt.headers)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tt.wantStatus, rec.Body)
			}
			var eb errorBody
			_ = json.Unmarshal(rec.Body.Bytes(), &eb)
			if eb.Error.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", eb.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestConflict(t *testing.T) {
	h := newTestHandler()
	body := `{"email":"dup@pulse.dev","name":"Dup"}`
	_ = do(h, http.MethodPost, "/v1/customers", body, auth())
	rec := do(h, http.MethodPost, "/v1/customers", body, auth())
	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rec.Code)
	}
}
