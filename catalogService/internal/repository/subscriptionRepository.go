package repository

import (
	"catalogService/internal/model"

	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrResourceNotFound = errors.New("target resource does not exist")

type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

func (sr *SubscriptionRepository) LinkUserById(ctx context.Context, userId, resourceId string) error {
	query := `
		INSERT INTO subscriptions (user_id, resource_id)
		VALUES ($1, $2) ON CONFLICT DO NOTHING
	`

	_, err := sr.pool.Exec(ctx, query, userId, resourceId)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrResourceNotFound
		}
		return err
	}

	return nil
}

func (sr *SubscriptionRepository) HasAccess(ctx context.Context, userId, resourceId string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM subscriptions WHERE user_id = $1 AND resource_id = $2
		)
	`

	var hasAccess bool

	err := sr.pool.QueryRow(ctx, query, userId, resourceId).Scan(&hasAccess)
	if err != nil {
		return false, err
	}

	return hasAccess, nil
}

func (sr *SubscriptionRepository) GetSubscribers(ctx context.Context, resourceId string) ([]*model.User, error) {
	query := `
		SELECT user_id
		FROM subscriptions
		WHERE resource_id = $1
	`

	dst := make([]*model.User, 0, 100)

	rows, err := sr.pool.Query(ctx, query, resourceId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		user := new(model.User)

		err := rows.Scan(&user.Id)
		if err != nil {
			return nil, fmt.Errorf("unable to read subscribers: %w", err)
		}

		dst = append(dst, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dst, nil
}
