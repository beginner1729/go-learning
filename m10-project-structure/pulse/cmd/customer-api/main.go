// Command customer-api is Pulse's public REST service. main only wires + starts.
package main

import (
	"os"

	"github.com/acme/pulse/internal/customer"
	"github.com/acme/pulse/internal/platform/config"
	"github.com/acme/pulse/internal/platform/httpx"
	"github.com/acme/pulse/internal/platform/log"
)

func main() {
	cfg := config.Load()
	logger := log.New(cfg.LogLevel)

	// Dependency wiring (manual DI). Swap NewInMemoryRepo for postgres in prod.
	repo := customer.NewInMemoryRepo()
	svc := customer.NewService(repo)
	handler := customer.NewHandler(svc)

	srv := httpx.NewServer(cfg.HTTPAddr, handler.Routes(), logger)
	if err := srv.Run(); err != nil {
		logger.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}
