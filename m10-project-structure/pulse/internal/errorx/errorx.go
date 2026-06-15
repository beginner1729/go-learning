// Package errorx maps domain errors to stable codes and HTTP statuses (from M1).
package errorx

import (
	"errors"
	"net/http"
)

const (
	CodeNotFound   = "NOT_FOUND"
	CodeValidation = "VALIDATION"
	CodeConflict   = "CONFLICT"
	CodeInternal   = "INTERNAL"
)

type codedError struct {
	code string
	err  error
}

func (e *codedError) Error() string { return e.code + ": " + e.err.Error() }
func (e *codedError) Unwrap() error  { return e.err }

func WithCode(err error, code string) error {
	if err == nil {
		return nil
	}
	return &codedError{code: code, err: err}
}

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
