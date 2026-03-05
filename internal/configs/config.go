package configs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
// It includes settings for RabbitMQ, database, Elasticsearch, and other optional features.
// The fields are populated from environment variables or defaults.
// The configuration is loaded using the MustLoad function.
type Config struct {
	Mode string

	// Basic Auth
	BasicAuthUser string
	BasicAuthPass string

	// Rabbit
	RabbitURL         string
	RabbitExchange    string
	RabbitQueue       string
	RabbitRoutingKeys []string
	RabbitPrefetch    int

	RabbitRetryTTLMS      int
	RabbitMaxRedeliveries int
	RabbitDLX             string
	RabbitRetryExchange   string

	// DB (optional)
	DbUser     string
	DbPassword string
	DbHost     string
	DbPort     int
	DbName     string

	DbMaxOpenConns    int
	DbMaxIdleConns    int
	DbConnMaxLifetime int

	// DB Connection Timeouts
	DbTimeout      int // seconds
	DbReadTimeout  int // seconds
	DbWriteTimeout int // seconds

	// Elastic (optional)
	ElasticEnabled             bool
	ElasticAddresses           []string
	ElasticIndex               string
	ElasticAPIKey              string
	ElasticUsername            string
	ElasticPassword            string
	ElasticBulkFlushBytes      int
	ElasticBulkFlushIntervalMS int

	// Other (optional)
	BISPAKEToken string

	AppName  string
	HTTPAddr string
	GrpcAddr string
}

// MustLoad loads the configuration from environment variables (seamless, no mode required)
func MustLoad(stage string) Config {
	envFile := ".env"
	if stage != "" {
		envFile = fmt.Sprintf(".env.stage.%s", stage)
	}
	loadDotenvIfPresent(envFile)

	// Build config from environment (with defaults)
	cfg := Config{
		// DB (optional)
		DbUser:            getenv("DB_USER", ""),
		DbPassword:        getenv("DB_PASSWORD", ""),
		DbHost:            getenv("DB_HOST", ""),
		DbPort:            getenvInt("DB_PORT", 0),
		DbName:            getenv("DB_NAME", ""),
		DbMaxOpenConns:    getenvInt("DB_MAX_OPEN_CONNS", 0),
		DbMaxIdleConns:    getenvInt("DB_MAX_IDLE_CONNS", 0),
		DbConnMaxLifetime: getenvInt("DB_CONN_MAX_LIFETIME_MIN", 0),

		// DB Connection Timeouts (default 5 seconds each)
		DbTimeout:      getenvInt("DB_TIMEOUT", 5),
		DbReadTimeout:  getenvInt("DB_READ_TIMEOUT", 5),
		DbWriteTimeout: getenvInt("DB_WRITE_TIMEOUT", 5),

		// Elastic (optional)
		ElasticEnabled:             getenvBool("ELASTIC_ENABLED", false),
		ElasticAddresses:           splitCSVDefault(getenv("ELASTIC_ADDRESSES", ""), []string{"http://localhost:9200"}),
		ElasticIndex:               getenv("ELASTIC_INDEX", "logs"),
		ElasticAPIKey:              getenv("ELASTIC_API_KEY", ""),
		ElasticUsername:            getenv("ELASTIC_USERNAME", ""),
		ElasticPassword:            getenv("ELASTIC_PASSWORD", ""),
		ElasticBulkFlushBytes:      getenvInt("ELASTIC_BULK_FLUSH_BYTES", 1_000_000),
		ElasticBulkFlushIntervalMS: getenvInt("ELASTIC_BULK_FLUSH_INTERVAL_MS", 5000),

		BISPAKEToken: getenv("BISPAKETOKEN", ""),

		AppName:  getenv("APP_NAME", "example"),
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		GrpcAddr: getenv("GRPC_ADDR", ":9090"),
	}

	// Seamless validation - just ensure HTTP basics are set
	requireNonEmpty("HTTP_ADDR", cfg.HTTPAddr)

	log.Printf("config loaded: stage=%s", stage)

	return cfg
}

func loadDotenvIfPresent(filename string) {
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	candidates := []string{
		filepath.Join(cwd, filename),          // run from repo root
		filepath.Join(exeDir, filename),       // near the binary
		filepath.Join(exeDir, "..", filename), // binary in ./bin
		filepath.Join(cwd, "..", filename),    // ran from ./cmd
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err != nil {
				log.Printf("failed loading %s: %v", p, err)
			} else {
				log.Printf("Loaded environment from %s", p)
			}
			return
		}
	}
	log.Printf("No %s file found in candidates: %v (using defaults and env vars)", filename, candidates)
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		// supports: 1, t, T, TRUE, true, True, yes, y / 0, f, false, no, n
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func splitCSVDefault(s string, def []string) []string {
	if strings.TrimSpace(s) == "" {
		return append([]string(nil), def...)
	}
	return splitCSV(s)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func requireNonEmpty(name, val string) {
	if strings.TrimSpace(val) == "" {
		panic(fmt.Errorf("missing %s", name))
	}
}
