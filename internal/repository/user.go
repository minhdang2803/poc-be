package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrConflict indicates a user with the same username already exists.
var ErrConflict = errors.New("user already exists")

// ErrNotFound indicates no user record matched the query.
var ErrNotFound = errors.New("user not found")

// User represents the persistence model.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	RestaurantID int64
	CreatedAt    time.Time
}

// UserRepository persists users.
type UserRepository struct {
	db *sql.DB
}

// NewUser wires the repository to a sql.DB.
func NewUser(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Exists checks whether the username is already stored.
func (r *UserRepository) Exists(ctx context.Context, username string) (bool, error) {
	const query = `SELECT 1 FROM users WHERE username = $1 LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, username)
	var marker int
	switch err := row.Scan(&marker); {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("scan username: %w", err)
	default:
		return true, nil
	}
}

// GetByUsername fetches a user record by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	const query = `SELECT id, username, password_hash, COALESCE(restaurant_id, 0), created_at FROM users WHERE username = $1`

	var user User
	err := r.db.QueryRowContext(ctx, query, username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.RestaurantID, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	user.CreatedAt = user.CreatedAt.UTC()
	return &user, nil
}

// GetByID returns a user by identifier.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	const query = `SELECT id, username, password_hash, COALESCE(restaurant_id, 0), created_at FROM users WHERE id = $1`

	var user User
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.RestaurantID, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	user.CreatedAt = user.CreatedAt.UTC()
	return &user, nil
}

// Create inserts a new user row.
func (r *UserRepository) Create(ctx context.Context, username, passwordHash string, restaurantID int64) (*User, error) {
	const query = `INSERT INTO users (username, password_hash, restaurant_id) VALUES ($1, $2, $3) RETURNING id, restaurant_id, created_at`

	var (
		id        int64
		rID       int64
		createdAt time.Time
	)
	if err := r.db.
		QueryRowContext(ctx, query, username, passwordHash, restaurantID).
		Scan(&id, &rID, &createdAt); err != nil {
		if isConstraintViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		RestaurantID: rID,
		CreatedAt:    createdAt.UTC(),
	}, nil
}

func isConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	// Fallback for other drivers (e.g. SQLite).
	errMsg := err.Error()
	return strings.Contains(errMsg, "UNIQUE constraint failed") || strings.Contains(errMsg, "constraint failed")
}
