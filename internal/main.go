package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/app"
	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/infra"
	grpcclock "github.com/Kenji-Uema/staffSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/logging"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"github.com/Kenji-Uema/staffSimulator/internal/transport"
)

const shutdownTimeout = 5 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "staff simulator failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configs, err := config.LoadConfigs()
	if err != nil {
		return err
	}

	slog.SetDefault(logging.NewLogger(configs.AppConfig))
	slog.InfoContext(ctx, "staff simulator starting")

	shutdownTelemetry, err := telemetry.Init(ctx, configs.AppConfig)
	if err != nil {
		return err
	}

	mongoDB, err := infra.NewMongoDb(ctx, err, configs)
	if err != nil {
		return err
	}

	rabbitmq, err := infra.NewRabbitmq(ctx, err, configs)
	if err != nil {
		return err
	}

	clockClient, err := grpcclock.NewClockClient(configs.Services)
	if err != nil {
		return err
	}

	httpServer := transport.StartHTTPServer(configs.AppConfig, rabbitmq.Connection, mongoDB.Connection)

	services, err := app.NewServices(configs.AppConfig, mongoDB, rabbitmq, clockClient)
	if err != nil {
		return err
	}
	services.Start(ctx)

	<-ctx.Done()
	slog.InfoContext(ctx, "shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdown(shutdownCtx, shutdownTelemetry, mongoDB, rabbitmq, clockClient, httpServer)

	return nil
}

func shutdown(ctx context.Context, shutdownTelemetry func(context.Context) error, mongoDB infra.Mongo, rabbitmq infra.Rabbitmq, clockClient port.Clock, httpServer *http.Server) {
	if err := shutdownTelemetry(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to shutdown telemetry", "error", err)
	}

	if err := mongoDB.ConnectionClose(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to close mongo connection", "error", err)
	}

	if err := rabbitmq.HourChangeConsumer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "failed to close hour change consumer channel", "error", err)
	}

	if err := rabbitmq.DayChangeConsumer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "failed to close day change consumer channel", "error", err)
	}

	if err := rabbitmq.CleaningConsumer.CloseChannel(); err != nil {
		slog.ErrorContext(ctx, "failed to close cleaning consumer channel", "error", err)
	}

	if err := rabbitmq.ConnectionClose(); err != nil {
		slog.ErrorContext(ctx, "failed to close rabbitmq connection", "error", err)
	}

	if err := clockClient.Close(); err != nil {
		slog.ErrorContext(ctx, "failed to close clock client", "error", err)
	}

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to close http server", "error", err)
	}
}
