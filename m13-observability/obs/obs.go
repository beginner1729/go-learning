// Package obs wires the three pillars: structured logging (slog), Prometheus
// metrics, and OpenTelemetry tracing — plus HTTP middleware that links them.
package obs

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// ----- Logging -----

func NewLogger(level string) *slog.Logger {
	lvl := slog.LevelInfo
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

// ----- Metrics -----

var (
	HTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, route, status.",
		},
		[]string{"method", "route", "status"},
	)
	HTTPDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	// Example business metric (from the M9 notification flow).
	NotificationsSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "notifications_sent_total",
		Help: "Total notifications successfully sent.",
	})
)

// Registry returns a registry with our metrics and a /metrics handler.
func Registry() (*prometheus.Registry, http.Handler) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(HTTPRequests, HTTPDuration, NotificationsSent)
	return reg, promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

// ----- Tracing -----

// InitTracer sets up a stdout-exporting tracer provider. Returns a shutdown func
// that flushes spans (call on graceful shutdown).
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName),
	))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// ----- Middleware (links the pillars) -----

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

type loggerKey struct{}

// LoggerFrom returns the request-scoped logger (with trace_id) or a default.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Instrument wraps a handler with tracing, metrics, and a trace-correlated logger.
// `route` is the bounded route pattern used as the metric label (NOT the raw path).
func Instrument(base *slog.Logger, route string, next http.Handler) http.Handler {
	tracer := otel.Tracer("pulse")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+route)
		defer span.End()
		span.SetAttributes(attribute.String("http.route", route))

		l := base
		if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
			l = base.With("trace_id", sc.TraceID().String())
		}
		ctx = context.WithValue(ctx, loggerKey{}, l)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))

		elapsed := time.Since(start)
		HTTPRequests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
		HTTPDuration.WithLabelValues(r.Method, route).Observe(elapsed.Seconds())
		l.Info("request", "method", r.Method, "route", route,
			"status", rec.status, "dur_ms", elapsed.Milliseconds())
	})
}
