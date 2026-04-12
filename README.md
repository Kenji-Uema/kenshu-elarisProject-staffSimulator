# Staff Simulator

Simulates operational staff work triggered by cleaning requests and time-change events.

## Main Docs

See the main project documentation: <https://kenji-uema.github.io/kenshu-elarisProject-docs/>

## What It Does

- consumes cleaning requests from RabbitMQ and dispatches work by request type
- simulates housekeepers, launderers, and stockers
- updates cottage cleaning state and shared stock in MongoDB
- reacts to day-change and hour-change events

## Interfaces

- RabbitMQ consumers for cleaning, day-change, and hour-change events
- MongoDB repositories for cottage and stock state
- gRPC client to `clockSimulator`

## RabbitMQ Specification

This service is a RabbitMQ consumer only.

- Produces to RabbitMQ: none in the runtime application
- Consumes from RabbitMQ: `q.staff.cleaning`, `q.staff.day-change`, `q.staff.hour-change` by default
- Wire format: Protobuf messages with `content_type=application/protobuf`
- Delivery mode used by the shared producer implementation: persistent
- Ack strategy: manual ack on valid messages, `Nack(requeue=false)` on invalid payloads

The actual queue and binding names are configurable through environment variables.

### Topology Overview

| Queue | Exchange | Routing key | Consumed by | Effect |
| --- | --- | --- | --- | --- |
| `q.staff.cleaning` | `ex.cleaning.request` | `cleaning.request` | `ManagerService` | Dispatches a cleaning job to the internal cleaning worker channel |
| `q.staff.day-change` | `ex.time.event` | `time.event.day` | `TimeEventNotificationService` | Notifies day-change subscribers and triggers restock selection through `ManagerService` |
| `q.staff.hour-change` | `ex.time.event` | `time.event.hour` | `TimeEventNotificationService` | Notifies hour-change subscribers |

### 1. Cleaning Requests

- Queue env: `CLEANING_QUEUE_*`
- Default queue: `q.staff.cleaning`
- Binding env: `CLEANING_BINDING_*`
- Default exchange: `ex.cleaning.request`
- Typical routing key: `cleaning.request`
- Consumer: `ManagerService.Start`
- Internal handoff: writes to `channels.cleaning`

Payload type: `staff.CleaningRequest`

```proto
message CleaningRequest {
  string roomName = 1;
  RequestType request = 2;
}

enum RequestType {
  UNSPECIFIED = 0;
  PREPARE_FOR_GUEST = 1;
  DAILY_CLEANING = 2;
  FULL_CLEANING = 3;
  PREPARE_FOR_SLEEP = 4;
}
```

Human-readable example:

```json
{
  "roomName": "A",
  "request": "PREPARE_FOR_GUEST"
}
```

Accepted request values:

- `PREPARE_FOR_GUEST`
- `DAILY_CLEANING`
- `FULL_CLEANING`
- `PREPARE_FOR_SLEEP`

Validation and delivery behavior:

- `roomName` must not be blank
- `request` must not be `UNSPECIFIED`
- invalid payloads are nacked without requeue
- valid payloads are acked after dispatch to the internal cleaning channel

### 2. Day-Change Events

- Queue env: `DAY_CHANGE_QUEUE_*`
- Default queue: `q.staff.day-change`
- Binding env: `DAY_CHANGE_BINDING_*`
- Default exchange: `ex.time.event`
- Default routing key: `time.event.day`
- Consumer: `TimeEventNotificationService.Start`
- Internal handoff: publishes a `time.Time` event to registered day-change subscribers

Payload type: `event.TimeEvent`

```proto
message TimeEvent {
  google.protobuf.Timestamp time = 1;
}
```

Human-readable example:

```json
{
  "time": "2026-04-12T00:00:00Z"
}
```

Behavior:

- message must contain a non-zero `time`
- invalid payloads are nacked without requeue
- valid payloads are acked, then forwarded to day-change subscribers
- `ManagerService` subscribes to day-change notifications and creates an internal restock request when one arrives

### 3. Hour-Change Events

- Queue env: `HOUR_CHANGE_QUEUE_*`
- Default queue: `q.staff.hour-change`
- Binding env: `HOUR_CHANGE_BINDING_*`
- Default exchange: `ex.time.event`
- Default routing key: `time.event.hour`
- Consumer: `TimeEventNotificationService.Start`
- Internal handoff: publishes a `time.Time` event to registered hour-change subscribers

