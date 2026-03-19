package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kenji-Uema/staffSimulator/internal/app"
	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	grpcclock "github.com/Kenji-Uema/staffSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/logging"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mdb"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

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
	telemetryInitialized := true
	defer func() {
		if telemetryInitialized {
			if shutdownErr := shutdownTelemetry(context.Background()); shutdownErr != nil {
				slog.Error("failed to shutdown telemetry", "error", shutdownErr)
			}
		}
	}()

	mongoDB, err := initMongoDb(ctx, err, configs)
	if err != nil {
		return err
	}
	mongoInitialized := true
	defer func() {
		if mongoInitialized {
			if closeErr := mongoDB.connectionClose(context.Background()); closeErr != nil {
				slog.Error("failed to close mongo connection", "error", closeErr)
			}
		}
	}()

	rabbitmq, err := initRabbitmq(ctx, err, configs)
	if err != nil {
		return err
	}
	rabbitInitialized := true
	defer func() {
		if rabbitInitialized {
			closeConsumer(rabbitmq.hourChangeConsumer, "hour change consumer")
			closeConsumer(rabbitmq.dayChangeConsumer, "day change consumer")
			closeConsumer(rabbitmq.cleaningConsumer, "cleaning consumer")

			if closeErr := rabbitmq.connectionClose(); closeErr != nil {
				slog.Error("failed to close rabbitmq connection", "error", closeErr)
			}
		}
	}()

	clockClient, err := grpcclock.NewClockClient(configs.Services)
	if err != nil {
		return err
	}
	clockInitialized := true
	defer func() {
		if clockInitialized {
			if closeErr := clockClient.Close(); closeErr != nil {
				slog.Error("failed to close clock client", "error", closeErr)
			}
		}
	}()

	channels := initChannels()

	services, err := initServices(configs.AppConfig, mongoDB, rabbitmq, channels, clockClient)
	if err != nil {
		return err
	}

	go services.housekeeperService.Work(ctx, channels.cleaning)
	go services.laundererService.Work(ctx, channels.launderer)
	go services.stockerService.Work(ctx, channels.stocker)
	services.managerService.Start(ctx)

	<-ctx.Done()
	slog.InfoContext(ctx, "shutdown signal received")

	clockInitialized = false
	rabbitInitialized = false
	mongoInitialized = false
	telemetryInitialized = false
	shutdown(shutdownTelemetry, mongoDB, rabbitmq, clockClient)

	return nil
}

func shutdown(shutdownTelemetry func(context.Context) error, mongoDB mongo, rabbitmq rabbitmq, clockClient port.Clock) {
	if err := shutdownTelemetry(context.Background()); err != nil {
		slog.Error("failed to shutdown telemetry", "error", err)
	}

	if err := mongoDB.connectionClose(context.Background()); err != nil {
		slog.Error("failed to close mongo connection", "error", err)
	}

	closeConsumer(rabbitmq.hourChangeConsumer, "hour change consumer")
	closeConsumer(rabbitmq.dayChangeConsumer, "day change consumer")
	closeConsumer(rabbitmq.cleaningConsumer, "cleaning consumer")

	if err := rabbitmq.connectionClose(); err != nil {
		slog.Error("failed to close rabbitmq connection", "error", err)
	}

	if err := clockClient.Close(); err != nil {
		slog.Error("failed to close clock client", "error", err)
	}
}

type channels struct {
	cleaning  chan domain.CleaningRequest
	launderer chan domain.WashRequest
	stocker   chan domain.RestockRequest
}

func initChannels() channels {
	return channels{
		cleaning:  make(chan domain.CleaningRequest, 32),
		launderer: make(chan domain.WashRequest, 32),
		stocker:   make(chan domain.RestockRequest, 16),
	}
}

type services struct {
	housekeeperService *app.HousekeeperService
	laundererService   *app.LaundererService
	stockerService     *app.StockerService
	managerService     *app.ManagerService
}

