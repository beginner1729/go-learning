package obs

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestInstrumentRecordsMetrics(t *testing.T) {
	base := slog.New(slog.NewJSONHandler(io.Discard, nil))
	h := Instrument(base, "/hello", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	got := testutil.ToFloat64(HTTPRequests.WithLabelValues("GET", "/hello", "418"))
	if got != 1 {
		t.Fatalf("counter = %v, want 1", got)
	}
}

func TestRegistryServesMetrics(t *testing.T) {
	_, handler := Registry()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "http_requests_total") {
		t.Fatalf("metrics output missing our metric:\n%s", rec.Body.String()[:200])
	}
}
