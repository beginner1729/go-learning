// Package interceptor is YOUR implementation target for M05 (gRPC middleware).
//
// Goal: make `go test ./interceptor/... ./profile/...` pass and
// `go run ./cmd/server` work. The tests in ../profile/profile_test.go wire
// these three interceptors into a bufconn server, so their exact signatures
// and behavior are fixed. Reference answer key: ../solution-interceptor/
// (and §2.6 + §4 of ../M05-grpc.md). Try it yourself before peeking.
//
// gRPC unary interceptors are middleware: each has the shape
//
//	grpc.UnaryServerInterceptor = func(ctx context.Context, req any,
//	    info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error)
//
// You wrap `handler` — call it to continue the chain, or short-circuit by
// returning an error before you do.
//
// Build, in THIS file (interceptor.go) — three constructors that each RETURN a
// grpc.UnaryServerInterceptor:
//
//   - AuthUnary(token string) grpc.UnaryServerInterceptor
//     Reads the "authorization" metadata header from the incoming context
//     (metadata.FromIncomingContext). If it is missing or is not exactly
//     "Bearer "+token, reject with status.Error(codes.Unauthenticated,
//     "invalid token"). Otherwise call handler and return its result.
//     The profile tests rely on this: a call with no metadata must come back
//     as codes.Unauthenticated.
//
//   - LogUnary(log *slog.Logger) grpc.UnaryServerInterceptor
//     Times the call, runs handler, then logs one line with the method
//     (info.FullMethod), resulting code (status.Code(err).String()), and
//     duration in ms. Returns handler's (resp, err) unchanged.
//
//   - RecoverUnary(log *slog.Logger) grpc.UnaryServerInterceptor
//     Defers a recover(); if the handler panics, log it (method + recovered
//     value + debug.Stack()) and return status.Error(codes.Internal,
//     "internal error") via a NAMED return so the panic does not escape.
//
// These are chained in cmd/server/main.go and the tests as:
//
//	grpc.ChainUnaryInterceptor(RecoverUnary(log), LogUnary(log), AuthUnary(token))
//
// Order matters: recovery outermost, then logging, then auth (see §4).
//
// Delete this comment block as you implement. The package will not compile
// until AuthUnary, LogUnary, and RecoverUnary exist.
package interceptor

// TODO(M05): implement AuthUnary, LogUnary, RecoverUnary
// (each returns a grpc.UnaryServerInterceptor).
