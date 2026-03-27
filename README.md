# staffSimulator

Simulates operational staff reacting to cleaning and time-based work.

## Responsibilities

- simulate housekeeper, launderer, and stocker behavior
- consume cleaning and time-change events
- coordinate employee-oriented workflows
- expose HTTP health and readiness probes

## Interfaces

- RabbitMQ consumers for cleaning, day change, and hour change events
- gRPC client to `clockSimulator`
- MongoDB persistence
- HTTP probes

## Run

```sh
go run ./internal
```

## Build

```sh
make build
make docker-build
```

## Configuration

Configuration is environment-driven. See:

- `internal/config/config.go`
- `internal/config/rabbitmq_config.go`

Important families:

- service HTTP: `SERVICE_*`
- employees: `CLEANING_EMPLOYEES_NAMES`, `LAUNDERING_EMPLOYEES_NAMES`, `STOCKERS_EMPLOYEES_NAMES`
- MongoDB: `MONGO_*`
- RabbitMQ: `RABBITMQ_*`, `CLEANING_*`, `DAY_CHANGE_*`, `HOUR_CHANGE_*`
- clock client: `CLOCK_EMU_GRPC_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Entry points

- `internal/main.go`
- `internal/app/manager_service.go`
- `internal/app/time_event_notification_service.go`
