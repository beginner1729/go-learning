// Package config is YOUR implementation target for M02 §2.3 (functional options).
//
// Goal: make `go test ./config/...` pass and `go run ./cmd/demo` work. The tests
// in config_test.go define the exact API. Reference answer key: ../solution-config/
// (and §2.3 of ../M02-generics-and-idioms.md).
//
// Build here — the functional-options pattern Pulse uses for constructors:
//
//   - `ServerConfig` struct with fields:
//     Addr string
//     ReadTimeout, WriteTimeout, ShutdownTimeout time.Duration
//     Logger *slog.Logger
//
//   - `Option func(*ServerConfig)` — a function that mutates a ServerConfig.
//     Adding new options must never break existing callers.
//
//   - Option constructors, each returning an Option:
//     WithReadTimeout(d time.Duration)
//     WithWriteTimeout(d time.Duration)
//     WithShutdownTimeout(d time.Duration)
//     WithLogger(l *slog.Logger)
//
//   - `New(addr string, opts ...Option) ServerConfig` — start from sensible
//     defaults, then apply each option in order. Defaults:
//     ReadTimeout 15s, WriteTimeout 15s, ShutdownTimeout 30s,
//     Logger slog.Default(). An option must override only its own field and
//     leave the other defaults intact.
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package config

// TODO(§2.3): implement ServerConfig, Option, With* options, and New.
