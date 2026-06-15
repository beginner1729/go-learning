// Package httpx provides a reusable HTTP server with timeouts and graceful shutdown.
package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	srv *http.Server
	log *slog.Logger
}

func NewServer(addr string, handler http.Handler, log *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		log: log,
	}
}

// Run starts the server and blocks until SIGINT/SIGTERM, then shuts down gracefully.
func (s *Server) Run() error {
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http listening", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutdownCtx)
	}
}
