package repositories

import (
	"analytics-service/internal/models"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClickRepository struct {
	pool *pgxpool.Pool
}

func NewClickRepository(pool *pgxpool.Pool) *ClickRepository {
	return &ClickRepository{pool: pool}
}

// InsertBatch inserts a batch of clciks.
func (r *ClickRepository) InsertBatch(ctx context.Context, clicks []models.Click) error {
	if len(clicks) == 0 {
		return nil
	}

	const insertClick = `
		INSERT INTO clicks (code, occurred_at, referrer, user_agent)
		VALUES ($1, $2, $3, $4)`

	batch := &pgx.Batch{}
	for _, click := range clicks {
		batch.Queue(insertClick, click.Code, click.OccurredAt, click.Referrer, click.UserAgent)
	}

	if err := r.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("inserting click events: %w", err)
	}

	return nil
}

// GetCodeAnalytics returns stats for the code.
func (r *ClickRepository) GetCodeAnalytics(ctx context.Context, code string) (*models.CodeAnalytics, error) {
	stats := models.CodeAnalytics{Code: code}

	const statsByCode = `
	SELECT
		count(*),
		count(*) FILTER (WHERE occurred_at > now() - interval '24 hours'),
		max(occurred_at)
	FROM clicks
	WHERE code = $1`

	err := r.pool.QueryRow(ctx, statsByCode, code).Scan(
		&stats.TotalClicks,
		&stats.Last24h,
		&stats.LastClickAt,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching code analytics: %w", err)
	}

	return &stats, nil
}
