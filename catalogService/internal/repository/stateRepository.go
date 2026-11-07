package repository

import (
	"catalogService/internal/model"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStateNotFound = errors.New("resource state not found")

type StateRepository struct {
	pool *pgxpool.Pool
}

func NewStateRepository(pool *pgxpool.Pool) *StateRepository {
	return &StateRepository{pool: pool}
}

func (sr *StateRepository) GetById(ctx context.Context, resourceId string) (*model.State, error) {
	query := `
		SELECT resource_id, is_up, consecutive_failures, consecutive_successes, current_incident_id, last_checked_at
		FROM resource_state
		WHERE resource_id = $1
	`

	var state model.State

	err := sr.pool.QueryRow(ctx, query, resourceId).Scan(
		&state.ResourceId,
		&state.IsUp,
		&state.ConsecutiveFailures,
		&state.ConsecutiveSuccesses,
		&state.CurrentIncidentId,
		&state.LastCheckedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStateNotFound
		}
		return nil, err
	}

	return &state, nil
}

func (sr *StateRepository) UpsertById(ctx context.Context, state *model.State) error {
	query := `
		INSERT INTO resource_state (resource_id, is_up, consecutive_failures, consecutive_successes, current_incident_id, last_checked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (resource_id) DO UPDATE SET
		is_up = EXCLUDED.is_up,
		consecutive_failures = EXCLUDED.consecutive_failures,
		consecutive_successes = EXCLUDED.consecutive_successes,
		current_incident_id = EXCLUDED.current_incident_id,
		last_checked_at = EXCLUDED.last_checked_at
	`

	_, err := sr.pool.Exec(
		ctx,
		query,
		state.ResourceId,
		state.IsUp,
		state.ConsecutiveFailures,
		state.ConsecutiveSuccesses,
		state.CurrentIncidentId,
		state.LastCheckedAt,
	)

	return err
}
