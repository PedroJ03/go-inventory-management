package migrations

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/pedroj/go-inventory-management/internal/config"
)

func TestPostgreSQLDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host: "db.example.test", Port: 5432, Name: "inventory/test",
		User: "migration user", Password: "secret&value", SSLMode: "require",
	}

	dsn, err := url.Parse(PostgreSQLDSN(cfg))
	if err != nil {
		t.Fatalf("PostgreSQLDSN() returned invalid URL: %v", err)
	}
	if dsn.Scheme != "postgres" || dsn.Host != "db.example.test:5432" || dsn.Path != "/inventory/test" {
		t.Fatalf("unexpected connection URL: %s", dsn)
	}
	if username := dsn.User.Username(); username != cfg.User {
		t.Errorf("username = %q, want %q", username, cfg.User)
	}
	password, _ := dsn.User.Password()
	if password != cfg.Password {
		t.Errorf("password = %q, want %q", password, cfg.Password)
	}
	if got := dsn.Query().Get("sslmode"); got != cfg.SSLMode {
		t.Errorf("sslmode = %q, want %q", got, cfg.SSLMode)
	}
}

func TestSourceURLUsesAbsoluteMigrationsDirectory(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(baseDir, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}

	sourceURL, err := SourceURL(baseDir)
	if err != nil {
		t.Fatalf("SourceURL() error = %v", err)
	}
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		t.Fatalf("SourceURL() returned invalid URL: %v", err)
	}
	wantPath, _ := filepath.Abs(filepath.Join(baseDir, "migrations"))
	if parsed.Scheme != "file" || parsed.Path != filepath.ToSlash(wantPath) {
		t.Fatalf("source URL = %q, want file URL for %q", sourceURL, wantPath)
	}
}

func TestSourceURLRejectsMissingMigrationsDirectory(t *testing.T) {
	if _, err := SourceURL(t.TempDir()); err == nil {
		t.Fatal("SourceURL() error = nil, want missing directory error")
	}
}
