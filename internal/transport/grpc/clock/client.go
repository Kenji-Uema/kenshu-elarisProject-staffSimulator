package clock

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Clock struct {
	conn   *grpc.ClientConn
	client ClockServiceClient
}

func NewClockEmu(cfg config.ClockEmuConfig) (*Clock, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.GrpcHost, cfg.GrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	if err != nil {
		return nil, err
	}

	return &Clock{conn: conn, client: NewClockServiceClient(conn)}, nil
}

func (e *Clock) Close() error {
	return e.conn.Close()
}

func (e *Clock) Now(ctx context.Context) (*time.Time, error) {
	createTime, err := e.client.Now(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	createdTimestamp := createTime.Time.AsTime()

	return &createdTimestamp, nil
}
