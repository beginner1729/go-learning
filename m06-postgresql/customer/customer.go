// Package customer is YOUR implementation target for M06.
//
// Goal: make `go test ./customer/... ./db/...` pass (against a real Postgres,
// or skip cleanly when PULSE_PG_DSN is unset) and `go run ./cmd/demo` work.
// The tests in customer_test.go define the exact API you must provide.
// Reference answer key: ../solution-customer/ (and §3, §4, §5 of
// ../M06-postgresql.md). Try it yourself before peeking.
//
// This is the domain plus a Postgres-backed Repository that maps driver errors
// to domain errors — the same interface the M4 HTTP service used. Use pgxpool
// directly (github.com/jackc/pgx/v5/pgxpool); never string-build SQL with user
// input — always use $1, $2 placeholders.
//
// Build, in THIS file (customer.go):
//
//   - Value types `ID` and `Email`, both string-based.
//
//   - Sentinel errors `ErrNotFound` and `ErrConflict` (vars via errors.New).
//     Callers branch on them with errors.Is; the repo maps driver errors to
//     these (pgx.ErrNoRows -> ErrNotFound; SQLSTATE 23505 -> ErrConflict;
//     SQLSTATE 23503 FK violation -> ErrNotFound).
//
//   - `Customer` struct: ID (ID), Email (Email), Name (string), and
//     CreatedAt/UpdatedAt time.Time (populated by the DB via RETURNING).
//
//   - `Validate() error` on Customer: Email must contain "@", Name must not be
//     blank.
//
//   - `Repository` interface with:
//     Create(ctx context.Context, c Customer) (Customer, error)
//     ByID(ctx context.Context, id ID) (Customer, error)
//     List(ctx context.Context, limit, offset int) ([]Customer, error)
//
//   - `PostgresRepo` implementing Repository over a *pgxpool.Pool, with
//     `NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo`.
//
//   - Create — generate an id ("cus_"+random hex), INSERT ... RETURNING the
//     row (one round-trip for DB timestamps), map 23505 -> ErrConflict.
//
//   - ByID — SELECT by id, map pgx.ErrNoRows -> ErrNotFound.
//
//   - List — ORDER BY created_at DESC LIMIT $1 OFFSET $2; default limit to
//     20 when <= 0; defer rows.Close(); return rows.Err() after the loop.
//
//   - Transactional campaign work (§5):
//     CreateCampaign(ctx, name string) (string, error) — INSERT a campaign
//     with a generated "cmp_"+hex id.
//     EnrollCustomer(ctx, customerID ID, campaignID string) error — in ONE
//     transaction (use an inTx closure helper with defer tx.Rollback):
//     insert an enrollment row, then bump campaigns.enrolled_count; roll
//     back if either fails. Map 23503 -> ErrNotFound, 23505 -> ErrConflict.
//     CampaignEnrolledCount(ctx, campaignID string) (int, error) — SELECT the
//     count.
//
// The schema lives in ../migrations/ (shared infrastructure — import it
// unchanged via cxm/m06/migrations; the db package runs it). The UNIQUE(email)
// constraint is what surfaces ErrConflict; the FK on enrollments surfaces the
// rollback path.
//
// Delete this comment block as you implement. The package will not compile
// until the types and functions the tests reference exist.
package customer

// TODO: implement ID, Email, ErrNotFound, ErrConflict, Customer, Validate,
// Repository, PostgresRepo, NewPostgresRepo, Create, ByID, List, CreateCampaign,
// EnrollCustomer, CampaignEnrolledCount.
