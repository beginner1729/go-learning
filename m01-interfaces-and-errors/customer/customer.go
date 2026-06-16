// Package customer is YOUR implementation target for M01 Exercise 3.1.
//
// Goal: make `go test ./customer/...` pass and `go run ./cmd/demo` work.
// The tests in customer_test.go define the exact API you must provide.
// Reference answer key: ../solution-customer/ (and the §3.1 details in
// ../M01-interfaces-and-errors.md). Try it yourself before peeking.
//
// Build, in THIS file (customer.go) — Exercise 3.1:
//
//   - Value types `CustomerID` and `Email`, both string-based. Distinct types
//     (not bare string) make signatures self-documenting and stop
//     argument-order bugs from compiling.
//
//   - `ErrNotFound` — a sentinel error (var, created with errors.New). Callers
//     branch on it with errors.Is. It carries no extra data.
//
//   - `AuditFields` struct with `CreatedAt` and `UpdatedAt time.Time`. EMBED it
//     into Customer (not as a named field) so the fields are promoted.
//
//   - `Customer` struct: ID (CustomerID), Email (Email), Name (string), and an
//     embedded AuditFields.
//
//   - `CustomerRepository` interface with:
//       Create(ctx context.Context, c Customer) (Customer, error)
//       ByID(ctx context.Context, id CustomerID) (Customer, error)
//
//   - `InMemoryRepo` — a concrete, concurrency-safe implementation backed by a
//     map, guarded by a sync.RWMutex (reads dominate). Provide
//     `NewInMemoryRepo() *InMemoryRepo`. Create stamps CreatedAt/UpdatedAt with
//     time.Now().UTC(); ByID returns ErrNotFound when the id is absent.
//
// Validation (ValidationError + Validate) goes in validate.go — Exercise 3.2.
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package customer

// TODO(3.1): implement CustomerID, Email, ErrNotFound, AuditFields, Customer,
// CustomerRepository, InMemoryRepo, NewInMemoryRepo.
