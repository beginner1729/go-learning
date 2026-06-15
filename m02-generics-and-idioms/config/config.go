// Package config demonstrates the functional-options pattern Pulse uses for
// constructing servers and services with defaults + optional overrides.
package config

import (
	"log/slog"
	"time"
)

type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Logger          *slog.Logger
}

// Option mutates a ServerConfig. Adding new options never breaks existing callers.
type Option func(*ServerConfig)

func WithReadTimeout(d time.Duration) Option {
	return func(c *ServerConfig) { c.ReadTimeout = d }
}

func WithWriteTimeout(d time.Duration) Option {
	return func(c *ServerConfig) { c.WriteTimeout = d }
}

func WithShutdownTimeout(d time.Duration) Option {
	return func(c *ServerConfig) { c.ShutdownTimeout = d }
}

func WithLogger(l *slog.Logger) Option {
	return func(c *ServerConfig) { c.Logger = l }
}

// New builds a ServerConfig with sensible defaults, then applies overrides.
func New(addr string, opts ...Option) ServerConfig {
	c := ServerConfig{
		Addr:            addr,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		Logger:          slog.Default(),
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}
