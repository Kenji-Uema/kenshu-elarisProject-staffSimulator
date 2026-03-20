package config

import (
	"context"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Secret string

func (s Secret) String() string {
	return "REDACTED"
}

type Configs struct {
	AppConfig
	MongoConfig
	RabbitMqConfig
	Services
}

type AppConfig struct {
	Name struct {
		ServiceName string `env:"SERVICE_NAME"`
		Version     string `env:"VERSION"`
	}
	Server struct {
		Host string `env:"SERVICE_HOST,required"`
		Port int    `env:"SERVICE_PORT,required"`
	}
	Log struct {
		Level int `env:"LOG_LEVEL,required" envDefault:"0"`
	}
	Telemetry struct {
		OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
		OTLPGrpcPort   int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT"`
		OTLPHealthPort int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT"`
		OTLPInsecure   bool   `env:"OTEL_EXPORTER_OTLP_INSECURE"`
	}
	Employees struct {
		Housekeepers []string `env:"CLEANING_EMPLOYEES_NAMES,required"`
		Launderers   []string `env:"LAUNDERING_EMPLOYEES_NAMES,required"`
		Stockers     []string `env:"STOCKERS_EMPLOYEES_NAMES,required"`
	}
}

type MongoConfig struct {
	Conn struct {
		Username Secret `env:"MONGO_INITDB_ROOT_USERNAME,required"`
		Password Secret `env:"MONGO_INITDB_ROOT_PASSWORD,required"`
		Host     string `env:"MONGO_HOST,required"`
		Database string `env:"MONGO_DATABASE,required"`
	}
	Collections struct {
		Cottage string `env:"COTTAGE_COLLECTION,required" envDefault:"Cottage"`
		Stock   string `env:"STOCK_COLLECTION,required" envDefault:"Stock"`
	}
}

type RabbitMqConfig struct {
	Username  Secret `env:"RABBITMQ_USERNAME,required"`
	Password  Secret `env:"RABBITMQ_PASSWORD,required"`
	Host      string `env:"RABBITMQ_HOST,required"`
	Port      int    `env:"RABBITMQ_PORT,required"`
	Consumers struct {
		Cleaning struct {
			Queue   QueueConfig   `envPrefix:"CLEANING_QUEUE_"`
			Binding BindingConfig `envPrefix:"CLEANING_BINDING_"`
			Consume ConsumeConfig `envPrefix:"CLEANING_CONSUME_"`
		}
		DayChange struct {
			Queue   QueueConfig   `envPrefix:"DAY_CHANGE_QUEUE_"`
			Binding BindingConfig `envPrefix:"DAY_CHANGE_BINDING_"`
			Consume ConsumeConfig `envPrefix:"DAY_CHANGE_CONSUME_"`
		}
		HourChange struct {
			Queue   QueueConfig   `envPrefix:"HOUR_CHANGE_QUEUE_"`
			Binding BindingConfig `envPrefix:"HOUR_CHANGE_BINDING_"`
			Consume ConsumeConfig `envPrefix:"HOUR_CHANGE_CONSUME_"`
		}
	}
}

type Services struct {
	ClockSimulator struct {
		GrpcHost string `env:"CLOCK_EMU_GRPC_HOST,required"`
		GrpcPort int    `env:"CLOCK_EMU_GRPC_PORT,required"`
	}
}

func LoadConfigs() (Configs, error) {
	var cfg Configs
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	slog.InfoContext(context.Background(), "config loaded", "config", cfg)

	return cfg, nil
}
