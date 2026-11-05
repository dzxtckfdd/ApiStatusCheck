package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTokenNotFound = errors.New("token not found")

type RefreshToken struct {
	Id        string    `db:"id"`
	UserId    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
	Revoked   bool      `db:"revoked"`
}

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (tr *RefreshTokenRepository) Create(ctx context.Context, userId string, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := tr.pool.Exec(ctx, query, userId, tokenHash, expiresAt)
	return err
}

func (tr *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	query := `
        SELECT id, user_id, token_hash, expires_at, created_at, revoked
        FROM refresh_tokens
        WHERE token_hash = $1
    `
	var token RefreshToken
	err := tr.pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.Id,
		&token.UserId,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.Revoked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

func (tr *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	query := `
	UPDATE refresh_tokens
	SET revoked = true
	WHERE token_hash = $1
	`

	_, err := tr.pool.Exec(ctx, query, tokenHash)
	return err
}
