// Command server runs the profile-svc gRPC server. Run: go run ./cmd/server
package main

import (
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"

	"cxm/m05/interceptor"
	"cxm/m05/profile"
	profilev1 "cxm/m05/proto/profile/v1"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	token := os.Getenv("GRPC_TOKEN")
	if token == "" {
		token = "dev-token"
	}

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Error("listen failed", "err", err)
		os.Exit(1)
	}

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		interceptor.RecoverUnary(log),
		interceptor.LogUnary(log),
		interceptor.AuthUnary(token),
	))
	profilev1.RegisterProfileServiceServer(srv, profile.NewServer(profile.NewInMemoryRepo()))

	log.Info("profile-svc listening", "addr", ":9090")
	if err := srv.Serve(lis); err != nil {
		log.Error("serve failed", "err", err)
		os.Exit(1)
	}
}
