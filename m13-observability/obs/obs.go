// Package obs is YOUR implementation target for M13 — Observability.
//
// Goal: make `go test ./obs/...` pass and `go run ./cmd/server` work.
// The tests in obs_test.go define the exact API you must provide.
// Reference answer key: ../solution-obs/ (and §2–§4 of
// ../M13-observability.md). Try it yourself before peeking.
//
// You are wiring the three pillars — structured logging (slog), Prometheus
// metrics, and OpenTelemetry tracing — plus HTTP middleware that links them.
// Keep the package clause `package obs`. External imports you'll need:
// log/slog, net/http, os, strconv, time, context; plus
// github.com/prometheus/client_golang/prometheus (+ /promhttp) and the
// go.opentelemetry.io/otel packages (otel, attribute, trace, the stdouttrace
// exporter, sdk/resource, sdk/trace, semconv).
//
// Build, in THIS file (obs.go):
//
//	----- Logging -----
//	- `NewLogger(level string) *slog.Logger` — returns a JSON-handler slog
//	  logger writing to os.Stdout. Map "debug"/"warn"/"error" to the matching
//	  slog.Level; anything else (incl. "info") is slog.LevelInfo.
//
//	----- Metrics -----
//	- Package-level Prometheus collectors (exported vars):
//	    `HTTPRequests`  = CounterVec "http_requests_total"
//	                      with labels {"method", "route", "status"}.
//	    `HTTPDuration`  = HistogramVec "http_request_duration_seconds"
//	                      with labels {"method", "route"}, prometheus.DefBuckets.
//	    `NotificationsSent` = Counter "notifications_sent_total" (business metric).
//	- `Registry() (*prometheus.Registry, http.Handler)` — a fresh registry that
//	  MustRegister's the three collectors above, returned alongside a
//	  promhttp handler that serves them (used at /metrics). The handler must
//	  respond 200 and its body must contain "http_requests_total".
//
//	----- Tracing -----
//	- `InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error)`
//	  — set up a stdout-exporting tracer provider (stdouttrace, pretty print),
//	  a resource carrying semconv.ServiceName(serviceName), AlwaysSample, batch
//	  exporter; call otel.SetTracerProvider; return the provider's Shutdown func
//	  (flushes spans on graceful shutdown).
//
//	----- Middleware (links the pillars) -----
//	- A `loggerKey` context key + `LoggerFrom(ctx context.Context) *slog.Logger`
//	  that returns the request-scoped logger stashed in ctx, or slog.Default().
//	- A small `statusRecorder` wrapping http.ResponseWriter to capture the
//	  status code (override WriteHeader; default 200).
//	- `Instrument(base *slog.Logger, route string, next http.Handler) http.Handler`
//	  — the core middleware. For each request:
//	    * start an otel span named `r.Method+" "+route` (tracer "pulse"),
//	      defer span.End(), set attribute "http.route"=route;
//	    * derive a child logger with "trace_id" from the span context when it
//	      HasTraceID, and stash it in ctx under loggerKey (so LoggerFrom finds it);
//	    * time the call, serve next with the enriched ctx via a statusRecorder;
//	    * after: Inc HTTPRequests{method, route, status} and Observe
//	      HTTPDuration{method, route} (seconds), then log one structured
//	      "request" line with method/route/status/dur_ms.
//	  NOTE: `route` is the BOUNDED route pattern used as the metric label, never
//	  the raw r.URL.Path — bounded cardinality is the key metrics discipline.
//
// The test TestInstrumentRecordsMetrics calls Instrument(base, "/hello", ...)
// against a handler that writes 418, then asserts
// HTTPRequests.WithLabelValues("GET","/hello","418") == 1.
// TestRegistryServesMetrics asserts Registry()'s handler serves 200 with the
// metric name in the body.
//
// Delete this comment block as you implement. The package will not compile
// until the types, vars, and functions the tests reference exist.
package obs

// TODO: implement NewLogger, HTTPRequests, HTTPDuration, NotificationsSent,
// Registry, InitTracer, LoggerFrom, statusRecorder, Instrument.
