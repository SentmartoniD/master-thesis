package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {

	// Service
	ServiceName string

	// Public HTTP server
	ServerHost        string
	ServerPort        int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	// Admin HTTP server
	AdminPort         int
	AdminWriteTimeout time.Duration

	// Shutdown
	ReadinessDrain  time.Duration
	ShutdownTimeout time.Duration

	// Database
	DBHost    string
	DBPort    int
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string

	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration
	DBConnectTimeout  time.Duration

	// RabbitMQ
	RabbitMQURL        string
	RabbitMQExchange   string
	RabbitMQQueue      string
	RabbitMQRoutingKey string
	ConsumerPrefetch   int
	BatchSize          int
	FlushInterval      time.Duration

	// Logging
	LogLevel  string
	LogFormat string

	// Build time flags
	BuildVersion string
	BuildCommit  string
}

func Load(serviceName string) (*Config, error) {
	cfg := &Config{
		ServiceName: serviceName,

		ServerHost:        getEnvString("SERVER_HOST", "0.0.0.0"),
		ServerPort:        getEnvInt("SERVER_PORT", 8000),
		ReadTimeout:       getEnvDuration("SERVER_READ_TIMEOUT", 5*time.Second),
		ReadHeaderTimeout: getEnvDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		WriteTimeout:      getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:       getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),

		AdminPort:         getEnvInt("ADMIN_PORT", 9090),
		AdminWriteTimeout: getEnvDuration("ADMIN_WRITE_TIMEOUT", 10*time.Second),

		ReadinessDrain:  getEnvDuration("SHUTDOWN_READINESS_DRAIN", 5*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),

		DBHost:    getEnvString("DB_HOST", ""),
		DBPort:    getEnvInt("DB_PORT", 5432),
		DBUser:    getEnvString("DB_USER", ""),
		DBPass:    getEnvString("DB_PASS", ""),
		DBName:    getEnvString("DB_NAME", ""),
		DBSSLMode: getEnvString("DB_SSLMODE", "disable"),

		DBMaxConns:        getEnvInt32("DB_MAX_CONNS", 10),
		DBMinConns:        getEnvInt32("DB_MIN_CONNS", 2),
		DBMaxConnLifetime: getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		DBMaxConnIdleTime: getEnvDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		DBConnectTimeout:  getEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),

		RabbitMQURL:        getEnvString("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQExchange:   getEnvString("RABBITMQ_EXCHANGE", "clicks"),
		RabbitMQQueue:      getEnvString("RABBITMQ_QUEUE", "clicks.analytics"),
		RabbitMQRoutingKey: getEnvString("RABBITMQ_ROUTING_KEY", "click.recorded"),
		ConsumerPrefetch:   getEnvInt("CONSUMER_PREFETCH", 1000),
		BatchSize:          getEnvInt("BATCH_SIZE", 500),
		FlushInterval:      getEnvDuration("FLUSH_INTERVAL", 1*time.Second),

		// debug/info/warn/error/fatal
		LogLevel: getEnvString("LOG_LEVEL", "info"),
		// console/json
		LogFormat: getEnvString("LOG_FORMAT", "json"),

		BuildVersion: getEnvString("BUILD_VERSION", "dev"),
		BuildCommit:  getEnvString("BUILD_COMMIT", "unknown"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnvString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt32(key string, fallback int32) int32 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func (c *Config) validate() error {
	var problems []string

	if c.ServerPort < 1 || c.ServerPort > 65535 {
		problems = append(problems, fmt.Sprintf("SERVER_PORT must be 1-65535, got %d", c.ServerPort))
	}
	if c.AdminPort < 1 || c.AdminPort > 65535 {
		problems = append(problems, fmt.Sprintf("ADMIN_PORT must be 1-65535, got %d", c.AdminPort))
	}
	if c.AdminPort == c.ServerPort {
		problems = append(problems, fmt.Sprintf("ADMIN_PORT must differ from SERVER_PORT (both %d)", c.ServerPort))
	}
	if c.DBHost == "" {
		problems = append(problems, "DB_HOST is required")
	}
	if c.DBUser == "" {
		problems = append(problems, "DB_USER is required")
	}
	if c.DBName == "" {
		problems = append(problems, "DB_NAME is required")
	}
	if c.DBPort < 1 || c.DBPort > 65535 {
		problems = append(problems, fmt.Sprintf("DB_PORT must be 1-65535, got %d", c.DBPort))
	}

	switch c.DBSSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		problems = append(problems, fmt.Sprintf("DB_SSLMODE %q is not a valid libpq sslmode", c.DBSSLMode))
	}

	if c.DBMaxConns < 1 {
		problems = append(problems, fmt.Sprintf("DB_MAX_CONNS must be at least 1, got %d", c.DBMaxConns))
	}
	if c.DBMinConns < 0 {
		problems = append(problems, fmt.Sprintf("DB_MIN_CONNS must not be negative, got %d", c.DBMinConns))
	}
	if c.DBMinConns > c.DBMaxConns {
		problems = append(problems, fmt.Sprintf("DB_MIN_CONNS (%d) must not exceed DB_MAX_CONNS (%d)", c.DBMinConns, c.DBMaxConns))
	}

	switch strings.ToLower(c.LogFormat) {
	case "json", "console":
	default:
		problems = append(problems, fmt.Sprintf("LOG_FORMAT %q must be json or console", c.LogFormat))
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}

	return nil
}

func (c *Config) GetPostgreSQLConnectionString() string {
	query := url.Values{}
	query.Set("sslmode", c.DBSSLMode)
	query.Set("application_name", c.ServiceName)

	dbConnection := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPass),
		Host:     fmt.Sprintf("%s:%d", c.DBHost, c.DBPort),
		Path:     "/" + c.DBName,
		RawQuery: query.Encode(),
	}
	return dbConnection.String()
}

// GetMigrationPostgreSQLConnectionString returns the connection string in the scheme golang-migrate's
// pgx/v5 driver registers
func (c *Config) GetMigrationPostgreSQLConnectionString() string {
	cStr := "pgx5://" + strings.TrimPrefix(c.GetPostgreSQLConnectionString(), "postgres://")
	return cStr + "&x-migrations-table=schema_migrations_analytics_service"
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

func (c *Config) AdminAddr() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.AdminPort)
}

// GetFields returns all the config fields.
func (c *Config) GetFields() map[string]any {
	return map[string]any{
		"addr":                 c.Addr(),
		"read_timeout":         c.ReadTimeout.String(),
		"write_timeout":        c.WriteTimeout.String(),
		"idle_timeout":         c.IdleTimeout.String(),
		"readiness_drain":      c.ReadinessDrain.String(),
		"shutdown_timeout":     c.ShutdownTimeout.String(),
		"db_host":              c.DBHost,
		"db_port":              c.DBPort,
		"db_name":              c.DBName,
		"db_user":              c.DBUser,
		"db_sslmode":           c.DBSSLMode,
		"db_max_conns":         c.DBMaxConns,
		"db_min_conns":         c.DBMinConns,
		"db_max_conn_lifetime": c.DBMaxConnLifetime.String(),
		"db_max_conn_idletime": c.DBMaxConnIdleTime.String(),
		"db_connect_timeout":   c.DBConnectTimeout.String(),
		"log_level":            c.LogLevel,
		"log_format":           c.LogFormat,
	}
}
