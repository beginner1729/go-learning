// Package db is YOUR implementation target for M06.
//
// Goal: make `go test ./customer/... ./db/...` pass (or skip cleanly when
// PULSE_PG_DSN is unset) and `go run ./cmd/migrate` work. The customer tests
// call db.Migrate and db.NewPool, so this package must exist for them to build.
// Reference answer key: ../solution-db/ (and §2.2 / §4 of ../M06-postgresql.md).
// Try it yourself before peeking.
//
// Build, in THIS file (db.go):
//
//   - `NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error)` —
//     parse the DSN with pgxpool.ParseConfig, set MaxConns (e.g. 10),
//     MaxConnIdleTime and HealthCheckPeriod, create the pool with
//     pgxpool.NewWithConfig, then Ping to fail fast on a bad connection
//     (Close the pool and return the error if Ping fails).
//
//   - `Migrate(dsn string) error` — apply all pending goose migrations.
//     Open a database/sql handle with the "pgx" driver
//     (blank-import github.com/jackc/pgx/v5/stdlib), then
//     goose.SetBaseFS(migrations.FS), goose.SetDialect("postgres"),
//     goose.Up(sqlDB, "."). The migration SQL lives in the SHARED
//     ../migrations/ package — import it unchanged as "cxm/m06/migrations"
//     (it embeds the *.sql files); do NOT copy or modify it.
//
// Delete this comment block as you implement. The package will not compile
// until NewPool and Migrate exist.
package db

// TODO: implement NewPool and Migrate, importing the shared cxm/m06/migrations
// package (unchanged) for the embedded SQL files.
