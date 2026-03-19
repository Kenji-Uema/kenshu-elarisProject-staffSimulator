package fakes

import (
	"context"
	"time"
)

type FakeClockClient struct {
	NowFn   func(ctx context.Context) (*time.Time, error)
	CloseFn func() error

	NowCallCount   int
	CloseCallCount int

	LastNowCtx context.Context
}

func (f *FakeClockClient) Now(ctx context.Context) (*time.Time, error) {
	f.NowCallCount++
	f.LastNowCtx = ctx

	if f.NowFn != nil {
		return f.NowFn(ctx)
	}

	now := time.Now()
	return &now, nil
}

func (f *FakeClockClient) Close() error {
	f.CloseCallCount++

	if f.CloseFn != nil {
		return f.CloseFn()
	}

	return nil
}
