package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	JWT    JWTConfig    `yaml:"jwt"`
}

type ServerConfig struct {
	Port                   int `yaml:"port"`
	ShutdownTimeoutSeconds int `yaml:"shutdown_timeout_seconds"`
}

// JWTConfig: use JWT_SECRET from env in production (no default).
type JWTConfig struct {
	TokenExpirySeconds int `yaml:"token_expiry_seconds"`
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:                   8080,
			ShutdownTimeoutSeconds: 10,
		},
		JWT: JWTConfig{
			TokenExpirySeconds: 3600, // 1 hour
		},
	}
}

// Load merges config from path with defaults. Missing file returns default config.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func applyDefaults(c *Config) {
	if c.Server.Port <= 0 {
		c.Server.Port = 8080
	}
	if c.Server.ShutdownTimeoutSeconds <= 0 {
		c.Server.ShutdownTimeoutSeconds = 10
	}
	if c.JWT.TokenExpirySeconds <= 0 {
		c.JWT.TokenExpirySeconds = 3600
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}

func (c *Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.Server.ShutdownTimeoutSeconds) * time.Second
}

func (c *Config) TokenExpiry() time.Duration {
	return time.Duration(c.JWT.TokenExpirySeconds) * time.Second
}
