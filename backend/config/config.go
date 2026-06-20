package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration, loaded once at startup.
// Every subsystem receives the slice of config it needs — no env lookups
// are scattered across packages.
type Config struct {
	Server  ServerConfig
	DB      DBConfig
	Redis   RedisConfig
	Batcher BatcherConfig
}

// ServerConfig controls the HTTP listener.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DBConfig controls the MySQL connection pool.
type DBConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	MaxOpenConns int
	MaxIdleConns int
	ConnLifetime time.Duration
}

// RedisConfig controls the Redis connection.
type RedisConfig struct {
	Addr         string
	PoolSize     int
	MinIdleConns int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// BatcherConfig controls the batch ingestion workers.
type BatcherConfig struct {
	BatchSize     int
	NumWorkers    int
	FlushInterval time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
// It returns an error if any value is malformed.
func Load() (*Config, error) {
	flushMS, err := envInt("FLUSH_INTERVAL_MS", 500)
	if err != nil {
		return nil, fmt.Errorf("FLUSH_INTERVAL_MS: %w", err)
	}

	batchSize, err := envInt("BATCH_SIZE", 2000)
	if err != nil {
		return nil, fmt.Errorf("BATCH_SIZE: %w", err)
	}

	numWorkers, err := envInt("NUM_WORKERS", 8)
	if err != nil {
		return nil, fmt.Errorf("NUM_WORKERS: %w", err)
	}

	maxOpen, err := envInt("DB_MAX_OPEN_CONNS", 50)
	if err != nil {
		return nil, fmt.Errorf("DB_MAX_OPEN_CONNS: %w", err)
	}

	maxIdle, err := envInt("DB_MAX_IDLE_CONNS", 25)
	if err != nil {
		return nil, fmt.Errorf("DB_MAX_IDLE_CONNS: %w", err)
	}

	redisPool, err := envInt("REDIS_POOL_SIZE", 200)
	if err != nil {
		return nil, fmt.Errorf("REDIS_POOL_SIZE: %w", err)
	}

	redisMinIdle, err := envInt("REDIS_MIN_IDLE_CONNS", 50)
	if err != nil {
		return nil, fmt.Errorf("REDIS_MIN_IDLE_CONNS: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port:         env("SERVER_PORT", "8080"),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 120 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		DB: DBConfig{
			Host:         env("DB_HOST", "localhost"),
			Port:         env("DB_PORT", "3306"),
			User:         env("DB_USER", "mailercloud"),
			Password:     env("DB_PASSWORD", "mailercloud"),
			Name:         env("DB_NAME", "mailercloud"),
			MaxOpenConns: maxOpen,
			MaxIdleConns: maxIdle,
			ConnLifetime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			Addr:         env("REDIS_ADDR", "localhost:6379"),
			PoolSize:     redisPool,
			MinIdleConns: redisMinIdle,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		Batcher: BatcherConfig{
			BatchSize:     batchSize,
			NumWorkers:    numWorkers,
			FlushInterval: time.Duration(flushMS) * time.Millisecond,
		},
	}, nil
}

// env reads a string env var with a fallback default.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt reads an integer env var with a fallback default.
// Returns an error only if the value is set but not a valid integer.
func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q for %s", v, key)
	}
	return n, nil
}
