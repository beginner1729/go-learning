# Module 13 — Observability & Config

> **Capstone contribution:** Pulse becomes operable — **structured logging**
> (`slog`), **Prometheus metrics**, **OpenTelemetry tracing**, typed **config**,
> **secrets** handling, and **graceful shutdown** wired through every service.

---

## 0. Setup & run

```bash
cd go-cxm-course/m13-observability
go mod tidy
go vet ./...
go test ./...
go run ./cmd/server          # serves :8080 (app) + /metrics; traces to stdout
# curl localhost:8080/hello?name=Ada ; curl -s localhost:8080/metrics | grep http_requests
```

Layout:

```
m13-observability/
  go.mod
  obs/                  # logger, metrics registry, tracer setup, middleware
  cmd/server/           # instrumented HTTP server with graceful shutdown
```

---

## 1. Learning objectives

By the end you will be able to:

- Emit **structured logs** with `slog`, including request-scoped fields.
- Expose **Prometheus metrics** (counters, histograms) and instrument HTTP handlers.
- Add **distributed tracing** with OpenTelemetry (spans, context propagation).
- Manage **configuration** (typed, validated, env-driven) and **secrets** safely.
- Implement **graceful shutdown** that drains servers and flushes telemetry.
- Correlate logs/metrics/traces via IDs (the "three pillars" working together).

---

## 2. Concepts

### 2.1 Structured logging with `slog`

`slog` (stdlib, Go 1.21+) is the standard. Log **key/value pairs**, not formatted
strings — so logs are queryable in aggregation systems:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
logger.Info("customer created", "customer_id", id, "email_domain", domain)
```

- **JSON handler** in prod (machine-parseable), text in dev.
- Attach context with `logger.With("service", "customer-api")` to create a child
  logger whose fields are on every line.
- Put a **request ID / trace ID** on every request's logs so you can correlate.
- **Levels:** Debug/Info/Warn/Error. Log expected conditions at Info/Debug, not Error
  (M1's "don't log not-found as error" lesson).

> **PII discipline:** never log full emails, tokens, or payloads. Log derived,
> non-identifying fields (`email_domain`, `customer_id`). This matters doubly in
> CXM. Add a redaction helper and review what you log.

### 2.2 Metrics with Prometheus

Prometheus scrapes a `/metrics` endpoint. The Go client (`client_golang`) gives
you the four metric types; the two you'll use most:

- **Counter** — monotonically increasing (requests total, errors total).
- **Histogram** — distributions (request latency, payload size) → percentiles.

```go
var (
    httpRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "http_requests_total", Help: "..."},
        []string{"method", "route", "status"},
    )
    httpDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "http_request_duration_seconds",
            Buckets: prometheus.DefBuckets},
        []string{"method", "route"},
    )
)
// register, then in middleware:
httpRequests.WithLabelValues(method, route, status).Inc()
httpDuration.WithLabelValues(method, route).Observe(elapsed.Seconds())
```

- **Label cardinality is the #1 footgun:** never use unbounded values (customer ID,
  email, raw path with IDs) as label values — it explodes memory. Use the *route
  pattern* (`/v1/customers/{id}`), not the concrete path.
- Expose `promhttp.Handler()` at `/metrics`.

### 2.3 Tracing with OpenTelemetry

A **trace** follows one request across services; each operation is a **span**.
OTel is the vendor-neutral standard.

```go
tracer := otel.Tracer("pulse/customer-api")
ctx, span := tracer.Start(ctx, "CreateCustomer")
defer span.End()
span.SetAttributes(attribute.String("customer.id", id))
// pass ctx down; child spans (DB call, gRPC call) attach automatically
```

- **Context propagation** is everything: the trace ID rides in `context.Context`
  in-process and in headers (W3C `traceparent`) across HTTP/gRPC. OTel instrumentation
  libraries (`otelhttp`, `otelgrpc`) inject/extract automatically — wrap your
  handlers and clients with them.
- An **exporter** ships spans to a backend (Jaeger, Tempo, an OTel Collector). In
  dev you can use a stdout exporter.
- Sample in prod (you don't trace 100% at scale): head/tail sampling.

**The payoff — correlation:** put the **trace ID into your logs** (`slog` attr) and
your metrics exemplars. Then an alert → a metric → an exemplar trace → the exact
logs for that request. The three pillars only shine when linked.

### 2.4 Configuration

- **Typed config struct**, loaded from env (12-factor), with **validation** and
  sensible defaults. Fail fast at startup on missing required values.
- Libraries: stdlib + `os.Getenv` for small apps; `envconfig`/`koanf`/`viper` for
  larger. Prefer the smallest that fits; `envconfig` (struct tags → env) is a clean
  middle ground.
- **Never** commit secrets. Read them from env injected by a secrets manager
  (Vault, AWS/GCP Secrets Manager, k8s Secrets). Don't log them; mark secret fields
  so they're redacted in any config dump.

```go
type Config struct {
    HTTPAddr   string `env:"HTTP_ADDR" default:":8080"`
    PostgresDSN string `env:"PULSE_PG_DSN,required"`   // secret: never logged
    OTelEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
}
```

### 2.5 Graceful shutdown (the full version)

On SIGTERM (k8s sends it on every deploy), in order:
1. Stop accepting new work (HTTP `Shutdown`, gRPC `GracefulStop`, drain NATS).
2. Let in-flight requests finish within a deadline.
3. **Flush telemetry** (force-flush the tracer provider, close exporters) so the
   last spans/metrics aren't lost.
4. Close DB pools and the NATS connection.

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
_ = httpSrv.Shutdown(shutdownCtx)
_ = tracerProvider.Shutdown(shutdownCtx) // flush spans
pool.Close(); nc.Drain()
```

