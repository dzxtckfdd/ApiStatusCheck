package repository

import (
	"catalogService/internal/model"
	"errors"
	"fmt"

	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmptyBatchInput error = errors.New("empty batch input")

type CheckRepository struct {
	pool *pgxpool.Pool
}

func NewCheckRepository(pool *pgxpool.Pool) *CheckRepository {
	return &CheckRepository{pool: pool}
}

func (cr *CheckRepository) SaveBatch(ctx context.Context, batch []*model.Check) error {
	if len(batch) == 0 {
		return ErrEmptyBatchInput
	}

	b := &pgx.Batch{}

	query := `
		INSERT INTO checks (resource_id, checked_at, status_code, latency_ms, is_up, error_msg)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, c := range batch {
		b.Queue(query, c.ResourceId, c.CheckedAt, c.StatusCode, c.LatencyMs, c.IsUp, c.ErrorMsg)
	}

	br := cr.pool.SendBatch(ctx, b)
	defer br.Close()

	for range batch {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch check insert: %w", err)
		}
	}

	return nil
}

func (cr *CheckRepository) GetNLastChecksById(ctx context.Context, resourceId string, limit int) ([]*model.Check, error) {
	query := `
		SELECT resource_id, checked_at, status_code, latency_ms, is_up, error_msg
		FROM checks
		WHERE resource_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`

	rows, err := cr.pool.Query(ctx, query, resourceId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := make([]*model.Check, 0, limit)

	for rows.Next() {
		var c model.Check
		if err := rows.Scan(
			&c.ResourceId,
			&c.CheckedAt,
			&c.StatusCode,
			&c.LatencyMs,
			&c.IsUp,
			&c.ErrorMsg,
		); err != nil {
			return nil, err
		}

		checks = append(checks, &c)
	}

	return checks, rows.Err()
}
