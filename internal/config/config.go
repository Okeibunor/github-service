package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Database struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
		SSLMode  string
	}
	GitHub struct {
		Token    string
		Repo     string
		Since    time.Time
		Interval time.Duration
	}
	Server ServerConfig
}

type ServerConfig struct {
	Port int
}

func NewConfig() *Config {
	return &Config{
		Database: struct {
			Host     string
			Port     int
			User     string
			Password string
			Name     string
			SSLMode  string
		}{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Name:     "github_service",
			SSLMode:  "disable",
		},
		GitHub: struct {
			Token    string
			Repo     string
			Since    time.Time
			Interval time.Duration
		}{
			Interval: time.Hour,
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Environment variables
	v.SetEnvPrefix("GITHUB_SERVICE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")

	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.url", "github_service.db")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")

	v.SetDefault("github.rate_limit", "1s")
	v.SetDefault("github.request_timeout", "30s")
	v.SetDefault("github.max_retries", 3)
	v.SetDefault("github.retry_backoff", "2s")

	v.SetDefault("monitor.interval", "1h")
	v.SetDefault("monitor.enabled", true)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.SSLMode == "" {
		return fmt.Errorf("database sslmode is required")
	}

	if c.GitHub.Token == "" {
		return fmt.Errorf("GitHub token is required")
	}

	return nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s sslrootcert=certs/ca.pem",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}