Payload type: `event.TimeEvent`

Human-readable example:

```json
{
  "time": "2026-04-12T14:00:00Z"
}
```

Behavior:

- message must contain a non-zero `time`
- invalid payloads are nacked without requeue
- valid payloads are acked, then forwarded to hour-change subscribers

### What This Service Produces

It does not publish any RabbitMQ messages in the runtime application.

What it does produce after consuming RabbitMQ messages:

- internal cleaning jobs on `channels.cleaning`
- internal day-change notifications to registered subscribers
- internal hour-change notifications to registered subscribers
- internal restock jobs on `channels.stocker` after a day-change event

The RabbitMQ producer in `internal/infra/mq/rabbitmq_producer.go` exists as shared infrastructure and is used by tests, but it is not wired into the application startup path.

### Publisher Notes For Upstream Services

If another service publishes into these queues through their bound exchanges:

- declare the exchange as `direct`
- publish Protobuf-encoded messages
- set `content_type` to `application/protobuf`
- use the expected routing key for the target queue binding

| Purpose | Exchange | Routing key | Payload |
| --- | --- | --- | --- |
| Request room cleaning | `ex.cleaning.request` | `cleaning.request` | `staff.CleaningRequest` |
| Signal day change | `ex.time.event` | `time.event.day` | `event.TimeEvent` |
| Signal hour change | `ex.time.event` | `time.event.hour` | `event.TimeEvent` |

## Local Commands

```sh
go run ./internal
go build ./internal
go test ./...
make generate
make docker-build
```

## Minimum Env To Start

Optional vars with defaults, such as `SERVICE_NAME`, `VERSION`, collection names, `LOG_LEVEL`, and timeout values, are omitted here.

```sh
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8080

CLEANING_EMPLOYEES_NAMES=alice
LAUNDERING_EMPLOYEES_NAMES=bob
STOCKERS_EMPLOYEES_NAMES=carol

CLOCK_EMU_GRPC_HOST=<clock host>
CLOCK_EMU_GRPC_PORT=50051

MONGO_INITDB_ROOT_USERNAME=<mongo user>
MONGO_INITDB_ROOT_PASSWORD=<mongo password>
MONGO_HOST=<mongo host>
MONGO_DATABASE=cottages

RABBITMQ_USERNAME=<rabbit user>
RABBITMQ_PASSWORD=<rabbit password>
RABBITMQ_HOST=<rabbit host>
RABBITMQ_PORT=5672

CLEANING_QUEUE_NAME=q.staff.cleaning
CLEANING_BINDING_EXCHANGE_NAME=ex.cleaning.request

DAY_CHANGE_QUEUE_NAME=q.staff.day-change
DAY_CHANGE_BINDING_EXCHANGE_NAME=ex.time.event
DAY_CHANGE_BINDING_ROUTING_KEY=time.event.day

HOUR_CHANGE_QUEUE_NAME=q.staff.hour-change
HOUR_CHANGE_BINDING_EXCHANGE_NAME=ex.time.event
HOUR_CHANGE_BINDING_ROUTING_KEY=time.event.hour

OTEL_EXPORTER_OTLP_ENDPOINT=<otel host>
OTEL_EXPORTER_OTLP_GRPC_PORT=4317
OTEL_EXPORTER_OTLP_HEALTH_PORT=13133
OTEL_EXPORTER_OTLP_INSECURE=true
```

## Configuration

Configuration is environment-driven. Start with:

- `internal/config/config.go`
- `internal/config/rabbitmq_config.go`

Important groups:

- service HTTP: `SERVICE_*`
- employees: `CLEANING_EMPLOYEES_NAMES`, `LAUNDERING_EMPLOYEES_NAMES`, `STOCKERS_EMPLOYEES_NAMES`
- MongoDB: `MONGO_*`
- RabbitMQ: `RABBITMQ_*`, `CLEANING_*`, `DAY_CHANGE_*`, `HOUR_CHANGE_*`
- clock client: `CLOCK_EMU_GRPC_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Key Files

- `internal/main.go`
- `internal/app/manager_service.go`
- `internal/app/housekeeper_service.go`
- `internal/app/launderer_service.go`
- `internal/app/stocker_service.go`
