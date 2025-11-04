package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// RestaurantRepository provides access to restaurant records.
type RestaurantRepository struct {
	db *sql.DB
}

// NewRestaurant creates a repository backed by the given connection.
func NewRestaurant(db *sql.DB) *RestaurantRepository {
	return &RestaurantRepository{db: db}
}

// Exists checks whether a restaurant with the provided id is present.
func (r *RestaurantRepository) Exists(ctx context.Context, id int64) (bool, error) {
	const query = `SELECT 1 FROM restaurants WHERE id = $1 LIMIT 1`

	var marker int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&marker)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("scan restaurant: %w", err)
	default:
		return true, nil
	}
}

// GetName returns the restaurant name for the provided id.
func (r *RestaurantRepository) GetName(ctx context.Context, id int64) (string, error) {
	const query = `SELECT name FROM restaurants WHERE id = $1`

	var name string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", fmt.Errorf("get restaurant name: %w", err)
	}

	return name, nil
}
