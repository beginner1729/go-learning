package profile_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"cxm/m07/mongodb"
	"cxm/m07/profile"
)

func mustRepo(t *testing.T) *profile.MongoRepo {
	t.Helper()
	uri := os.Getenv("PULSE_MONGO_URI")
	if uri == "" {
		t.Skip("PULSE_MONGO_URI not set; skipping Mongo integration test")
	}
	ctx := context.Background()
	client, err := mongodb.Connect(ctx, uri)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(ctx) })

	// Use a throwaway database per test run for isolation.
	dbName := "pulse_test"
	_ = client.Database(dbName).Drop(ctx)

	repo, err := profile.NewMongoRepo(ctx, client, dbName)
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	return repo
}

func TestUpsertGet(t *testing.T) {
	repo := mustRepo(t)
	ctx := context.Background()

	if _, err := repo.Upsert(ctx, profile.Profile{
		CustomerID: "cus_1", Traits: map[string]string{"tier": "gold"},
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// Second upsert updates in place.
	if _, err := repo.Upsert(ctx, profile.Profile{
		CustomerID: "cus_1", Traits: map[string]string{"tier": "platinum", "vip": "true"},
	}); err != nil {
		t.Fatalf("upsert2: %v", err)
	}

	got, err := repo.Get(ctx, "cus_1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Traits["tier"] != "platinum" || got.Traits["vip"] != "true" {
		t.Fatalf("traits = %v", got.Traits)
	}

	if _, err := repo.Get(ctx, "ghost"); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestEventsAndAggregation(t *testing.T) {
	repo := mustRepo(t)
	ctx := context.Background()
	base := time.Now().UTC()

	events := []profile.Event{
		{ID: "e1", CustomerID: "cus_1", Type: "page_view", OccurredAt: base.Add(1 * time.Second)},
		{ID: "e2", CustomerID: "cus_1", Type: "page_view", OccurredAt: base.Add(2 * time.Second)},
		{ID: "e3", CustomerID: "cus_1", Type: "purchase", OccurredAt: base.Add(3 * time.Second)},
		{ID: "e4", CustomerID: "cus_2", Type: "page_view", OccurredAt: base},
	}
	for _, e := range events {
		if err := repo.AppendEvent(ctx, e); err != nil {
			t.Fatalf("append %s: %v", e.ID, err)
		}
	}

	got, err := repo.Events(ctx, "cus_1")
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(got) != 3 || got[0].ID != "e1" { // sorted ascending by time
		t.Fatalf("events = %+v", got)
	}

	counts, err := repo.EventCountsByType(ctx, "cus_1")
	if err != nil {
		t.Fatalf("agg: %v", err)
	}
	if counts["page_view"] != 2 || counts["purchase"] != 1 {
		t.Fatalf("counts = %v", counts)
	}
}
