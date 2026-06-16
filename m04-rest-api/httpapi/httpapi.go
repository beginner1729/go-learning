// Package httpapi is YOUR implementation target for M04. It is the REST layer
// over the customer service: routing (chi), middleware, safe JSON, DTOs, and the
// M1 error-to-status bridge.
//
// Goal: make `go test ./customer/... ./httpapi/...` pass and `go run ./cmd/api`
// work. The tests in handler_test.go define the exact behaviour and the exported
// surface you must provide. Reference answer key: ../solution-httpapi/ (which
// splits this across errors.go, handler.go, json.go, middleware.go — you may use
// matching files or one file, your call). Lesson + reasoning, especially §3, §4
// and §5: ../M04-rest-api.md. Try it yourself before peeking.
//
// This package imports the sibling domain package "cxm/m04/customer" (the
// LEARNER one). It must NOT serialize domain types directly — DTOs in, DTOs out.
//
// ----- Error mapping (errors.go in the solution) -----
//
//   - Exported, client-safe code constants (strings):
//     CodeNotFound     = "NOT_FOUND"
//     CodeValidation   = "VALIDATION"
//     CodeConflict     = "CONFLICT"
//     CodeUnauthorized = "UNAUTHORIZED"
//     CodeInternal     = "INTERNAL"
//
//   - An unexported `codedError` { code string; err error } with Error() and
//     Unwrap() (Unwrap is what lets errors.Is/As see through it), and a helper
//     `withCode(err error, code string) error` that returns nil for a nil err.
//
//   - `codeOf(err) string` — explicit code wins (errors.As on *codedError), then
//     known sentinels (customer.ErrNotFound -> NOT_FOUND, customer.ErrConflict ->
//     CONFLICT, *customer.ValidationError -> VALIDATION), else INTERNAL.
//
//   - `httpStatus(code) int` — NOT_FOUND->404, VALIDATION->400, CONFLICT->409,
//     UNAUTHORIZED->401, default->500.
//
//   - DTOs `errorDetail { Code, Message string }` and `errorBody { Error
//     errorDetail }` (json tags code/message/error), plus `respondError(w, err)`:
//     resolve code+status, replace the message with "internal server error" for
//     500s (never leak internals), and writeJSON the envelope.
//
// ----- Safe JSON (json.go) -----
//
//   - `writeJSON(w, status, v)` — set Content-Type, WriteHeader, then encode.
//   - `decodeJSON[T any](w, r) (T, error)` — cap the body (MaxBytesReader),
//     DisallowUnknownFields, decode one object, and error if there is trailing
//     content (dec.More()).
//
// ----- Middleware (middleware.go) -----
//
//   - `RequestID` — read/generate an X-Request-ID, echo it on the response, and
//     stash it in the context; `RequestIDFrom(ctx) string` reads it back.
//   - `Logging(log)` — log method/path/status/duration/request_id (capture the
//     status with a wrapped ResponseWriter).
//   - `Recover(log)` — recover panics, log with stack, respond 500 JSON.
//   - `BearerAuth(token)` — require "Authorization: Bearer <token>"; otherwise
//     respondError with CodeUnauthorized (401).
//
// ----- Handler + routes (handler.go) -----
//
//   - `Handler` holding *customer.Service, *slog.Logger, and the auth token.
//   - `NewHandler(svc *customer.Service, log *slog.Logger, token string) *Handler`.
//   - `(*Handler) Routes() http.Handler` — a chi router with middleware ordered
//     Recover -> RequestID -> Logging; GET /healthz; and a /v1 group protected by
//     BearerAuth exposing POST /customers, GET /customers, GET /customers/{id}.
//   - DTOs `createCustomerRequest { Email, Name }` and `customerResponse { ID,
//     Email, Name, CreatedAt }` (json tags id/email/name/created_at, CreatedAt as
//     RFC3339). A `toResponse(c, redactPII bool)` that redacts the email local
//     part to "a***@pulse.dev" unless PII is allowed.
//   - Handlers: createCustomer (decode -> Validate -> svc.Create -> 201),
//     getCustomer (svc.Get -> 200/404), listCustomers (svc.List + paginate via
//     ?page=&?size=, returning an items/page/total_pages/total/has_next envelope).
//   - PII rule: redact unless the request carries `X-Scope: pii`.
//
// Delete this comment block as you implement. The package will not compile until
// these types and functions exist.
package httpapi

// TODO: implement the code constants, codedError/withCode/codeOf/httpStatus,
// errorDetail/errorBody/respondError, writeJSON/decodeJSON, RequestID/
// RequestIDFrom/Logging/Recover/BearerAuth, and Handler/NewHandler/Routes with
// its DTOs, handlers, and pagination.
