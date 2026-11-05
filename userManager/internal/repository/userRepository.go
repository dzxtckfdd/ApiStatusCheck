package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserAlreadyExists = errors.New("user with this login already exists")
var ErrUserNotFound = errors.New("user not found")

type User struct {
	Id           string    `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (ur *UserRepository) Create(ctx context.Context, login, passwordHash string) error {
	query := `
		INSERT INTO users (login, password_hash) 
		VALUES ($1, $2)
	`

	_, err := ur.pool.Exec(ctx, query, login, passwordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserAlreadyExists
		}
		return err
	}

	return nil
}

func (ur *UserRepository) GetByLogin(ctx context.Context, login string) (*User, error) {
	query := `
	SELECT id, login, password_hash, created_at, updated_at
	FROM users
	WHERE login = $1
	`

	var user User

	err := ur.pool.QueryRow(ctx, query, login).Scan(&user.Id, &user.Login, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
