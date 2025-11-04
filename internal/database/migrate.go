package database

import (
	"database/sql"
	"fmt"
)

// Migrate ensures the required tables exist in the PostgreSQL database.
func Migrate(db *sql.DB) error {
	const createRestaurants = `
CREATE TABLE IF NOT EXISTS restaurants (
	id SERIAL PRIMARY KEY,
	code TEXT NOT NULL,
	name TEXT NOT NULL,
	address TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMPTZ
);`

	if _, err := db.Exec(createRestaurants); err != nil {
		return fmt.Errorf("create restaurants table: %w", err)
	}

	const createUsers = `
CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	restaurant_id INT REFERENCES restaurants(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	if _, err := db.Exec(createUsers); err != nil {
		return fmt.Errorf("create users table: %w", err)
	}

	const ensureRestaurantIDColumn = `
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS restaurant_id INT REFERENCES restaurants(id);`

	if _, err := db.Exec(ensureRestaurantIDColumn); err != nil {
		return fmt.Errorf("ensure users.restaurant_id column: %w", err)
	}

	const createIngredients = `
CREATE TABLE IF NOT EXISTS ingredients (
	id SERIAL PRIMARY KEY,
	code TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMPTZ
);`

	if _, err := db.Exec(createIngredients); err != nil {
		return fmt.Errorf("create ingredients table: %w", err)
	}

	const createOrders = `
CREATE TABLE IF NOT EXISTS orders (
	id SERIAL PRIMARY KEY,
	code TEXT NOT NULL UNIQUE,
	restaurant_id INT NOT NULL,
	ingredient_id INT NOT NULL,
	number INT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMPTZ,
	CONSTRAINT fk_orders_restaurant FOREIGN KEY (restaurant_id) REFERENCES restaurants(id),
	CONSTRAINT fk_orders_ingredient FOREIGN KEY (ingredient_id) REFERENCES ingredients(id)
);`

	if _, err := db.Exec(createOrders); err != nil {
		return fmt.Errorf("create orders table: %w", err)
	}

	return nil
}
