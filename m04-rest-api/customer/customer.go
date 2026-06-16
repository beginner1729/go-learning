// Package customer is YOUR implementation target for M04. It carries the domain
// (trimmed from M1) plus a thin service the HTTP layer depends on.
//
// Goal: make `go test ./customer/... ./httpapi/...` pass and `go run ./cmd/api`
// work. The handler tests in ../httpapi/handler_test.go drive the exact API the
// httpapi package — and therefore this package — must expose. Reference answer
// key: ../solution-customer/ (and the §3/§5 details in ../M04-rest-api.md). Try
// it yourself before peeking.
//
// Build, in THIS file (customer.go):
//
//   - Value types `ID` and `Email`, both string-based (distinct types make
//     signatures self-documenting and stop argument-order bugs from compiling).
//
//   - Sentinel errors (var, errors.New):
//     ErrNotFound — id absent from the repo.
//     ErrConflict — an email already exists.
//     Callers branch on these with errors.Is; httpapi maps them to 404/409.
//
//   - `ValidationError` struct { Field, Msg string } with a pointer Error()
//     method `func (e *ValidationError) Error() string`. httpapi detects it with
//     errors.As and maps it to 400 VALIDATION.
//
//   - `AuditFields` struct with `CreatedAt` and `UpdatedAt time.Time`. EMBED it
//     into Customer (not a named field) so the fields are promoted.
//
//   - `Customer` struct: ID (ID), Email (Email), Name (string), embedded
//     AuditFields. A method `func (c Customer) Validate() error` that returns the
//     joined (errors.Join) result of validating email (must contain "@") and name
//     (must not be blank after TrimSpace); each failure is a *ValidationError.
//
//   - `Repository` interface — the contract the service depends on (swapped for
//     Postgres in M6):
//     Create(ctx context.Context, c Customer) (Customer, error)
//     ByID(ctx context.Context, id ID) (Customer, error)
//     List(ctx context.Context) ([]Customer, error)
//
//   - `InMemoryRepo` — concurrency-safe, map-backed, guarded by a sync.RWMutex,
//     with a monotonic seq counter. Provide `NewInMemoryRepo() *InMemoryRepo`.
//     Create: reject duplicate emails with ErrConflict, assign ID
//     fmt.Sprintf("cus_%04d", seq), stamp CreatedAt/UpdatedAt with
//     time.Now().UTC(), store and return the customer. ByID: ErrNotFound when
//     absent. List: snapshot of all stored customers.
//
//   - `Service` struct wrapping a Repository, with
//     `NewService(repo Repository) *Service` and methods
//     Create(ctx, c) (Customer, error), Get(ctx, id) (Customer, error),
//     List(ctx) ([]Customer, error) that delegate to the repo.
//
// Delete this comment block as you implement. The package will not compile until
// the types and functions httpapi and the tests reference exist.
package customer

// TODO: implement ID, Email, ErrNotFound, ErrConflict, ValidationError,
// AuditFields, Customer (+ Validate), Repository, InMemoryRepo, NewInMemoryRepo,
// Service, NewService.
