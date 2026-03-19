package clock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeClockServiceClient struct {
	nowCallCount int
	nowFn        func(ctx context.Context, in *emptypb.Empty) (*dto.TimeEvent, error)
}

func (f *fakeClockServiceClient) Now(ctx context.Context, in *emptypb.Empty, _ ...grpc.CallOption) (*dto.TimeEvent, error) {
	f.nowCallCount++
	return f.nowFn(ctx, in)
}

func TestClientNow(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		want := time.Date(2026, 3, 19, 12, 30, 0, 0, time.UTC)
		fakeClient := &fakeClockServiceClient{
			nowFn: func(ctx context.Context, in *emptypb.Empty) (*dto.TimeEvent, error) {
				return &dto.TimeEvent{Time: timestamppb.New(want)}, nil
			},
		}

		client := &Client{client: fakeClient}

		got, err := client.Now(context.Background())
		if err != nil {
			t.Fatalf("Now() error = %v", err)
		}
		if got == nil {
			t.Fatal("expected time, got nil")
		}
		if !got.Equal(want) {
			t.Fatalf("Now() = %v, want %v", got, want)
		}
		if fakeClient.nowCallCount != 1 {
			t.Fatalf("nowCallCount = %d, want 1", fakeClient.nowCallCount)
		}
	})

	t.Run("error case", func(t *testing.T) {
		wantErr := errors.New("clock unavailable")
		fakeClient := &fakeClockServiceClient{
			nowFn: func(ctx context.Context, in *emptypb.Empty) (*dto.TimeEvent, error) {
				return nil, wantErr
			},
		}

		client := &Client{client: fakeClient}

		got, err := client.Now(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, wantErr) {
			t.Fatalf("Now() error = %v, want %v", err, wantErr)
		}
		if got != nil {
			t.Fatalf("Now() = %v, want nil", got)
		}
		if fakeClient.nowCallCount != 1 {
			t.Fatalf("nowCallCount = %d, want 1", fakeClient.nowCallCount)
		}
	})
}
