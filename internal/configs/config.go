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
// It includes settings for RabbitMQ, database, and other optional features.
// The fields are populated from environment variables or defaults.
// The configuration is loaded using the MustLoad function.
type Config struct {
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

	// OpenTelemetry Configuration
	OtelEnabled        bool
	OtelServiceVersion string
	OtelEnvironment    string
	OtelServiceNodeName string
	OtelOtlpEndpoint   string
	OtelOtlpHeaders    map[string]string
	OtelProtocol       string

	LogLevel string

	AppName  string
	HTTPAddr string
	CorsAllowedOrigins []string
	GrpcAddr string
}

func MustLoad(stage string) Config {
	if stage != "" {
		envFile := fmt.Sprintf(".env.stage.%s", stage)
		loadDotenvIfPresent(envFile)
	} else {
		loadDotenvIfPresent(".env")
	}

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

		// OpenTelemetry Configuration
		OtelEnabled:        getenvBool("OTEL_ENABLED", false),
		OtelServiceVersion: getenv("OTEL_SERVICE_VERSION", "1.0.0"),
		OtelEnvironment:    getenv("OTEL_ENVIRONMENT", "development"),
		OtelServiceNodeName: getenv("OTEL_SERVICE_NODE_NAME", hostnameOrEmpty()),
		OtelOtlpEndpoint:   getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		OtelOtlpHeaders:    parseHeaders(getenv("OTEL_EXPORTER_OTLP_HEADERS", "")),
		OtelProtocol:       getenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http"),

		LogLevel:           getenv("LOG_LEVEL", ""),

		CorsAllowedOrigins: splitCSVDefault(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), []string{"http://localhost:3000"}),
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

func hostnameOrEmpty() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(hostname)
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

// parseHeaders parses OTLP headers from environment variable
// Format: "key1=value1,key2=value2"
func parseHeaders(headerStr string) map[string]string {
	headers := make(map[string]string)
	if headerStr == "" {
		return headers
	}
	
	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		if kv := strings.Split(strings.TrimSpace(pair), "="); len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return headers
}
