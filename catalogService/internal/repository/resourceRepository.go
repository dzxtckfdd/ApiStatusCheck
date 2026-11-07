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
		SELECT id, url, name, check_interval_sec, is_active, is_public, created_at
		FROM resources
		WHERE id = $1
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
		SELECT r.id, r.url, r.name, r.check_interval_sec, r.is_active, r.is_public, r.created_at
		FROM resources r
		INNER JOIN subscriptions s ON r.id = s.resource_id
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
		); err != nil {
			return nil, err
		}

		resources = append(resources, &r)
	}

	return resources, rows.Err()
}

func (rr *ResourceRepository) GetPublic(ctx context.Context) ([]*model.Resource, error) {
	query := `
		SELECT id, url, name, check_interval_sec, is_active, is_public, created_at
		FROM resources
		WHERE is_public = true AND is_active = true
		ORDER BY name ASC
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
		); err != nil {
			return nil, err
		}

		resources = append(resources, &r)
	}

	return resources, rows.Err()
}

func (rr *ResourceRepository) GetActiveTargets(ctx context.Context) ([]*model.Resource, error) {
	query := `
		SELECT id, url, name, check_interval_sec, is_active, is_public, created_at
		FROM resources
		WHERE is_active = true
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
		); err != nil {
			return nil, err
		}

		resources = append(resources, &r)
	}

	return resources, rows.Err()
}
