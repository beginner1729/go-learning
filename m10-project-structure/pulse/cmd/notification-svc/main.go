// Command notification-svc consumes events and dispatches notifications
// (skeleton; the JetStream consumer + dispatcher are wired in the capstone, M14).
package main

import (
	"github.com/acme/pulse/internal/platform/config"
	"github.com/acme/pulse/internal/platform/log"
)

func main() {
	cfg := config.Load()
	logger := log.New(cfg.LogLevel)
	logger.Info("notification-svc skeleton; JetStream consumer wired in the capstone (M14)",
		"nats_url", cfg.NATSURL)
}
