package configs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration loaded from environment variables and .env files.
type Config struct {
	RabbitURL         string   `mapstructure:"RABBIT_URL"`
	RabbitExchange    string   `mapstructure:"RABBIT_EXCHANGE"`
	RabbitQueue       string   `mapstructure:"RABBIT_QUEUE"`
	RabbitRoutingKeys []string `mapstructure:"RABBIT_ROUTING_KEYS"`
	RabbitPrefetch    int      `mapstructure:"RABBIT_PREFETCH"`

	RabbitRetryTTLMS      int    `mapstructure:"RABBIT_RETRY_TTL_MS"`
	RabbitMaxRedeliveries int    `mapstructure:"RABBIT_MAX_REDELIVERIES"`
	RabbitDLX             string `mapstructure:"RABBIT_DLX"`
	RabbitRetryExchange   string `mapstructure:"RABBIT_RETRY_EXCHANGE"`

	DbUser     string `mapstructure:"DB_USER"`
	DbPassword string `mapstructure:"DB_PASSWORD"`
	DbHost     string `mapstructure:"DB_HOST"`
	DbPort     int    `mapstructure:"DB_PORT"`
	DbName     string `mapstructure:"DB_NAME"`

	DbMaxOpenConns    int `mapstructure:"DB_MAX_OPEN_CONNS"`
	DbMaxIdleConns    int `mapstructure:"DB_MAX_IDLE_CONNS"`
	DbConnMaxLifetime int `mapstructure:"DB_CONN_MAX_LIFETIME_MIN"`

	ElasticEnabled             bool     `mapstructure:"ELASTIC_ENABLED"`
	ElasticAddresses           []string `mapstructure:"ELASTIC_ADDRESSES"`
	ElasticIndex               string   `mapstructure:"ELASTIC_INDEX"`
	ElasticAPIKey              string   `mapstructure:"ELASTIC_API_KEY"`
	ElasticUsername            string   `mapstructure:"ELASTIC_USERNAME"`
	ElasticPassword            string   `mapstructure:"ELASTIC_PASSWORD"`
	ElasticBulkFlushBytes      int      `mapstructure:"ELASTIC_BULK_FLUSH_BYTES"`
	ElasticBulkFlushIntervalMS int      `mapstructure:"ELASTIC_BULK_FLUSH_INTERVAL_MS"`

	BISPAKEToken string `mapstructure:"BISPAKETOKEN"`

	AppName  string `mapstructure:"APP_NAME"`
	HTTPAddr string `mapstructure:"HTTP_ADDR"`
	GrpcAddr string `mapstructure:"GRPC_ADDR"`
}

// MustLoad loads the configuration from environment variables and .env file.
// It panics if any required configuration is missing or if loading fails.
func MustLoad() Config {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("RABBIT_URL", "amqp://guest:guest@localhost:5672/")
	v.SetDefault("RABBIT_EXCHANGE", "app.events")
	v.SetDefault("RABBIT_QUEUE", "orders.q")
	v.SetDefault("RABBIT_ROUTING_KEYS", []string{"dwh.*", "#"})
	v.SetDefault("RABBIT_PREFETCH", 16)
	v.SetDefault("RABBIT_RETRY_TTL_MS", 15000)
	v.SetDefault("RABBIT_MAX_REDELIVERIES", 5)
	v.SetDefault("RABBIT_DLX", "app.dlx")
	v.SetDefault("RABBIT_RETRY_EXCHANGE", "app.retry")
	v.SetDefault("APP_NAME", "rmq-consumer")
	v.SetDefault("HTTP_ADDR", ":8080")
	v.SetDefault("GRPC_ADDR", ":9090")
	v.SetDefault("ELASTIC_ENABLED", "false")
	v.SetDefault("ELASTIC_ADDRESSES", []string{"http://localhost:9200"})
	v.SetDefault("ELASTIC_INDEX", "logs")
	v.SetDefault("ELASTIC_API_KEY", "")
	v.SetDefault("ELASTIC_USERNAME", "")
	v.SetDefault("ELASTIC_PASSWORD", "")
	v.SetDefault("ELASTIC_BULK_FLUSH_BYTES", 1000000)
	v.SetDefault("ELASTIC_BULK_FLUSH_INTERVAL_MS", 5000)

	v.SetDefault("BISPAKETOKEN", "true")

	// Determine .env file based on STAGE
	stage := os.Getenv("STAGE")
	envFile := ".env"
	if stage != "" {
		envFile = fmt.Sprintf(".env.stage.%s", stage)
	}

	// Load from .env file if exists
	v.SetConfigFile(filepath.Clean(envFile))
	if err := v.ReadInConfig(); err != nil {
		log.Printf("No %s file found, using defaults and env vars", envFile)
	} else {
		log.Printf("Loaded configuration from %s", envFile)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %w", err))
	}

	if cfg.RabbitQueue == "" || cfg.RabbitExchange == "" {
		panic(fmt.Errorf("missing RABBIT_EXCHANGE or RABBIT_QUEUE"))
	}

	log.Printf("config loaded: stage=%s exchange=%s queue=%s keys=%v", stage, cfg.RabbitExchange, cfg.RabbitQueue, cfg.RabbitRoutingKeys)
	return cfg
}