func initServices(configs config.AppConfig, mongo mongo, rabbitmq rabbitmq, channels channels, clock port.Clock) (services, error) {
	stockerService, err := app.NewStockerService(configs.Employees.Stockers, mongo.stockRepo)
	if err != nil {
		return services{}, err
	}

	housekeeperService, err := app.NewHousekeeperService(
		configs.Employees.Housekeepers,
		clock,
		mongo.cottageRepo,
		mongo.stockRepo,
		stockerService,
		channels.launderer,
	)
	if err != nil {
		return services{}, err
	}

	laundererService, err := app.NewLaundererService(
		configs.Employees.Launderers,
		clock,
		rabbitmq.hourChangeConsumer,
		mongo.stockRepo,
		stockerService,
	)
	if err != nil {
		return services{}, err
	}

	managerService, err := app.NewManagerService(rabbitmq.cleaningConsumer, rabbitmq.dayChangeConsumer, channels.cleaning, channels.stocker)
	if err != nil {
		return services{}, err
	}
	return services{
		housekeeperService: housekeeperService,
		laundererService:   laundererService,
		stockerService:     stockerService,
		managerService:     managerService,
	}, nil
}

type rabbitmq struct {
	cleaningConsumer   port.MqConsumer
	dayChangeConsumer  port.MqConsumer
	hourChangeConsumer port.MqConsumer
	connectionClose    func() error
}

func initRabbitmq(ctx context.Context, err error, configs config.Configs) (rabbitmq, error) {
	rabbitConn, err := mq.NewRabbitMqConnection(ctx, configs.RabbitMqConfig)
	if err != nil {
		return rabbitmq{}, err
	}

	cleaningConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.RabbitMqConfig.Consumers.Cleaning.Consume)
	if err != nil {
		return rabbitmq{}, err
	}

	dayChangeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.RabbitMqConfig.Consumers.DayChange.Consume)
	if err != nil {
		return rabbitmq{}, err
	}

	hourChangeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.RabbitMqConfig.Consumers.HourChange.Consume)
	if err != nil {
		return rabbitmq{}, err
	}

	if err := cleaningConsumer.DeclareQueue(ctx, configs.RabbitMqConfig.Consumers.Cleaning.Queue); err != nil {
		return rabbitmq{}, err
	}
	if err := cleaningConsumer.BindQueue(ctx, configs.RabbitMqConfig.Consumers.Cleaning.Binding); err != nil {
		return rabbitmq{}, err
	}
	if err := dayChangeConsumer.DeclareQueue(ctx, configs.RabbitMqConfig.Consumers.DayChange.Queue); err != nil {
		return rabbitmq{}, err
	}
	if err := dayChangeConsumer.BindQueue(ctx, configs.RabbitMqConfig.Consumers.DayChange.Binding); err != nil {
		return rabbitmq{}, err
	}
	if err := hourChangeConsumer.DeclareQueue(ctx, configs.RabbitMqConfig.Consumers.HourChange.Queue); err != nil {
		return rabbitmq{}, err
	}
	if err := hourChangeConsumer.BindQueue(ctx, configs.RabbitMqConfig.Consumers.HourChange.Binding); err != nil {
		return rabbitmq{}, err
	}
	return rabbitmq{
		cleaningConsumer:   cleaningConsumer,
		dayChangeConsumer:  dayChangeConsumer,
		hourChangeConsumer: hourChangeConsumer,
		connectionClose:    rabbitConn.Close,
	}, nil
}

type mongo struct {
	stockRepo       port.StockRepo
	cottageRepo     port.CottageRepo
	connectionClose func(context.Context) error
}

func initMongoDb(ctx context.Context, err error, configs config.Configs) (mongo, error) {
	mongoDB, err := mdb.NewMongoDb(ctx, configs.MongoConfig)
	if err != nil {
		return mongo{}, err
	}

	stockRepo := mdb.NewStockRepo(mongoDB.Database)
	cottageRepo := mdb.NewCottageRepo(mongoDB.Database)
	return mongo{
		stockRepo:       stockRepo,
		cottageRepo:     cottageRepo,
		connectionClose: mongoDB.Close,
	}, nil
}

func closeConsumer(consumer interface{ CloseChannel() error }, name string) {
	if err := consumer.CloseChannel(); err != nil && !errors.Is(err, os.ErrClosed) {
		slog.Error("failed to close consumer channel", "consumer", name, "error", err)
	}
}
