package dbs

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	"log"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
)

type RabbitMQConnection struct {
	cfg       configs.Config
	conn      *amqp.Connection
	mu        sync.RWMutex
	connected atomic.Bool
	closeCh   chan struct{}
}

// NewRabbitMQConnection creates a singleton resilient RabbitMQ connection
func NewRabbitMQConnection(lc fx.Lifecycle, cfg configs.Config) (*RabbitMQConnection, error) {
	// Graceful degradation: if no URL is provided, skip connecting
	if cfg.RabbitURL == "" {
		log.Println("RabbitMQ URL is not configured. Messaging will be disabled.")
		return &RabbitMQConnection{}, nil
	}

	rmq := &RabbitMQConnection{
		cfg:     cfg,
		closeCh: make(chan struct{}),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := rmq.connect(); err != nil {
				return fmt.Errorf("failed to connect to rabbitmq: %w", err)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(rmq.closeCh)
			return rmq.Close()
		},
	})

	return rmq, nil
}

func (r *RabbitMQConnection) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	connName := generateConnName(r.cfg.AppName, "rabbitmq")
	conn, err := amqp.DialConfig(r.cfg.RabbitURL, amqp.Config{
		Properties: amqp.Table{
			"connection_name": connName,
		},
	})
	if err != nil {
		return err
	}

	r.conn = conn
	r.connected.Store(true)
	
	log.Printf("RabbitMQ client connected (Name: %s)", connName)

	log.Println("RabbitMQ connected successfully")

	// Listen for unexpected connection drops
	go r.reconnectLoop(conn.NotifyClose(make(chan *amqp.Error)))

	return nil
}

func (r *RabbitMQConnection) reconnectLoop(notifyClose <-chan *amqp.Error) {
	select {
	case err := <-notifyClose:
		if err != nil {
			log.Printf("RabbitMQ connection closed: %v. Attempting to reconnect...", err)
			r.connected.Store(false)
			r.reconnect()
		}
	case <-r.closeCh:
		// Normal shutdown
		return
	}
}

func (r *RabbitMQConnection) reconnect() {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-r.closeCh:
			return
		default:
			err := r.connect()
			if err == nil {
				return
			}

			log.Printf("Failed to reconnect to RabbitMQ: %v. Retrying in %v...", err, backoff)
			time.Sleep(backoff)

			// Exponential backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// Channel returns a new AMQP channel using the current underlying connection
func (r *RabbitMQConnection) Channel() (*amqp.Channel, error) {
	if !r.IsConnected() {
		return nil, fmt.Errorf("rabbitmq is not connected")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.conn.Channel()
}

// IsConnected returns the current connection state
func (r *RabbitMQConnection) IsConnected() bool {
	return r.connected.Load()
}

// Close gracefully shuts down the connection
func (r *RabbitMQConnection) Close() error {
	if !r.IsConnected() {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.connected.Store(false)
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