#### Pitfalls

- **High-cardinality metric labels** → Prometheus OOM. Bound your label values.
- **Logging PII / secrets** → compliance incident. Redact; log derived fields.
- **Forgetting to flush traces on shutdown** → missing the most interesting spans.
- **`time.Sleep` instead of real readiness** → flaky shutdown; use context/deadlines.
- **String-formatted logs** (`fmt.Sprintf` into a message) → unqueryable; use attrs.
- **Tracing everything at 100% in prod** → cost/overhead; sample.

---

## 3. Hands-on exercises (with full solutions)

### Exercise 3.1 — Metrics middleware

**Task:** Write HTTP middleware that records `http_requests_total{method,route,status}`
and `http_request_duration_seconds{method,route}` using the *route pattern* (not the
concrete path) as the label.

<details>
<summary>Reference solution</summary>

See `m13-observability/obs/metrics.go`. Core:

```go
func Metrics(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rec := &statusRecorder{ResponseWriter: w, status: 200}
        next.ServeHTTP(rec, r)
        route := routePattern(r) // e.g. "/v1/customers/{id}", bounded cardinality
        httpRequests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
        httpDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
    })
}
```

**Reasoning:** using the matched pattern (from the router) instead of `r.URL.Path`
keeps label cardinality bounded — the single most important metrics discipline.

</details>

### Exercise 3.2 — A request logger that carries the trace ID

**Task:** Middleware that derives a child `slog.Logger` with the trace ID from the
span in `ctx`, stashes it in `ctx`, and logs one structured line per request.

<details>
<summary>Reference solution</summary>

```go
func Logging(base *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            sc := trace.SpanContextFromContext(r.Context())
            l := base
            if sc.HasTraceID() {
                l = base.With("trace_id", sc.TraceID().String())
            }
            ctx := context.WithValue(r.Context(), loggerKey{}, l)
            start := time.Now()
            rec := &statusRecorder{ResponseWriter: w, status: 200}
            next.ServeHTTP(rec, r.WithContext(ctx))
            l.Info("request", "method", r.Method, "path", r.URL.Path,
                "status", rec.status, "dur_ms", time.Since(start).Milliseconds())
        })
    }
}
```

**Reasoning:** every log line for a request now carries the `trace_id`, so logs and
traces are joinable in your backend. The per-request logger is passed via context.

</details>

---

## 4. Fill-in-the-blank

Complete the graceful shutdown that also flushes traces.

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
<-ctx.Done()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

_ = httpSrv./* ___1: stop accepting + drain ___ */(shutdownCtx)
_ = tracerProvider./* ___2: flush spans ___ */(shutdownCtx)
pool.Close()
_ = nc./* ___3: graceful NATS drain ___ */()
```

<details>
<summary>Answers</summary>

```go
_ = httpSrv.Shutdown(shutdownCtx)         // 1
_ = tracerProvider.Shutdown(shutdownCtx)  // 2
_ = nc.Drain()                            // 3
```

</details>

---

## 5. Implement it yourself

**Problem:** Instrument the Pulse repo end-to-end:
- Wrap HTTP handlers with `otelhttp` and gRPC with `otelgrpc` so traces propagate
  across `customer-api` → `profile-svc`.
- Add a metrics middleware + `/metrics` to each service; add business metrics
  (`notifications_sent_total`, `events_dead_lettered_total` from M9).
- Add a typed config loader (`envconfig`) with `required` + defaults; fail fast.
- Wire full graceful shutdown (HTTP + gRPC + tracer flush + DB pools + NATS drain).
- Stand up a local **Grafana/Tempo/Prometheus** (or an OTel Collector) via compose
  and watch a request flow across services with a single trace ID.

**Curated resources:**
- `slog` — https://pkg.go.dev/log/slog  ·  guide: https://go.dev/blog/slog
- Prometheus Go client — https://github.com/prometheus/client_golang  ·  best practices: https://prometheus.io/docs/practices/naming/
- OpenTelemetry Go — https://opentelemetry.io/docs/languages/go/
- `otelhttp` / `otelgrpc` — https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation
- 12-Factor config — https://12factor.net/config
- `envconfig` — https://github.com/sethvargo/go-envconfig
- Graceful shutdown — https://pkg.go.dev/net/http#Server.Shutdown

**Hints:** use the OTLP exporter pointed at a local collector; in tests use the
stdout exporter. Keep metric label values bounded (route patterns, status classes).

---

## 6. Capstone contribution

Pulse is now operable: structured logs with trace correlation, Prometheus metrics
(HTTP + business), distributed traces across services, typed config with secrets
handling, and full graceful shutdown that flushes telemetry. Everything M14 needs
to run and debug the whole system is in place.

---

## 7. Self-check — you should now be able to…

- [ ] Emit structured `slog` logs with request/trace-scoped fields (and redact PII).
- [ ] Expose Prometheus counters/histograms with bounded label cardinality.
- [ ] Add OpenTelemetry spans and propagate context across HTTP/gRPC.
- [ ] Load typed, validated config and handle secrets without leaking them.
- [ ] Implement graceful shutdown that drains servers and flushes telemetry.
