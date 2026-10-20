package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrorLinkNotFound = errors.New("link not found")
)

type LinkRepository struct {
	pool *pgxpool.Pool
}

func NewLinkRepository(pool *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{pool: pool}
}

// GetLinkByCode returns link for the specified code
func (r *LinkRepository) GetLinkByCode(ctx context.Context, code string) (string, error) {
	var targetURL string

	const getLinkQuery = `
		SELECT target_url
		FROM links
		WHERE code = $1 AND is_active`

	if err := r.pool.QueryRow(ctx, getLinkQuery, code).Scan(&targetURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrorLinkNotFound
		}
		return "", fmt.Errorf("failed getting link: %w", err)
	}

	return targetURL, nil
}
