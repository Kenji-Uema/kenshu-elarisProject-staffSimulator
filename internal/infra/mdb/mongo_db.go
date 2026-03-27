package mdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

type Mdb struct {
	client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDb(ctx context.Context, config config.MongoConfig) (*Mdb, error) {
	startCtx, cancel := context.WithTimeout(ctx, time.Duration(config.Conn.StartupTimeoutInSeconds)*time.Second)
	defer cancel()

	uri := buildMongoURI(config)

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(otelmongo.NewMonitor(
			otelmongo.WithCommandAttributeDisabled(true),
		)).
		SetConnectTimeout(time.Duration(config.Conn.ConnectionTimeoutInSeconds) * time.Second).
		SetServerSelectionTimeout(time.Duration(config.Conn.ServerSelectionTimeoutInSeconds) * time.Second).
		SetMaxConnIdleTime(time.Duration(config.Conn.MaxConnIdleTimeInSeconds) * time.Second).
		SetMaxPoolSize(config.Conn.MaxPoolSize).
		SetMinPoolSize(config.Conn.MinPoolSize).
		SetRetryWrites(config.Conn.RetryWrites)
	if config.Conn.ReplicaSet != "" {
		clientOptions.SetReplicaSet(config.Conn.ReplicaSet)
	}

	client, err := mongo.Connect(clientOptions)

	if err != nil {
		return nil, err
	}

	databaseContext, databaseCancel := context.WithTimeout(startCtx, time.Duration(config.Conn.PingTimeoutInSeconds)*time.Second)
	defer databaseCancel()

	if err := client.Ping(databaseContext, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping failed for URI: %s, error: %w", uri, err)
	}

	return &Mdb{client: client, Database: client.Database(config.Conn.Database)}, nil
}

func buildMongoURI(config config.MongoConfig) string {
	host := config.Conn.Host
	if strings.HasPrefix(host, "mongodb://") || strings.HasPrefix(host, "mongodb+srv://") {
		return host
	}

	return fmt.Sprintf("mongodb://%s:%s@%s", string(config.Conn.Username), string(config.Conn.Password), host)
}

func (d *Mdb) Ping() error {
	return d.client.Ping(context.Background(), readpref.Primary())
}

func (d *Mdb) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}
