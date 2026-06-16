package httpapi

import (
	"errors"
	"net/http"

	customer "cxm/m04/solution-customer"
)

// Stable, client-safe error codes.
const (
	CodeNotFound     = "NOT_FOUND"
	CodeValidation   = "VALIDATION"
	CodeConflict     = "CONFLICT"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeInternal     = "INTERNAL"
)

type codedError struct {
	code string
	err  error
}

func (e *codedError) Error() string { return e.code + ": " + e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

func withCode(err error, code string) error {
	if err == nil {
		return nil
	}
	return &codedError{code: code, err: err}
}

// codeOf inspects the error: explicit code wins, then known sentinels, else INTERNAL.
func codeOf(err error) string {
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	switch {
	case errors.Is(err, customer.ErrNotFound):
		return CodeNotFound
	case errors.Is(err, customer.ErrConflict):
		return CodeConflict
	default:
		var ve *customer.ValidationError
		if errors.As(err, &ve) {
			return CodeValidation
		}
		return CodeInternal
	}
}

func httpStatus(code string) int {
	switch code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeValidation:
		return http.StatusBadRequest
	case CodeConflict:
		return http.StatusConflict
	case CodeUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type errorBody struct {
	Error errorDetail `json:"error"`
}

// respondError maps any error to a consistent JSON envelope + status.
func respondError(w http.ResponseWriter, err error) {
	code := codeOf(err)
	status := httpStatus(code)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal server error" // never leak internals to clients
	}
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: msg}})
}
