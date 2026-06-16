// Package interceptor holds gRPC unary interceptors (auth, logging, recovery).
package interceptor

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthUnary checks a static bearer token in the "authorization" metadata header.
func AuthUnary(token string) grpc.UnaryServerInterceptor {
	want := "Bearer " + token
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		vals := md.Get("authorization")
		if len(vals) == 0 || vals[0] != want {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(ctx, req)
	}
}

// LogUnary logs method, duration, and resulting code.
func LogUnary(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Info("grpc unary",
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"dur_ms", time.Since(start).Milliseconds(),
		)
		return resp, err
	}
}

// RecoverUnary converts panics into a clean Internal error.
func RecoverUnary(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("grpc panic recovered",
					"method", info.FullMethod, "err", rec, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}
