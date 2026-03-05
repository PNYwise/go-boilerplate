package dbs

import (
	"context"
	"database/sql"
	"fmt"
	"go-boilerplate/internal/configs"
	"math/rand"
	"time"
)

// NewMySQLDB initializes a new MySQL database connection with the provided configuration.
// It sets up connection parameters such as user, password, host, port, and database name.
// The function also configures connection timeouts and maximum connection settings.
// It returns a pointer to the sql.DB instance, a cleanup function, and an error if the connection fails.
// The cleanup function should be called to properly close the database connection.
func NewMySQLDB(cfg configs.Config) (*sql.DB, func(), error) {
	// Add timeouts & parseTime
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=5s&readTimeout=5s&writeTimeout=5s",
		cfg.DbUser, cfg.DbPassword, cfg.DbHost, cfg.DbPort, cfg.DbName,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, err
	}

	db.SetMaxOpenConns(cfg.DbMaxOpenConns)
	db.SetMaxIdleConns(cfg.DbMaxIdleConns)

	base := time.Duration(cfg.DbConnMaxLifetime) * time.Minute
	if base > 0 {
		// add small jitter but never go negative
		j := time.Duration(rand.Intn(5)) * time.Minute
		if j < base {
			base -= j
		}
	}
	db.SetConnMaxLifetime(base)
	db.SetConnMaxIdleTime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, err
	}

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup, nil
}
