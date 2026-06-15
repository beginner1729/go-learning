package customer

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryRepo_CreateAndByID(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepo()

	in := Customer{ID: "c1", Email: "a@b.com", Name: "Ada"}
	got, err := repo.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("audit fields not stamped: %+v", got)
	}

	fetched, err := repo.ByID(ctx, "c1")
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if fetched.Email != "a@b.com" {
		t.Fatalf("got email %q", fetched.Email)
	}
}

func TestInMemoryRepo_ByID_NotFound(t *testing.T) {
	repo := NewInMemoryRepo()
	_, err := repo.ByID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestValidate_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		in        Customer
		wantField string // "" means expect no error
	}{
		{"ok", Customer{Email: "a@b.com", Name: "Ada"}, ""},
		{"bad email", Customer{Email: "nope", Name: "Ada"}, "email"},
		{"empty name", Customer{Email: "a@b.com", Name: "  "}, "name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("want nil, got %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *ValidationError, got %v", err)
			}
			if ve.Field != tt.wantField {
				t.Fatalf("want field %q, got %q", tt.wantField, ve.Field)
			}
		})
	}
}
