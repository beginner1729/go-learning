package customer

import (
	"context"
	"errors"
	"testing"

	"github.com/acme/pulse/internal/errorx"
)

func TestServiceCreateValidationAndConflict(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewInMemoryRepo())

	if _, err := svc.Create(ctx, Customer{Email: "bad", Name: "X"}); errorx.Code(err) != errorx.CodeValidation {
		t.Fatalf("want VALIDATION, got %v", err)
	}

	c, err := svc.Create(ctx, Customer{Email: "a@b.com", Name: "Ada"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := svc.Get(ctx, c.ID)
	if err != nil || got.Email != "a@b.com" {
		t.Fatalf("get: %v %+v", err, got)
	}

	if _, err := svc.Create(ctx, Customer{Email: "a@b.com", Name: "Dup"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}

	if _, err := svc.Get(ctx, "ghost"); errorx.Code(err) != errorx.CodeNotFound {
		t.Fatalf("want NOT_FOUND, got %v", err)
	}
}
