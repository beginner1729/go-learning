// Command demo exercises the M1 customer + errorx packages end-to-end.
// Run: go run ./cmd/demo
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"cxm/m01/customer"
	"cxm/m01/errorx"
)

// LoggingRepo is the decorator from the lesson: it embeds the interface so
// unoverridden methods are promoted, and adds logging to the ones it cares about.
type LoggingRepo struct {
	customer.CustomerRepository
	log *slog.Logger
}

func (r LoggingRepo) Create(ctx context.Context, c customer.Customer) (customer.Customer, error) {
	r.log.Info("creating customer", "email", string(c.Email))
	return r.CustomerRepository.Create(ctx, c)
}

func (r LoggingRepo) ByID(ctx context.Context, id customer.CustomerID) (customer.Customer, error) {
	c, err := r.CustomerRepository.ByID(ctx, id)
	if err != nil && !errors.Is(err, customer.ErrNotFound) {
		r.log.Error("byID failed", "id", string(id), "err", err)
	}
	return c, err
}

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var repo customer.CustomerRepository = LoggingRepo{
		CustomerRepository: customer.NewInMemoryRepo(),
		log:                log,
	}

	// 1. Validation with typed errors.
	bad := customer.Customer{ID: "c1", Email: "nope", Name: ""}
	if err := bad.Validate(); err != nil {
		var ve *customer.ValidationError
		if errors.As(err, &ve) {
			fmt.Printf("first invalid field: %s (%s)\n", ve.Field, ve.Msg)
		}
		fmt.Printf("all validation errors: %v\n", err)
	}

	// 2. Happy path create + fetch.
	good := customer.Customer{ID: "c1", Email: "ada@pulse.dev", Name: "Ada"}
	if _, err := repo.Create(ctx, good); err != nil {
		fmt.Println("create failed:", err)
		return
	}
	got, _ := repo.ByID(ctx, "c1")
	fmt.Printf("fetched: %s <%s> created=%s\n", got.Name, got.Email, got.CreatedAt.Format("15:04:05"))

	// 3. Sentinel + errorx code/status mapping.
	_, err := repo.ByID(ctx, "missing")
	coded := errorx.WithCode(err, errorx.CodeNotFound)
	fmt.Printf("missing lookup -> code=%s http=%d isNotFound=%v\n",
		errorx.Code(coded), errorx.HTTPStatus(coded), errors.Is(coded, customer.ErrNotFound))
}
