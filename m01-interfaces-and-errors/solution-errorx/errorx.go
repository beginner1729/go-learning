// Package errorx bridges domain errors to stable, machine-readable codes and
// HTTP statuses. It preserves wrapping so errors.Is/As keep working through it.
//
// This is the reference solution to M1's "implement it yourself" task.
package errorx

import (
	"errors"
	"net/http"
)

// Stable, machine-readable codes. These are part of your API contract and are
// safe to expose to clients, unlike raw error strings.
const (
	CodeNotFound   = "NOT_FOUND"
	CodeValidation = "VALIDATION"
	CodeConflict   = "CONFLICT"
	CodeInternal   = "INTERNAL"
)

// codedError attaches a code to an error while keeping the cause reachable.
type codedError struct {
	code string
	err  error
}

func (e *codedError) Error() string { return e.code + ": " + e.err.Error() }

// Unwrap is what makes errors.Is/As traverse this wrapper.
func (e *codedError) Unwrap() error { return e.err }

// WithCode tags err with a code. Returns nil if err is nil so it composes in
// `return errorx.WithCode(doThing(), errorx.CodeInternal)` style.
func WithCode(err error, code string) error {
	if err == nil {
		return nil
	}
	return &codedError{code: code, err: err}
}

// Code walks the wrap chain and returns the first attached code, or
// CodeInternal if none was set (a safe default — never leak details).
func Code(err error) string {
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	if err == nil {
		return ""
	}
	return CodeInternal
}

// HTTPStatus maps a code to an HTTP status. Built on Code() so it works through
// wrapping too.
func HTTPStatus(err error) int {
	switch Code(err) {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeValidation:
		return http.StatusBadRequest
	case CodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
