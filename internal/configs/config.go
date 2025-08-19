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

// MustLoad loads the configuration from environment variables and returns a Config instance.
// It parses command-line flags for mode and stage, loads a dotenv file if present,
// and validates the configuration based on the selected mode.
// If any required fields are missing, it panics with an error message.
func MustLoad(mode, stage string) Config {
	// ---- 2) Load dotenv file based on stage (no viper) ----
	envFile := ".env"
	if stage != "" {
		envFile = fmt.Sprintf(".env.stage.%s", stage)
	}
	loadDotenvIfPresent(envFile)

	// ---- 3) Build config from environment (with defaults) ----
	cfg := Config{
		Mode: mode,

		// Rabbit defaults
		RabbitURL:         getenv("RABBIT_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:    getenv("RABBIT_EXCHANGE", "app.events"),
		RabbitQueue:       getenv("RABBIT_QUEUE", "orders.q"),
		RabbitRoutingKeys: splitCSVDefault(getenv("RABBIT_ROUTING_KEYS", ""), []string{"dwh.*", "#"}),
		RabbitPrefetch:    getenvInt("RABBIT_PREFETCH", 16),

		RabbitRetryTTLMS:      getenvInt("RABBIT_RETRY_TTL_MS", 15000),
		RabbitMaxRedeliveries: getenvInt("RABBIT_MAX_REDELIVERIES", 5),
		RabbitDLX:             getenv("RABBIT_DLX", "app.dlx"),
		RabbitRetryExchange:   getenv("RABBIT_RETRY_EXCHANGE", "app.retry"),

		// DB (optional)
		DbUser:            getenv("DB_USER", ""),
		DbPassword:        getenv("DB_PASSWORD", ""),
		DbHost:            getenv("DB_HOST", ""),
		DbPort:            getenvInt("DB_PORT", 0),
		DbName:            getenv("DB_NAME", ""),
		DbMaxOpenConns:    getenvInt("DB_MAX_OPEN_CONNS", 0),
		DbMaxIdleConns:    getenvInt("DB_MAX_IDLE_CONNS", 0),
		DbConnMaxLifetime: getenvInt("DB_CONN_MAX_LIFETIME_MIN", 0),

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

		AppName:  getenv("APP_NAME", "rmq-consumer"),
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		GrpcAddr: getenv("GRPC_ADDR", ":9090"),
	}

	// ---- 4) Validate ONLY what the selected mode needs ----
	switch cfg.Mode {
	case "http":
		requireNonEmpty("HTTP_ADDR", cfg.HTTPAddr)
	case "rabbit":
		requireNonEmpty("RABBIT_URL", cfg.RabbitURL)
		requireNonEmpty("RABBIT_EXCHANGE", cfg.RabbitExchange)
		requireNonEmpty("RABBIT_QUEUE", cfg.RabbitQueue)
		if len(cfg.RabbitRoutingKeys) == 0 {
			panic(fmt.Errorf("missing RABBIT_ROUTING_KEYS"))
		}
	default:
		log.Printf("Unknown MODE=%q, falling back to MODE=http validation", cfg.Mode)
		requireNonEmpty("HTTP_ADDR", cfg.HTTPAddr)
	}

	log.Printf("config loaded: stage=%s mode=%s exchange=%s queue=%s keys=%v",
		stage, cfg.Mode, cfg.RabbitExchange, cfg.RabbitQueue, cfg.RabbitRoutingKeys)

	return cfg
}

// --- helpers ---

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
