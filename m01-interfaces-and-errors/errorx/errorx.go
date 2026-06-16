// Package errorx is YOUR implementation target for M01 §5 ("Implement it
// yourself"). It bridges domain errors to stable, machine-readable codes and
// HTTP statuses, while preserving wrapping so errors.Is/As keep working.
//
// Goal: make `go test ./errorx/...` pass. The tests in errorx_test.go define
// the contract. Reference answer key: ../solution-errorx/.
//
// Build here:
//
//   - Exported code constants (strings — part of your API contract, safe to
//     show clients):
//         CodeNotFound   = "NOT_FOUND"
//         CodeValidation = "VALIDATION"
//         CodeConflict   = "CONFLICT"
//         CodeInternal   = "INTERNAL"
//
//   - An unexported `codedError` type that attaches a code to an error:
//         type codedError struct { code string; err error }
//     with Error() string AND — crucially — Unwrap() error returning the
//     wrapped err. Unwrap is what makes errors.Is/As traverse your wrapper.
//
//   - WithCode(err error, code string) error — tags err with code; returns nil
//     when err is nil so it composes in `return WithCode(doThing(), CodeX)`.
//
//   - Code(err error) string — walk the chain with errors.As looking for
//     *codedError; return its code. Return "" for nil, and CodeInternal as the
//     safe default for an untagged non-nil error (never leak raw details).
//
//   - HTTPStatus(err error) int — switch on Code(err): NotFound->404,
//     Validation->400, Conflict->409, default->500. net/http has the constants.
//
// TODO: implement the constants, codedError, WithCode, Code, HTTPStatus.
package errorx
