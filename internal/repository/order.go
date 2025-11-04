package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Order represents the orders table row.
type Order struct {
	ID           int64
	Code         string
	RestaurantID int64
	IngredientID int64
	Number       int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OrderRepository persists orders.
type OrderRepository struct {
	db *sql.DB
}

// NewOrder wires the repository to a sql.DB.
func NewOrder(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateBulk inserts multiple orders for a restaurant.
func (r *OrderRepository) CreateBulk(ctx context.Context, restaurantID int64, items []Order) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	const query = `
INSERT INTO orders (code, restaurant_id, ingredient_id, number)
VALUES ($1, $2, $3, $4)`

	for _, item := range items {
		_, execErr := tx.ExecContext(ctx, query, item.Code, restaurantID, item.IngredientID, item.Number)
		if execErr != nil {
			tx.Rollback()
			return fmt.Errorf("insert order: %w", execErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit orders: %w", err)
	}

	return nil
}

// ListByRestaurant fetches all orders for a restaurant.
func (r *OrderRepository) ListByRestaurant(ctx context.Context, restaurantID int64) ([]Order, error) {
	const query = `
SELECT id, code, restaurant_id, ingredient_id, number, created_at, updated_at
FROM orders
WHERE restaurant_id = $1
ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var (
			order     Order
			updatedAt sql.NullTime
		)
		if scanErr := rows.Scan(
			&order.ID,
			&order.Code,
			&order.RestaurantID,
			&order.IngredientID,
			&order.Number,
			&order.CreatedAt,
			&updatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan order: %w", scanErr)
		}

		order.CreatedAt = order.CreatedAt.UTC()
		if updatedAt.Valid {
			order.UpdatedAt = updatedAt.Time.UTC()
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

// Get fetches an order by identifier.
func (r *OrderRepository) Get(ctx context.Context, id int64) (*Order, error) {
	const query = `
SELECT id, code, restaurant_id, ingredient_id, number, created_at, updated_at
FROM orders
WHERE id = $1`

	var (
		order     Order
		updatedAt sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.Code,
		&order.RestaurantID,
		&order.IngredientID,
		&order.Number,
		&order.CreatedAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	order.CreatedAt = order.CreatedAt.UTC()
	if updatedAt.Valid {
		order.UpdatedAt = updatedAt.Time.UTC()
	}

	return &order, nil
}
