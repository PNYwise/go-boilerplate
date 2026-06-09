package dbs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"go-boilerplate/internal/configs"
	"log"
	"math/rand"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.opentelemetry.io/otel/attribute"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// NewMySQLDB initializes a new MySQL database connection with OpenTelemetry instrumentation.
// It uses otelsql.Open to automatically trace all database operations including queries,
// transactions, and connection pool metrics. This provides observability into database
// performance and integrates seamlessly with the ELK stack via OpenTelemetry traces.
func NewMySQLDB(cfg configs.Config) (*sql.DB, func(), error) {
	// Build DSN with timeouts & parseTime
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%ds&readTimeout=%ds&writeTimeout=%ds&multiStatements=true",
		cfg.DbUser, cfg.DbPassword, cfg.DbHost, cfg.DbPort, cfg.DbName,
		cfg.DbTimeout, cfg.DbReadTimeout, cfg.DbWriteTimeout,
	)

	// Use otelsql.Open instead of sql.Open for automatic database instrumentation
	// This will trace all SQL operations and send them to OpenTelemetry/ELK stack
	db, err := otelsql.Open("mysql", dsn,
		otelsql.WithAttributes(
			attribute.String("db.system", "mysql"),                                                    // Database type
			attribute.String("db.name", cfg.DbName),                                                   // Database name
			attribute.String("db.user", cfg.DbUser),                                                   // Database user
			attribute.String("server.address", cfg.DbHost),                                            // Database host
			attribute.Int("server.port", cfg.DbPort),                                                  // Database port
			attribute.String("db.connection.pool.name", "main"),                                       // Connection pool identifier
			attribute.Int("db.connection.max_open", cfg.DbMaxOpenConns),                               // Max open connections
			attribute.Int("db.connection.max_idle", cfg.DbMaxIdleConns),                               // Max idle connections
			attribute.String("db.connection.max_lifetime", fmt.Sprintf("%dm", cfg.DbConnMaxLifetime)), // Connection lifetime
			attribute.String("db.timeout", fmt.Sprintf("%ds", cfg.DbTimeout)),                         // Connection timeout
			attribute.String("db.read_timeout", fmt.Sprintf("%ds", cfg.DbReadTimeout)),                // Read timeout
			attribute.String("db.write_timeout", fmt.Sprintf("%ds", cfg.DbWriteTimeout)),              // Write timeout
		),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			DisableErrSkip: true,
			RecordError: func(err error) bool {
				return err != nil && !errors.Is(err, driver.ErrSkip)
			},
		}),
		otelsql.WithSQLCommenter(true), // Add trace_id to SQL comments for correlation
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open instrumented database connection: %w", err)
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
		return nil, nil, fmt.Errorf("database ping failed: %w", err)
	}

	// Run migrations
	migrationDriver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("could not start sql migration driver: %w", err)
	}
	sourceDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("could not create iofs source driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "mysql", migrationDriver)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("could not create migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.Close()
		return nil, nil, fmt.Errorf("could not run sql migration: %w", err)
	}
	log.Printf("Database migrations completed successfully")

	// Register database connection pool metrics for monitoring
	// This tracks connection pool stats (idle, in-use, wait count) in OpenTelemetry metrics
	closeFunc, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		attribute.String("db.system", "mysql"),                          // Database type
		attribute.String("db.name", cfg.DbName),                         // Database name
		attribute.String("db.user", cfg.DbUser),                         // Database user
		attribute.String("server.address", cfg.DbHost),                  // Database host
		attribute.Int("server.port", cfg.DbPort),                        // Database port
		attribute.String("service.name", cfg.AppName),                   // Service name for correlation
		attribute.String("service.version", cfg.OtelServiceVersion),     // Service version
		attribute.String("deployment.environment", cfg.OtelEnvironment), // Environment (dev/prod)
	))
	if err != nil {
		log.Printf("Warning: failed to register DB stats metrics: %v", err)
	}
	_ = closeFunc // Store closeFunc for potential cleanup

	cleanup := func() {
		// Cleanup database connection and metrics
		_ = db.Close()
	}

	log.Printf("MySQL database initialized with OpenTelemetry instrumentation: %s@%s:%d/%s",
		cfg.DbUser, cfg.DbHost, cfg.DbPort, cfg.DbName)

	return db, cleanup, nil
}
