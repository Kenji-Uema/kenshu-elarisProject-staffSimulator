package port

import (
	"context"
	"time"
)

type Client interface {
	Now(ctx context.Context) (*time.Time, error)
	Close() error
}

type Clock = Client
