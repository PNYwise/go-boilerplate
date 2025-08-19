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
// It returns a pointer to the sql.DB instance or an error if the connection fails.
// The connection string includes options for parsing time and setting read/write timeouts.
// It also ensures that the database connection is healthy by pinging it before returning.
// The connection pool is configured with maximum open connections, idle connections, and connection lifetime settings.
// The jitter is added to the connection max lifetime to avoid thundering herd problems.
// The function uses the context package to set a timeout for the ping operation to ensure that the connection is responsive.
// If the ping fails, it closes the database connection and returns an error.
// The function is designed to be used in applications that require a MySQL database connection with specific configurations.
// It is a common pattern in Go applications to manage database connections efficiently.
// The function is part of the dbs package, which contains database-related functionalities.
// It is expected to be called during the application initialization phase to set up the database connection.
// The function is designed to be flexible and can be adapted for different MySQL configurations as needed.
func NewMySQLDB(cfg configs.Config) (*sql.DB, error) {
	// Add timeouts & parseTime
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=5s&readTimeout=5s&writeTimeout=5s",
		cfg.DbUser, cfg.DbPassword, cfg.DbHost, cfg.DbPort, cfg.DbName,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DbMaxOpenConns)
	db.SetMaxIdleConns(cfg.DbMaxIdleConns)

	jitter := time.Duration(rand.Intn(5)) * time.Minute
	db.SetConnMaxLifetime(time.Duration(cfg.DbConnMaxLifetime)*time.Minute - jitter)
	db.SetConnMaxIdleTime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
