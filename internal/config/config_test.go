package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromAppliesEnvironmentOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `[server]
address = ":9000"
log_level = "debug"

[database]
host = "db"
port = 5432
name = "inventory"
user = "app"
password = "file-password"
ssl_mode = "require"
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INVENTORY_DB_PASSWORD", "environment-password")
	t.Setenv("INVENTORY_DB_PORT", "6543")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Database.Password != "environment-password" || cfg.Database.Port != 6543 {
		t.Fatalf("environment overrides not applied: %+v", cfg.Database)
	}
}

func TestLoadFromRejectsInvalidEnvironmentPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`[server]
address = ":8080"
log_level = "info"
[database]
host = "localhost"
port = 5432
name = "inventory"
user = "inventory"
ssl_mode = "disable"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INVENTORY_DB_PORT", "not-a-port")

	if _, err := LoadFrom(path); err == nil {
		t.Fatal("LoadFrom() error = nil, want invalid port error")
	}
}
