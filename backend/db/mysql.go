package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"mailercloud-backend/config"
)

// NewMySQL opens a connection pool to MySQL using the provided configuration.
// It retries the connection up to 30 times to handle Docker startup races.
func NewMySQL(cfg config.DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&interpolateParams=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Connection pool tuning (see DESIGN.md §5)
	// Scale with NUM_WORKERS: each worker holds 1 tx + 1 idle conn overhead.
	database.SetMaxOpenConns(cfg.MaxOpenConns)
	database.SetMaxIdleConns(cfg.MaxIdleConns)
	database.SetConnMaxLifetime(cfg.ConnLifetime)

	// Verify connectivity with retries (Docker startup race)
	for i := 0; i < 30; i++ {
		if err := database.Ping(); err == nil {
			slog.Info("mysql connected",
				"host", cfg.Host,
				"port", cfg.Port,
				"max_open", cfg.MaxOpenConns,
				"max_idle", cfg.MaxIdleConns,
			)
			return database, nil
		}
		slog.Warn("waiting for mysql",
			"attempt", i+1,
			"max_attempts", 30,
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to MySQL after 30 retries")
}
