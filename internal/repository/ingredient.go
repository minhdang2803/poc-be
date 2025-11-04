package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// IngredientRepository provides access to ingredient records.
type IngredientRepository struct {
	db *sql.DB
}

// NewIngredient creates a repository backed by the given connection.
func NewIngredient(db *sql.DB) *IngredientRepository {
	return &IngredientRepository{db: db}
}

// Exists checks whether the ingredient with provided id is present.
func (r *IngredientRepository) Exists(ctx context.Context, id int64) (bool, error) {
	const query = `SELECT 1 FROM ingredients WHERE id = $1 LIMIT 1`

	var marker int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&marker)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("scan ingredient: %w", err)
	default:
		return true, nil
	}
}
