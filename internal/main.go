package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Kenji-Uema/staffSimulator/internal/app"
	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mdb"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	grpcclock "github.com/Kenji-Uema/staffSimulator/internal/transport/grpc/clock"
)

const (
	cleaningQueueName  = "cleaning.requests"
	dayChangeQueueName = "day.change"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.LoadConfigs()
	if err != nil {
		return err
	}

	shutdownTelemetry, err := telemetry.Init(ctx, cfg.TelemetryConfig, cfg.AppConfig)
	if err != nil {
		return err
	}
	defer func() {
		if err := shutdownTelemetry(context.Background()); err != nil {
			slog.Error("failed to shutdown telemetry", "error", err)
		}
	}()

	mongoDB, err := mdb.NewMongoDb(ctx, cfg.MongoConfig)
	if err != nil {
		return err
	}
	defer func() {
		if err := mongoDB.Close(context.Background()); err != nil {
			slog.Error("failed to close mongo connection", "error", err)
		}
	}()

	clockClient, err := grpcclock.NewClockEmu(cfg.ClockEmuConfig)
	if err != nil {
		return err
	}
	defer func() {
		if err := clockClient.Close(); err != nil {
			slog.Error("failed to close clock client", "error", err)
		}
	}()

	rabbitConn, err := mq.NewRabbitMqConnection(ctx, cfg.RabbitMqConfig)
	if err != nil {
		return err
	}
	defer func() {
		if err := rabbitConn.Close(); err != nil {
			slog.Error("failed to close rabbitmq connection", "error", err)
		}
	}()

	cleaningConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, config.ConsumeConfig{})
	if err != nil {
		return err
	}
	defer closeConsumer(cleaningConsumer, "cleaning consumer")

	timeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, config.ConsumeConfig{})
	if err != nil {
		return err
	}
	defer closeConsumer(timeConsumer, "time consumer")

	laundererTimeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, config.ConsumeConfig{})
	if err != nil {
		return err
	}
	defer closeConsumer(laundererTimeConsumer, "launderer time consumer")

	if err := cleaningConsumer.DeclareQueue(ctx, config.QueueConfig{Name: cleaningQueueName}); err != nil {
		return err
	}
	if err := timeConsumer.DeclareQueue(ctx, config.QueueConfig{Name: dayChangeQueueName}); err != nil {
		return err
	}
	if err := laundererTimeConsumer.DeclareQueue(ctx, config.QueueConfig{Name: dayChangeQueueName}); err != nil {
		return err
	}

	stockRepo := mdb.NewStockRepo(mongoDB.Database)

	cleaningCh := make(chan domain.CleaningRequest, 32)
	laundererCh := make(chan domain.WashRequest, 32)
	stockerCh := make(chan domain.RestockRequest, 16)

	housekeeperService, err := app.NewHousekeeperService([]string{"housekeeper-1"}, *clockClient, stockRepo, laundererCh)
	if err != nil {
		return err
	}

	laundererService, err := app.NewLaundererService([]string{"launderer-1"}, *clockClient, laundererTimeConsumer, stockRepo)
	if err != nil {
		return err
	}

	stockerService, err := app.NewStockerService([]string{"stocker-1"}, stockRepo)
	if err != nil {
		return err
	}

	managerService := app.NewManagerService(cleaningConsumer, timeConsumer, cleaningCh, stockerCh)

	var wg sync.WaitGroup
	start := func(name string, fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.InfoContext(ctx, "starting worker", "name", name)
			fn()
		}()
	}

	start("housekeeper-service", func() { housekeeperService.Run(ctx, cleaningCh) })
	start("launderer-service", func() { laundererService.Run(ctx, laundererCh) })
	start("stocker-service", func() { stockerService.Run(ctx, stockerCh) })
	start("manager-service", func() { managerService.Start(ctx) })

	<-ctx.Done()
	slog.InfoContext(ctx, "shutdown signal received")
	wg.Wait()

	return nil
}

func closeConsumer(consumer interface{ CloseChannel() error }, name string) {
	if err := consumer.CloseChannel(); err != nil && !errors.Is(err, os.ErrClosed) {
		slog.Error("failed to close consumer channel", "consumer", name, "error", err)
	}
}
