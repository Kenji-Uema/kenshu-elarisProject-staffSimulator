package clock

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	conn   *grpc.ClientConn
	client ClockServiceClient
}

var _ port.Clock = (*Client)(nil)

func NewClockClient(cfg config.Services) (*Client, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.ClockSimulator.GrpcHost, cfg.ClockSimulator.GrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &Client{conn: conn, client: NewClockServiceClient(conn)}, nil
}

func (e *Client) Close() error {
	return e.conn.Close()
}

func (e *Client) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := e.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}
