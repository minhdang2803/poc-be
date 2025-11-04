package database

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresConfig describes the connection settings for a PostgreSQL database.
type PostgresConfig struct {
	URL string
}

// OpenPostgres returns a ready to use PostgreSQL handle.
func OpenPostgres(cfg PostgresConfig) (*sql.DB, error) {
	if cfg.URL == "" {
		return nil, errors.New("database url must not be empty")
	}

	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
