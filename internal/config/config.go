package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

const DefaultPath = "config.toml"

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
}

type ServerConfig struct {
	Address  string `toml:"address"`
	LogLevel string `toml:"log_level"`
}

type DatabaseConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Name     string `toml:"name"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	SSLMode  string `toml:"ssl_mode"`
}

func Load() (Config, error) {
	return LoadFrom(DefaultPath)
}

func LoadFrom(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", path, err)
	}

	if err := applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("server address must not be empty")
	}
	if c.Server.LogLevel == "" {
		return fmt.Errorf("server log level must not be empty")
	}
	if c.Database.Host == "" || c.Database.Name == "" || c.Database.User == "" {
		return fmt.Errorf("database host, name, and user must not be empty")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.Database.SSLMode == "" {
		return fmt.Errorf("database ssl mode must not be empty")
	}
	return nil
}

func applyEnvironment(cfg *Config) error {
	setString("INVENTORY_SERVER_ADDRESS", &cfg.Server.Address)
	setString("INVENTORY_LOG_LEVEL", &cfg.Server.LogLevel)
	setString("INVENTORY_DB_HOST", &cfg.Database.Host)
	setString("INVENTORY_DB_NAME", &cfg.Database.Name)
	setString("INVENTORY_DB_USER", &cfg.Database.User)
	setString("INVENTORY_DB_PASSWORD", &cfg.Database.Password)
	setString("INVENTORY_DB_SSL_MODE", &cfg.Database.SSLMode)

	if value, ok := os.LookupEnv("INVENTORY_DB_PORT"); ok {
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse INVENTORY_DB_PORT: %w", err)
		}
		cfg.Database.Port = port
	}
	return nil
}

func setString(environment string, destination *string) {
	if value, ok := os.LookupEnv(environment); ok {
		*destination = value
	}
}
