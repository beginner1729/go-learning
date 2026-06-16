// Command demo exercises the M2 generic helpers, store, and config options.
// It is wired to YOUR packages and won't build until you implement them
// (genx, store, config). Run: go run ./cmd/demo
package main

import (
	"fmt"
	"time"

	"cxm/m02/config"
	"cxm/m02/genx"
	"cxm/m02/store"
)

type Customer struct {
	ID   string
	Name string
}

func main() {
	// Generic store standing in for a customer repository.
	repo := store.NewStore[string, Customer]()
	for i := 1; i <= 25; i++ {
		id := fmt.Sprintf("c%02d", i)
		repo.Put(id, Customer{ID: id, Name: fmt.Sprintf("Customer %d", i)})
	}

	// genx.Map to project names, genx.Paginate for a list endpoint window.
	names := genx.Map(repo.All(), func(c Customer) string { return c.Name })
	fmt.Printf("total customers: %d\n", len(names))

	page := genx.Paginate(repo.All(), 2, 10)
	fmt.Printf("page %d/%d, items=%d, hasNext=%v\n",
		page.Page, page.TotalPages, len(page.Items), page.HasNext)

	// Set for deduping tags.
	tags := genx.NewSet("vip", "trial", "vip", "newsletter")
	fmt.Printf("distinct tags: %d\n", tags.Len())

	// Functional options.
	cfg := config.New(":8080", config.WithReadTimeout(5*time.Second))
	fmt.Printf("server addr=%s read=%s shutdown=%s\n",
		cfg.Addr, cfg.ReadTimeout, cfg.ShutdownTimeout)
}
