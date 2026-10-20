package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"
	"urlshortener-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrorCodeUsed     = errors.New("code already used")
	ErrorLinkNotFound = errors.New("link not found")
)

type LinkRepository struct {
	pool *pgxpool.Pool
}

func NewLinkRepository(pool *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{pool: pool}
}

// Create inserts a new row into the links table.
func (r *LinkRepository) Create(ctx context.Context, code string, targetURL string, expiresAt *time.Time) (*models.Link, error) {

	var link models.Link

	const insertLinkQuery = `
		INSERT INTO links (code, target_url, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, code, target_url, expires_at, is_active, created_at`

	err := r.pool.QueryRow(ctx, insertLinkQuery, code, targetURL, expiresAt).Scan(
		&link.ID,
		&link.Code,
		&link.TargetURL,
		&link.ExpiresAt,
		&link.IsActive,
		&link.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		// Code: "23505" is uniqueViolation
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrorCodeUsed
		}
		return nil, fmt.Errorf("failed creating link: %w", err)
	}

	return &link, nil
}

// GetByCode returns link for the code
func (r *LinkRepository) GetByCode(ctx context.Context, code string) (*models.Link, error) {
	var link models.Link

	const selectByCode = `
		SELECT id, code, target_url, expires_at, is_active, created_at
		FROM links
		WHERE code = $1`

	err := r.pool.QueryRow(ctx, selectByCode, code).Scan(
		&link.ID,
		&link.Code,
		&link.TargetURL,
		&link.ExpiresAt,
		&link.IsActive,
		&link.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrorLinkNotFound
		}
		return nil, fmt.Errorf("failed getting link by code: %w", err)
	}

	return &link, nil
}
