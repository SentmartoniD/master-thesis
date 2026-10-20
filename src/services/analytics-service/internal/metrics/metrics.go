package metrics

import (
	"analytics-service/platform/database"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// map of request counts by status code
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests count by statuss.",
	}, []string{"status"})

	RequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Handler latency.",
		Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.25, 0.5, 1, 2.5, 5, 10},
	})

	// Saturation
	InFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Requests currently being processed.",
	})

	// []string{"outcome"}: ok/no_rows/error
	DBQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_queries_total",
		Help: "Database queries by outcome.",
	}, []string{"outcome"})

	DBQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Time spent executing a database query.",
		Buckets: []float64{0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	})

	// rabbitmq metrics
	ClicksConsumedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "clicks_consumed_total",
		Help: "Click events taken off the queue, by outcome.",
	}, []string{"outcome"})

	ClicksFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "clicks_flush_total",
		Help: "Batch flushes by outcome.",
	}, []string{"outcome"})

	ClicksFlushDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "clicks_flush_duration_seconds",
		Help:    "Time to persist one batch of click events.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	})

	ClicksBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "clicks_batch_size",
		Help:    "Click events per flushed batch.",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})
)

// RegisterPoolMetrics exposes pgxpool statistics.
func Instrument() gin.HandlerFunc {
	return func(c *gin.Context) {
		InFlight.Inc()
		defer InFlight.Dec()

		start := time.Now()
		c.Next()
		RequestDuration.Observe(time.Since(start).Seconds())
		RequestsTotal.WithLabelValues(strconv.Itoa(c.Writer.Status())).Inc()
	}
}

// RegisterDBPoolMetrics exposes pgxpool statistics. The functions are evaluated
// at scrape time, so no background goroutine is needed.
func RegisterDBPoolMetrics() {
	stat := func(f func(*pgxpool.Stat) float64) func() float64 {
		return func() float64 {
			s := database.Stats()
			if s == nil {
				return 0
			}
			return f(s)
		}
	}

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_pool_total_conns", Help: "Connections currently open.",
	}, stat(func(s *pgxpool.Stat) float64 { return float64(s.TotalConns()) }))

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_pool_acquired_conns", Help: "Connections currently in use.",
	}, stat(func(s *pgxpool.Stat) float64 { return float64(s.AcquiredConns()) }))

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "db_pool_max_conns", Help: "Configured pool ceiling.",
	}, stat(func(s *pgxpool.Stat) float64 { return float64(s.MaxConns()) }))

	promauto.NewCounterFunc(prometheus.CounterOpts{
		Name: "db_pool_empty_acquire_total", Help: "Acquires that found no free connection.",
	}, stat(func(s *pgxpool.Stat) float64 { return float64(s.EmptyAcquireCount()) }))

	promauto.NewCounterFunc(prometheus.CounterOpts{
		Name: "db_pool_empty_acquire_wait_seconds_total", Help: "Total time blocked waiting for a connection.",
	}, stat(func(s *pgxpool.Stat) float64 { return s.EmptyAcquireWaitTime().Seconds() }))
}

// Create the label combinations up front so the series exist at zero rather
// than appearing only after the first occurrence.
func init() {
	for _, outcome := range []string{"ok", "malformed"} {
		ClicksConsumedTotal.WithLabelValues(outcome)
	}
	for _, outcome := range []string{"ok", "requeued"} {
		ClicksFlushTotal.WithLabelValues(outcome)
	}
}
