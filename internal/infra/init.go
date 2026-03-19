package infra

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mdb"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

type Mongo struct {
	Connection      *mdb.Mdb
	StockRepo       port.StockRepo
	CottageRepo     port.CottageRepo
	ConnectionClose func(context.Context) error
}

type Rabbitmq struct {
	Connection         *mq.RabbitMqConnection
	CleaningConsumer   port.MqConsumer
	DayChangeConsumer  port.MqConsumer
	HourChangeConsumer port.MqConsumer
	ConnectionClose    func() error
}

func NewMongoDb(ctx context.Context, err error, configs config.Configs) (Mongo, error) {
	mongoDB, err := mdb.NewMongoDb(ctx, configs.MongoConfig)
	if err != nil {
		return Mongo{}, err
	}

	stockRepo := mdb.NewStockRepo(mongoDB.Database)
	cottageRepo := mdb.NewCottageRepo(mongoDB.Database)
	return Mongo{
		Connection:      mongoDB,
		StockRepo:       stockRepo,
		CottageRepo:     cottageRepo,
		ConnectionClose: mongoDB.Close,
	}, nil
}

func NewRabbitmq(ctx context.Context, err error, configs config.Configs) (Rabbitmq, error) {
	rabbitConn, err := mq.NewRabbitMqConnection(ctx, configs.RabbitMqConfig)
	if err != nil {
		return Rabbitmq{}, err
	}

	cleaningConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.RabbitMqConfig.Consumers.Cleaning.Consume)
	if err != nil {
		return Rabbitmq{}, err
	}

	dayChangeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.RabbitMqConfig.Consumers.DayChange.Consume)
	if err != nil {
		return Rabbitmq{}, err
	}

	hourChangeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.RabbitMqConfig.Consumers.HourChange.Consume)
	if err != nil {
		return Rabbitmq{}, err
	}

	if err := declareAndBindConsumer(ctx, cleaningConsumer, configs.RabbitMqConfig.Consumers.Cleaning.Queue, configs.RabbitMqConfig.Consumers.Cleaning.Binding); err != nil {
		return Rabbitmq{}, err
	}
	if err := declareAndBindConsumer(ctx, dayChangeConsumer, configs.RabbitMqConfig.Consumers.DayChange.Queue, configs.RabbitMqConfig.Consumers.DayChange.Binding); err != nil {
		return Rabbitmq{}, err
	}
	if err := declareAndBindConsumer(ctx, hourChangeConsumer, configs.RabbitMqConfig.Consumers.HourChange.Queue, configs.RabbitMqConfig.Consumers.HourChange.Binding); err != nil {
		return Rabbitmq{}, err
	}
	return Rabbitmq{
		Connection:         rabbitConn,
		CleaningConsumer:   cleaningConsumer,
		DayChangeConsumer:  dayChangeConsumer,
		HourChangeConsumer: hourChangeConsumer,
		ConnectionClose:    rabbitConn.Close,
	}, nil
}

func declareAndBindConsumer(ctx context.Context, consumer port.MqConsumer, queueCfg config.QueueConfig, bindingCfg config.BindingConfig) error {
	if err := consumer.DeclareQueue(ctx, queueCfg); err != nil {
		return err
	}

	return consumer.BindQueue(ctx, bindingCfg)
}
