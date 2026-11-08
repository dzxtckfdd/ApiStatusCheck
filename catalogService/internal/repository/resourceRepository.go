package repository

import (
	"catalogService/internal/model"
	"errors"

	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResourceRepository struct {
	pool *pgxpool.Pool
}

func NewResourceRepository(pool *pgxpool.Pool) *ResourceRepository {
	return &ResourceRepository{pool: pool}
}

func (rr *ResourceRepository) Create(ctx context.Context, url, name string, checkIntervalSec int, isPublic bool) (string, error) {
	query := `
		INSERT INTO resources (url, name, check_interval_sec, is_public)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id string

	err := rr.pool.QueryRow(ctx, query, url, name, checkIntervalSec, isPublic).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (rr *ResourceRepository) GetById(ctx context.Context, id string) (*model.Resource, error) {
	query := `
		SELECT 
			r.id, r.url, r.name, r.check_interval_sec, r.is_active, r.is_public, r.created_at,
			COALESCE(rs.is_up, true) AS is_up,
			rs.last_checked_at
		FROM resources r
		LEFT JOIN resource_state rs ON r.id = rs.resource_id
		WHERE r.id = $1
	`

	var r model.Resource

	err := rr.pool.QueryRow(ctx, query, id).Scan(
		&r.Id,
		&r.Url,
		&r.Name,
		&r.CheckIntervalSec,
		&r.IsActive,
		&r.IsPublic,
		&r.CreatedAt,
		&r.IsUp,
		&r.LastCheckedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	return &r, nil
}

func (rr *ResourceRepository) GetByUrl(ctx context.Context, url string) (*model.Resource, error) {
	query := `
		SELECT 
			r.id, r.url, r.name, r.check_interval_sec, r.is_active, r.is_public, r.created_at,
			COALESCE(rs.is_up, true) AS is_up,
			rs.last_checked_at
		FROM resources r
		LEFT JOIN resource_state rs ON r.id = rs.resource_id
		WHERE r.url = $1
		LIMIT 1
	`

	var r model.Resource

	err := rr.pool.QueryRow(ctx, query, url).Scan(
		&r.Id,
		&r.Url,
		&r.Name,
		&r.CheckIntervalSec,
		&r.IsActive,
		&r.IsPublic,
		&r.CreatedAt,
		&r.IsUp,
		&r.LastCheckedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	return &r, nil
}

func (rr *ResourceRepository) GetByUserId(ctx context.Context, userId string) ([]*model.Resource, error) {
	query := `
		SELECT 
			r.id, r.url, r.name, r.check_interval_sec, r.is_active, r.is_public, r.created_at,
			COALESCE(rs.is_up, true) AS is_up,
			rs.last_checked_at
		FROM resources r
		INNER JOIN subscriptions s ON r.id = s.resource_id
		LEFT JOIN resource_state rs ON r.id = rs.resource_id
		WHERE s.user_id = $1
		ORDER BY r.created_at DESC
	`

	rows, err := rr.pool.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]*model.Resource, 0, 20)

	for rows.Next() {
		var r model.Resource

		if err := rows.Scan(
			&r.Id,
			&r.Url,
			&r.Name,
			&r.CheckIntervalSec,
			&r.IsActive,
			&r.IsPublic,
			&r.CreatedAt,
			&r.IsUp,
			&r.LastCheckedAt,
		); err != nil {
			return nil, err
		}

		resources = append(resources, &r)
	}

	return resources, rows.Err()
}

func (rr *ResourceRepository) GetPublic(ctx context.Context) ([]*model.Resource, error) {
	query := `
		SELECT 
			r.id, r.url, r.name, r.check_interval_sec, r.is_active, r.is_public, r.created_at,
			COALESCE(rs.is_up, true) AS is_up,
			rs.last_checked_at
		FROM resources r
		LEFT JOIN resource_state rs ON r.id = rs.resource_id
		WHERE r.is_public = true AND r.is_active = true
		ORDER BY r.name ASC
	`

	rows, err := rr.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]*model.Resource, 0, 10)

	for rows.Next() {
		var r model.Resource

		if err := rows.Scan(
			&r.Id,
			&r.Url,
			&r.Name,
			&r.CheckIntervalSec,
			&r.IsActive,
			&r.IsPublic,
			&r.CreatedAt,
			&r.IsUp,
			&r.LastCheckedAt,
		); err != nil {
			return nil, err
		}

		resources = append(resources, &r)
	}

	return resources, rows.Err()
}

func (rr *ResourceRepository) GetActiveResources(ctx context.Context) ([]*model.Resource, error) {
	query := `
		SELECT 
			r.id, r.url, r.name, r.check_interval_sec, r.is_active, r.is_public, r.created_at,
			COALESCE(rs.is_up, true) AS is_up,
			rs.last_checked_at
		FROM resources r
		LEFT JOIN resource_state rs ON r.id = rs.resource_id
		WHERE r.is_active = true
	`

	rows, err := rr.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]*model.Resource, 0, 100)

	for rows.Next() {
		var r model.Resource

		if err := rows.Scan(
			&r.Id,
			&r.Url,
			&r.Name,
			&r.CheckIntervalSec,
			&r.IsActive,
			&r.IsPublic,
			&r.CreatedAt,
			&r.IsUp,
			&r.LastCheckedAt,
		); err != nil {
			return nil, err
		}

		resources = append(resources, &r)
	}

	return resources, rows.Err()
}
