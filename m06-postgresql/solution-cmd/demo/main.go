// Command demo runs an end-to-end flow against real Postgres.
// This is the reference (answer-key) demo wired to the solution packages.
// Requires PULSE_PG_DSN. Run: go run ./solution-cmd/demo
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	customer "cxm/m06/solution-customer"
	db "cxm/m06/solution-db"
)

func main() {
	dsn := os.Getenv("PULSE_PG_DSN")
	if dsn == "" {
		log.Fatal("PULSE_PG_DSN is required")
	}
	ctx := context.Background()

	if err := db.Migrate(dsn); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	repo := customer.NewPostgresRepo(pool)
	c, err := repo.Create(ctx, customer.Customer{Email: customer.Email(fmt.Sprintf("u%d@pulse.dev", os.Getpid())), Name: "Demo User"})
	if err != nil {
		log.Fatalf("create: %v", err)
	}
	fmt.Printf("created %s (%s) at %s\n", c.ID, c.Email, c.CreatedAt.Format("15:04:05"))

	cmp, _ := repo.CreateCampaign(ctx, "Onboarding")
	if err := repo.EnrollCustomer(ctx, c.ID, cmp); err != nil {
		log.Fatalf("enroll: %v", err)
	}
	n, _ := repo.CampaignEnrolledCount(ctx, cmp)
	fmt.Printf("enrolled in %s; enrolled_count=%d\n", cmp, n)

	list, _ := repo.List(ctx, 5, 0)
	fmt.Printf("most recent %d customers:\n", len(list))
	for _, cu := range list {
		fmt.Printf("  - %s %s\n", cu.ID, cu.Email)
	}
}
