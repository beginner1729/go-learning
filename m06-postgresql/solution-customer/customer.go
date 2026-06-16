// Package customer holds the domain plus a Postgres-backed Repository that maps
// driver errors to domain errors. Same interface the M4 HTTP service used.
package customer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ID string
type Email string

var (
	ErrNotFound = errors.New("customer: not found")
	ErrConflict = errors.New("customer: already exists")
)

type Customer struct {
	ID        ID
	Email     Email
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c Customer) Validate() error {
	if !strings.Contains(string(c.Email), "@") {
		return fmt.Errorf("email must contain @")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("name must not be empty")
	}
	return nil
}

type Repository interface {
	Create(ctx context.Context, c Customer) (Customer, error)
	ByID(ctx context.Context, id ID) (Customer, error)
	List(ctx context.Context, limit, offset int) ([]Customer, error)
}

// PostgresRepo implements Repository over a pgx pool.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }

func newID() ID {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return ID("cus_" + hex.EncodeToString(b))
}

func (r *PostgresRepo) Create(ctx context.Context, c Customer) (Customer, error) {
	const q = `
		INSERT INTO customers (id, email, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, name, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, string(newID()), string(c.Email), c.Name)
	out, err := scanCustomer(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return Customer{}, ErrConflict
	}
	if err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return out, nil
}

func (r *PostgresRepo) ByID(ctx context.Context, id ID) (Customer, error) {
	const q = `SELECT id, email, name, created_at, updated_at FROM customers WHERE id = $1`
	out, err := scanCustomer(r.pool.QueryRow(ctx, q, string(id)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("byID: %w", err)
	}
	return out, nil
}

func (r *PostgresRepo) List(ctx context.Context, limit, offset int) ([]Customer, error) {
	if limit <= 0 {
		limit = 20
	}
	const q = `
		SELECT id, email, name, created_at, updated_at
		FROM customers ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list query: %w", err)
	}
	defer rows.Close()

	var out []Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanCustomer(s scanner) (Customer, error) {
	var c Customer
	err := s.Scan(&c.ID, &c.Email, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// inTx runs fn inside a transaction with guaranteed rollback on error/panic.
func (r *PostgresRepo) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // harmless after a successful Commit
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CreateCampaign + EnrollCustomer demonstrate transactional, multi-statement work.
func (r *PostgresRepo) CreateCampaign(ctx context.Context, name string) (string, error) {
	id := "cmp_" + func() string { b := make([]byte, 6); _, _ = rand.Read(b); return hex.EncodeToString(b) }()
	_, err := r.pool.Exec(ctx, `INSERT INTO campaigns (id, name) VALUES ($1, $2)`, id, name)
	if err != nil {
		return "", fmt.Errorf("create campaign: %w", err)
	}
	return id, nil
}

func (r *PostgresRepo) EnrollCustomer(ctx context.Context, customerID ID, campaignID string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO enrollments (customer_id, campaign_id) VALUES ($1, $2)`,
			string(customerID), campaignID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				switch pgErr.Code {
				case "23503": // foreign_key_violation -> unknown customer/campaign
					return ErrNotFound
				case "23505": // already enrolled
					return ErrConflict
				}
			}
			return fmt.Errorf("insert enrollment: %w", err)
		}
		_, err = tx.Exec(ctx,
			`UPDATE campaigns SET enrolled_count = enrolled_count + 1 WHERE id = $1`, campaignID)
		if err != nil {
			return fmt.Errorf("bump count: %w", err) // triggers rollback of the insert
		}
		return nil
	})
}

func (r *PostgresRepo) CampaignEnrolledCount(ctx context.Context, campaignID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT enrolled_count FROM campaigns WHERE id = $1`, campaignID).Scan(&n)
	return n, err
}
