package dbs

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	"log"

	"github.com/go-redsync/redsync/v4"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client connection
func NewRedisClient(cfg configs.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Printf("Redis client connected to: %s:%d (DB: %d)", cfg.RedisHost, cfg.RedisPort, cfg.RedisDB)
	return client, nil
}

// NewRedsync creates a new Redsync instance for distributed locking
func NewRedsync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}
