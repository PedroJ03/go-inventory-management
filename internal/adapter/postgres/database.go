package postgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	_ "github.com/lib/pq"

	"github.com/pedroj/go-inventory-management/internal/config"
)

// Open creates a PostgreSQL connection pool. It does not change database schema.
func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn := (&url.URL{
		Scheme: "postgres",
		Host:   cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Path:   "/" + cfg.Name,
		User:   url.UserPassword(cfg.User, cfg.Password),
	}).String()
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("build PostgreSQL connection URL: %w", err)
	}
	query := parsed.Query()
	query.Set("sslmode", cfg.SSLMode)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	return db, nil
}
