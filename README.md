# staffSimulator

Simulates operational staff work triggered by cleaning requests and time-change events.

## What It Does

- consumes cleaning requests from RabbitMQ and dispatches work by request type
- simulates housekeepers, launderers, and stockers
- updates cottage cleaning state and shared stock in MongoDB
- reacts to day-change and hour-change events
- exposes `/healthz` and `/readyz` probes

## Interfaces

- RabbitMQ consumers for cleaning, day-change, and hour-change events
- MongoDB repositories for cottage and stock state
- gRPC client to `clockSimulator`
- HTTP probes

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
