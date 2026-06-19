package messaging

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// headersCarrier adapts amqp.Table to satisfy the propagation.TextMapCarrier interface
type headersCarrier amqp.Table

func (c headersCarrier) Get(key string) string {
	val, ok := c[key]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return ""
	}
	return strVal
}

func (c headersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c headersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
