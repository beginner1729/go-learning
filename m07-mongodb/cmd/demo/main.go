// Command demo runs an end-to-end flow against real MongoDB, wired to YOUR
// packages. It won't build until you implement cxm/m07/mongodb and
// cxm/m07/profile. Requires PULSE_MONGO_URI. Run: go run ./cmd/demo
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cxm/m07/mongodb"
	"cxm/m07/profile"
)

func main() {
	uri := os.Getenv("PULSE_MONGO_URI")
	if uri == "" {
		log.Fatal("PULSE_MONGO_URI is required")
	}
	ctx := context.Background()

	client, err := mongodb.Connect(ctx, uri)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(ctx)

	repo, err := profile.NewMongoRepo(ctx, client, "pulse")
	if err != nil {
		log.Fatalf("repo: %v", err)
	}

	cust := "cus_demo"
	_, _ = repo.Upsert(ctx, profile.Profile{CustomerID: cust, Traits: map[string]string{
		"tier": "gold", "channel": "email", "locale": "en-US",
	}})
	now := time.Now().UTC()
	for i, typ := range []string{"page_view", "page_view", "add_to_cart", "purchase"} {
		_ = repo.AppendEvent(ctx, profile.Event{
			ID:         fmt.Sprintf("%s_e%d_%d", cust, i, now.UnixNano()),
			CustomerID: cust, Type: typ, OccurredAt: now.Add(time.Duration(i) * time.Second),
		})
	}

	p, _ := repo.Get(ctx, cust)
	fmt.Printf("profile %s traits=%v\n", p.CustomerID, p.Traits)

	counts, _ := repo.EventCountsByType(ctx, cust)
	fmt.Printf("event counts by type: %v\n", counts)
}
