// Command profile-svc is Pulse's internal gRPC service (skeleton; filled in M14).
package main

import (
	"github.com/acme/pulse/internal/platform/config"
	"github.com/acme/pulse/internal/platform/log"
)

func main() {
	cfg := config.Load()
	logger := log.New(cfg.LogLevel)
	logger.Info("profile-svc skeleton; gRPC server wired in the capstone (M14)",
		"grpc_addr", cfg.GRPCAddr)
}
