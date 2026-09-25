package migrations

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/pedroj/go-inventory-management/internal/config"
)

// Run applies all pending SQL migrations from the repository migrations directory.
func Run(cfg config.Config, baseDir string) error {
	sourceURL, err := SourceURL(baseDir)
	if err != nil {
		return err
	}

	databaseURL := PostgreSQLDSN(cfg.Database)
	migration, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	defer func() {
		_, _ = migration.Close()
	}()

	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// PostgreSQLDSN builds a PostgreSQL URL without exposing credentials to logs.
func PostgreSQLDSN(cfg config.DatabaseConfig) string {
	databaseURL := &url.URL{
		Scheme: "postgres",
		Host:   cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Path:   "/" + cfg.Name,
		User:   url.UserPassword(cfg.User, cfg.Password),
	}
	query := databaseURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}

// SourceURL returns an absolute file URL so migration loading does not depend on
// a later working-directory change. The base directory follows the config.toml
// convention: commands are expected to start in the project or deployment root.
func SourceURL(baseDir string) (string, error) {
	path, err := filepath.Abs(filepath.Join(baseDir, "migrations"))
	if err != nil {
		return "", fmt.Errorf("resolve migrations directory: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat migrations directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("migrations path %q is not a directory", path)
	}
	return (&url.URL{Scheme: "file", Path: path}).String(), nil
}
