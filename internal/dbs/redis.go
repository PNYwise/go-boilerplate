package dbs

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	"log"
	"time"

	"github.com/go-redsync/redsync/v4"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client connection
func NewRedisClient(cfg configs.Config) (*redis.Client, error) {
	connName := generateConnName(cfg.AppName, "redis")
	client := redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password:   cfg.RedisPassword,
		DB:         cfg.RedisDB,
		ClientName: connName,
	})

	// Health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Printf("Redis client connected to: %s:%d (DB: %d, Name: %s)", cfg.RedisHost, cfg.RedisPort, cfg.RedisDB, connName)
	return client, nil
}

// NewRedsync creates a new Redsync instance for distributed locking
func NewRedsync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}
