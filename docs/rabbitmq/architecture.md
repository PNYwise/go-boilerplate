# RabbitMQ Architecture in Go Boilerplate

This boilerplate implements a highly resilient, enterprise-grade RabbitMQ architecture optimized for **Clean Architecture** and the **Uber Fx** dependency injection framework. 

This document explains how connections, producers, and consumers are structured so your team can easily build new event-driven features.

---

## 1. The Core Infrastructure (The Connection)

**File:** `internal/dbs/rabbitmq.go`

Unlike basic implementations that open new TCP connections for every publisher or consumer, this boilerplate maintains exactly **one singleton TCP connection** to the RabbitMQ server.

- **Resiliency**: The connection has a built-in reconnect loop with exponential backoff. If RabbitMQ restarts, the app automatically reconnects.
- **Multiplexing**: Both Producers and Consumers share this single TCP connection by opening lightweight `amqp.Channel` streams over it. This prevents file-descriptor exhaustion and is the officially recommended best practice for Go AMQP.
- **Fail-Fast Booting**: Through `fx.Invoke`, the application verifies this connection at startup. If RabbitMQ is down, the app refuses to boot, preventing zombie deployments.

---

## 2. The Producer Flow (Outbound)

When the application needs to publish an event (e.g., an HTTP API receives data and needs to trigger a background job), it acts as a **Producer**.

### Architecture
- **Wrapper**: `internal/messaging/producer.go`
- **Role**: Infrastructure dependency.

### How it works:
1. The **Service Layer** (Business Logic) decides an event needs to be published.
2. The Service calls the `messaging.Producer` interface.
3. The Producer:
   - Requests a temporary `amqp.Channel` from the singleton `RabbitMQConnection`.
   - Serializes your Go struct (DTO) into a JSON byte array.
   - **OpenTelemetry Injection**: Automatically injects the active trace context into the AMQP Headers. This ensures that traces span across microservices!
   - Publishes the message.
   - Instantly closes the temporary channel.

### Code Example
```go
// Inside internal/services/your_service.go
func (s *yourService) DoSomething(ctx context.Context, data DTO) error {
    // 1. Do business logic
    
    // 2. Publish to RabbitMQ
    err := s.producer.PublishJSON(ctx, "exchange_name", "routing_key", data)
    return err
}
```

---

## 3. The Consumer Flow (Inbound / Entrypoint)

When the application needs to listen to a queue and process background jobs, it acts as a **Consumer**. 

In Clean Architecture, a Consumer is fundamentally an **Input Gateway / Transport**—exactly like an HTTP Server. Because of this, consumer logic lives in the `transports` layer, completely decoupled from business logic.

### Architecture
- **Consumer Interface**: `internal/messaging/consumer.go`
- **Transport Server**: `internal/transports/rabbitmq/server.go`
- **Routers**: `internal/transports/rabbitmq/routers/`
- **Handlers**: `internal/transports/rabbitmq/handlers/`

### How it works:
1. **Startup**: When the app boots, `server.go` is hooked into the Fx lifecycle. It triggers the `routers`.
2. **Routing**: The router (`audit_routes.go`) declares the queues/bindings to ensure they exist on the RabbitMQ server, and maps them to specific Handlers.
3. **Listening Loop**: The `messaging.Consumer` requests a persistent `amqp.Channel` from the singleton connection and starts a background goroutine listening to `<-msgs`.
4. **Handling (The Worker)**: When a message arrives, it hits the Handler (`audit_worker.go`).
   - **Trace Extraction**: The handler extracts the OpenTelemetry trace from the headers to continue the distributed trace.
   - **Deserialization**: It parses the JSON back into a Go struct.
   - **Handoff**: It passes the struct into the **Service Layer** to do the actual business logic.
5. **Ack/Nack**: If the Service returns `nil`, the message is Ack'd (success). If the Service returns an `error`, the message is Nack'd and requeued.

### Adding a New Consumer
To add a new worker to the application, follow these steps:
1. Create a worker in `transports/rabbitmq/handlers/my_worker.go`.
2. Add a route in `transports/rabbitmq/routers/my_routes.go`.
3. Register the route in `transports/rabbitmq/server.go`.
4. Provide the worker in the `HandlerModule` inside `internal/apps/fx.go`.

---

## 4. End-to-End Example: The Audit Log System

To see this in action, follow the Audit Log flow:

1. **HTTP In**: Client sends a POST to `/api/v1/audit-logs/` (`transports/http/handlers/audit_handler.go`).
2. **Business Logic**: `AuditService.PublishAuditLog` is called. It validates the DTO.
3. **Produce**: `AuditService` calls `Producer.PublishJSON` to push it to `audit_logs_queue`.
4. **Consume**: Instantly, the `RabbitMQServer` (which is listening in the background) catches it.
5. **Handle**: `transports/rabbitmq/handlers/audit_worker.go` receives the AMQP delivery, unpacks it, and calls `AuditService.ProcessIncomingAuditLog`.
6. **Log/Save**: The Service processes the background task.

This ensures complete decoupling: The HTTP layer doesn't know about RabbitMQ, and the Consumer layer doesn't know about the database. Everything meets cleanly in the Service layer!
