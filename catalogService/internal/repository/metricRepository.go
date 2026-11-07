package repository

import (
	"catalogService/internal/model"

	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AggregateResolution string

const (
	ResolutionHourly AggregateResolution = "checks_hourly"
	ResolutionDaily  AggregateResolution = "checks_daily"
)

type MetricRepository struct {
	pool *pgxpool.Pool
}

func NewMetricRepository(pool *pgxpool.Pool) *MetricRepository {
	return &MetricRepository{pool: pool}
}

func (mr *MetricRepository) GetSummary(ctx context.Context, resourceId string, resolution AggregateResolution, from, to time.Time) (*model.Metric, error) {
	query := fmt.Sprintf(`
        SELECT 
            COALESCE(
                CASE WHEN SUM(total_checks) > 0 
                     THEN (SUM(up_checks)::float / SUM(total_checks)::float) * 100 
                     ELSE 100.0 
                END, 100.0) AS uptime_percentage,
            COALESCE(AVG(avg_latency_ms), 0) AS avg_latency,
            COALESCE(MIN(min_latency_ms), 0) AS min_latency,
            COALESCE(MAX(max_latency_ms), 0) AS max_latency
        FROM %s
        WHERE resource_id = $1 AND bucket >= $2 AND bucket <= $3
    `, resolution)

	var m model.Metric

	if err := mr.pool.QueryRow(ctx, query, resourceId, from, to).Scan(
		&m.UptimePercentage,
		&m.AvgLatencyMs,
		&m.MinLatencyMs,
		&m.MaxLatencyMs,
	); err != nil {
		return nil, err
	}
	return &m, nil
}
