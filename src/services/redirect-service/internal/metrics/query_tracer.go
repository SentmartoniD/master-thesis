package metrics

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// in pgx
// type QueryTracer interface {
// 	TraceQueryStart(ctx context.Context, conn *Conn, data TraceQueryStartData) context.Context
// 	TraceQueryEnd(ctx context.Context, conn *Conn, data TraceQueryEndData)
// }

// queryStartKey carries the start time from TraceQueryStart to TraceQueryEnd.
type queryStartKey struct{}

// QueryTracer records RED metrics for every query pgx executes.
type QueryTracer struct{}

func (QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, time.Now())
}

func (QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	start, ok := ctx.Value(queryStartKey{}).(time.Time)
	if !ok {
		return
	}

	DBQueryDuration.Observe(time.Since(start).Seconds())

	outcome := "ok"
	switch {
	case data.Err == nil:
	case errors.Is(data.Err, pgx.ErrNoRows):
		outcome = "no_rows"
	default:
		outcome = "error"
	}
	DBQueriesTotal.WithLabelValues(outcome).Inc()
}

func (t *QueryTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, time.Now())
}

// Called once per queued statement. Count each one so a batch of 500 inserts
// reads as 500 queries, matching how the single-query path counts.
func (t *QueryTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	outcome := "ok"
	if data.Err != nil {
		outcome = "error"
	}
	DBQueriesTotal.WithLabelValues(outcome).Inc()
}

// Duration is per batch, not per statement — a batch is one round trip.
func (t *QueryTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	if start, ok := ctx.Value(queryStartKey{}).(time.Time); ok {
		DBQueryDuration.Observe(time.Since(start).Seconds())
	}
}
