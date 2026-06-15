package customer_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cxm/m06/customer"
	"cxm/m06/db"
)

// dsn returns the test DSN or skips the test if none is configured.
func mustPool(t *testing.T) (*customer.PostgresRepo, func()) {
	t.Helper()
	dsn := os.Getenv("PULSE_PG_DSN")
	if dsn == "" {
		t.Skip("PULSE_PG_DSN not set; skipping Postgres integration test")
	}
	if err := db.Migrate(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	// Clean tables for a deterministic run.
	_, _ = pool.Exec(context.Background(), `TRUNCATE enrollments, campaigns, customers CASCADE`)
	return customer.NewPostgresRepo(pool), pool.Close
}

func TestCreateByIDConflict(t *testing.T) {
	repo, closeFn := mustPool(t)
	defer closeFn()
	ctx := context.Background()

	c, err := repo.Create(ctx, customer.Customer{Email: "ada@pulse.dev", Name: "Ada"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.ByID(ctx, c.ID)
	if err != nil || got.Email != "ada@pulse.dev" {
		t.Fatalf("byID: %v %+v", err, got)
	}

	_, err = repo.Create(ctx, customer.Customer{Email: "ada@pulse.dev", Name: "Dup"})
	if !errors.Is(err, customer.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}

	_, err = repo.ByID(ctx, "ghost")
	if !errors.Is(err, customer.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestTransactionalEnroll(t *testing.T) {
	repo, closeFn := mustPool(t)
	defer closeFn()
	ctx := context.Background()

	c, _ := repo.Create(ctx, customer.Customer{Email: "tx@pulse.dev", Name: "Tx"})
	cmp, err := repo.CreateCampaign(ctx, "Welcome Series")
	if err != nil {
		t.Fatalf("campaign: %v", err)
	}

	if err := repo.EnrollCustomer(ctx, c.ID, cmp); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	n, _ := repo.CampaignEnrolledCount(ctx, cmp)
	if n != 1 {
		t.Fatalf("enrolled_count = %d, want 1", n)
	}

	// Enrolling an unknown customer must roll back (FK violation -> ErrNotFound)
	// and must NOT bump the count.
	if err := repo.EnrollCustomer(ctx, "cus_ghost", cmp); !errors.Is(err, customer.ErrNotFound) {
		t.Fatalf("want ErrNotFound on bad FK, got %v", err)
	}
	n, _ = repo.CampaignEnrolledCount(ctx, cmp)
	if n != 1 {
		t.Fatalf("count changed after rollback: %d", n)
	}
}
