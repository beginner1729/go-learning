// Command migrate applies all pending migrations (answer-key wiring).
// Run: go run ./solution-cmd/migrate
package main

import (
	"log"
	"os"

	db "cxm/m06/solution-db"
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
