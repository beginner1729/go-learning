// Command server is a fully instrumented HTTP service: structured logs with
// trace IDs, Prometheus /metrics, OpenTelemetry spans, and graceful shutdown
// that flushes telemetry. Run: go run ./cmd/server
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cxm/m13/obs"
)

func main() {
	logger := obs.NewLogger("info")

	ctx := context.Background()
	shutdownTracer, err := obs.InitTracer(ctx, "pulse-demo")
	if err != nil {
		logger.Error("tracer init", "err", err)
		os.Exit(1)
	}

	_, metricsHandler := obs.Registry()

	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)
	mux.Handle("/hello", obs.Instrument(logger, "/hello",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := r.URL.Query().Get("name")
			if name == "" {
				name = "world"
			}
			// Use the request-scoped (trace-correlated) logger.
			obs.LoggerFrom(r.Context()).Info("greeting", "name", name)
			obs.NotificationsSent.Inc() // pretend we sent a welcome
			fmt.Fprintf(w, "hello, %s\n", name)
		})))

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve", "err", err)
		}
	}()

	// Graceful shutdown: drain HTTP, then flush spans.
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = shutdownTracer(shutdownCtx) // flush remaining spans
	logger.Info("stopped")
}
