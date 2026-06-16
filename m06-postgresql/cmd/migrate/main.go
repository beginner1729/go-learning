// Command migrate applies all pending migrations, wired to YOUR db package.
// It will NOT build until you implement cxm/m06/db. Requires PULSE_PG_DSN.
// Run: go run ./cmd/migrate
package main

import (
	"log"
	"os"

	"cxm/m06/db"
)

func main() {
	dsn := os.Getenv("PULSE_PG_DSN")
	if dsn == "" {
		log.Fatal("PULSE_PG_DSN is required")
	}
	if err := db.Migrate(dsn); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
	log.Println("migrations applied")
}
