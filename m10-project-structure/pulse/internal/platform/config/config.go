// Package config loads service configuration from the environment.
package config

import "os"

type Config struct {
	HTTPAddr    string
	GRPCAddr    string
	PostgresDSN string
	MongoURI    string
	NATSURL     string
	LogLevel    string
}

func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		GRPCAddr:    getenv("GRPC_ADDR", ":9090"),
		PostgresDSN: getenv("PULSE_PG_DSN", ""),
		MongoURI:    getenv("PULSE_MONGO_URI", ""),
		NATSURL:     getenv("PULSE_NATS_URL", "nats://localhost:4222"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
