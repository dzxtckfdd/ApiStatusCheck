package repository

import (
	"catalogService/internal/model"

	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrIncidentNotFound = errors.New("incident not found")

type IncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{pool: pool}
}

func (ir *IncidentRepository) Open(ctx context.Context, resourceId, description string) (string, error) {
	query := `
		INSERT INTO incidents (resource_id, description)
		VALUES ($1, $2)
		RETURNING id
	`

	var id string

	err := ir.pool.QueryRow(ctx, query, resourceId, description).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return "", ErrResourceNotFound
		}
		return "", err
	}

	return id, nil
}

func (ir *IncidentRepository) Resolve(ctx context.Context, incidentId string) error {
	query := `
		UPDATE incidents
		SET resolved_at = now()
		WHERE id = $1
	`

	tag, err := ir.pool.Exec(ctx, query, incidentId)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrIncidentNotFound
	}

	return nil
}

func (ir *IncidentRepository) GetRecentByResourceId(ctx context.Context, resourceId string, limit int) ([]*model.Incident, error) {
	query := `
		SELECT id, resource_id, started_at, resolved_at, description
		FROM incidents
		WHERE resource_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := ir.pool.Query(ctx, query, resourceId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]*model.Incident, 0, 100)

	for rows.Next() {
		var i model.Incident
		if err := rows.Scan(
			&i.Id,
			&i.ResourceId,
			&i.StartedAt,
			&i.ResolvedAt,
			&i.Description,
		); err != nil {
			return nil, err
		}

		incidents = append(incidents, &i)
	}

	return incidents, rows.Err()
}
