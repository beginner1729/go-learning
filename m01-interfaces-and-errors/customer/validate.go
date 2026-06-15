package customer

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError is a typed error: it carries structured data (which field,
// why) so the caller can build a precise response. Callers extract it with
// errors.As(err, &ve).
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s", e.Field, e.Msg)
}

// Validate returns the error interface (not *ValidationError) to avoid the
// typed-nil trap and to allow errors.Join to combine multiple field failures.
func (c Customer) Validate() error {
	return errors.Join(
		validateEmail(c.Email),
		validateName(c.Name),
	)
}

func validateEmail(e Email) error {
	if !strings.Contains(string(e), "@") {
		return &ValidationError{Field: "email", Msg: "must contain @"}
	}
	return nil
}

func validateName(n string) error {
	if strings.TrimSpace(n) == "" {
		return &ValidationError{Field: "name", Msg: "must not be empty"}
	}
	return nil
}
