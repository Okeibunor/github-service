package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig
	GitHub   GitHubConfig
	Server   ServerConfig
	Monitor  MonitorConfig
	Log      LogConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type GitHubConfig struct {
	Token          string
	RateLimit      time.Duration
	RequestTimeout time.Duration
	MaxRetries     int
	RetryBackoff   time.Duration
	Repo           string        // Optional: specific repository to monitor
	Since          time.Time     // Optional: sync commits since this time
	Interval       time.Duration // Optional: sync interval
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MonitorConfig struct {
	Interval time.Duration
	Enabled  bool
}

type LogConfig struct {
	Level  string
	Format string
}

// Load reads configuration from file and environment variables
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

	// Override with environment variables
	envVars := map[string]string{
		"database.host":     "DB_HOST",
		"database.port":     "DB_PORT",
		"database.user":     "DB_USER",
		"database.password": "DB_PASSWORD",
		"database.name":     "DB_NAME",
		"database.sslmode":  "DB_SSLMODE",
		"github.token":      "GITHUB_TOKEN",
		"monitor.interval":  "MONITOR_INTERVAL",
		"log.level":         "LOG_LEVEL",
		"log.format":        "LOG_FORMAT",
	}

	for configKey, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			v.Set(configKey, value)
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
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "github_service")
	v.SetDefault("database.sslmode", "disable")

	// GitHub defaults
	v.SetDefault("github.rate_limit", "1s")
	v.SetDefault("github.request_timeout", "30s")
	v.SetDefault("github.max_retries", 3)
	v.SetDefault("github.retry_backoff", "2s")

	// Monitor defaults
	v.SetDefault("monitor.interval", "1h")
	v.SetDefault("monitor.enabled", true)

	// Log defaults
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
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}
