package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/fakes"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewLaundererService(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no employees are configured", func(t *testing.T) {
		service, err := NewLaundererService(nil, &fakes.FakeClockClient{}, &fakes.FakeMqConsumer{}, &fakes.FakeStockRepo{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when hour change consumer is nil", func(t *testing.T) {
		var consumer *fakes.FakeMqConsumer

		service, err := NewLaundererService([]string{"Alice"}, &fakes.FakeClockClient{}, consumer, &fakes.FakeStockRepo{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("creates service with configured dependencies", func(t *testing.T) {
		service, err := NewLaundererService(
			[]string{"Alice"},
			&fakes.FakeClockClient{},
			&fakes.FakeMqConsumer{},
			&fakes.FakeStockRepo{},
			nil,
		)
		if err != nil {
			t.Fatalf("NewLaundererService() error = %v", err)
		}

		if service.employeeCount != 1 {
			t.Fatalf("employeeCount = %d, want 1", service.employeeCount)
		}

		if service.work == nil {
			t.Fatal("expected launderer work handler to be configured")
		}
	})
}

func TestLaundererServiceRun(t *testing.T) {
	t.Parallel()

	t.Run("washes towels", func(t *testing.T) {
		validationFn := func(t *testing.T, consumer *fakes.FakeMqConsumer, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if consumer.ConsumeCallCount != 1 {
				t.Fatalf("ConsumeCallCount = %d, want 1", consumer.ConsumeCallCount)
			}
			if stockRepo.ConsumeItemCallCount != 1 {
				t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.LastConsumeItemName != "soap" {
				t.Fatalf("LastConsumeItemName = %q, want %q", stockRepo.LastConsumeItemName, "soap")
			}
			if stockRepo.LastConsumeQuantity != 10 {
				t.Fatalf("LastConsumeQuantity = %d, want 10", stockRepo.LastConsumeQuantity)
			}
			if stockRepo.RestockItemCallCount != 0 {
				t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
			}
		}

		laundererRun(t, domain.WashRequest{RoomName: "A", Item: "towels"}, nil, validationFn)
	})

	t.Run("washes linens", func(t *testing.T) {
		validationFn := func(t *testing.T, consumer *fakes.FakeMqConsumer, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if consumer.ConsumeCallCount != 1 {
				t.Fatalf("ConsumeCallCount = %d, want 1", consumer.ConsumeCallCount)
			}
			if stockRepo.ConsumeItemCallCount != 1 {
				t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.LastConsumeItemName != "soap" {
				t.Fatalf("LastConsumeItemName = %q, want %q", stockRepo.LastConsumeItemName, "soap")
			}
			if stockRepo.LastConsumeQuantity != 20 {
				t.Fatalf("LastConsumeQuantity = %d, want 20", stockRepo.LastConsumeQuantity)
			}
			if stockRepo.RestockItemCallCount != 0 {
				t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
			}
		}

		laundererRun(t, domain.WashRequest{RoomName: "A", Item: "linens"}, nil, validationFn)
	})

	t.Run("does not consume stock for unsupported item", func(t *testing.T) {
		validationFn := func(t *testing.T, consumer *fakes.FakeMqConsumer, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if consumer.ConsumeCallCount != 0 {
				t.Fatalf("ConsumeCallCount = %d, want 0", consumer.ConsumeCallCount)
			}
			if stockRepo.ConsumeItemCallCount != 0 {
				t.Fatalf("ConsumeItemCallCount = %d, want 0", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.RestockItemCallCount != 0 {
				t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
			}
		}

		laundererRun(t, domain.WashRequest{RoomName: "A", Item: "unknown"}, nil, validationFn)
	})
}

func TestLaundererWorkRetry(t *testing.T) {
	t.Parallel()

	t.Run("restocks and retries when stock is insufficient", func(t *testing.T) {
		consumeCalls := 0
		stockRepo := &fakes.FakeStockRepo{
			ConsumeItemFn: func(ctx context.Context, itemName string, quantity int) error {
				consumeCalls++
				if consumeCalls == 1 {
					return &dbErrors.StockInsufficientQuantityErr{ItemName: itemName, Quantity: quantity}
				}
				return nil
			},
		}

		validationFn := func(t *testing.T, consumer *fakes.FakeMqConsumer, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if consumer.ConsumeCallCount != 1 {
				t.Fatalf("ConsumeCallCount = %d, want 1", consumer.ConsumeCallCount)
			}
			if consumeCalls != 2 {
				t.Fatalf("consumeCalls = %d, want 2", consumeCalls)
			}
			if stockRepo.RestockItemCallCount != 1 {
				t.Fatalf("RestockItemCallCount = %d, want 1", stockRepo.RestockItemCallCount)
			}
			if stockRepo.LastRestockItemName != "soap" {
				t.Fatalf("LastRestockItemName = %q, want %q", stockRepo.LastRestockItemName, "soap")
			}
			if stockRepo.LastRestockQuantity != stockReplenishmentQuantity {
				t.Fatalf("LastRestockQuantity = %d, want %d", stockRepo.LastRestockQuantity, stockReplenishmentQuantity)
			}
		}

		laundererRun(t, domain.WashRequest{RoomName: "A", Item: "linens"}, stockRepo, validationFn)
	})

	t.Run("does not restock for non-stock errors", func(t *testing.T) {
		stockRepo := &fakes.FakeStockRepo{
			ConsumeItemFn: func(ctx context.Context, itemName string, quantity int) error {
				return errors.New("db down")
			},
		}

		validationFn := func(t *testing.T, consumer *fakes.FakeMqConsumer, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if stockRepo.RestockItemCallCount != 0 {
				t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
			}
			if stockRepo.ConsumeItemCallCount != 1 {
				t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
			}
			if consumer.ConsumeCallCount != 0 {
				t.Fatalf("ConsumeCallCount = %d, want 0", consumer.ConsumeCallCount)
			}
		}

		laundererRun(t, domain.WashRequest{RoomName: "A", Item: "linens"}, stockRepo, validationFn)
	})
}

func laundererRun(t *testing.T, request domain.WashRequest, stockRepo *fakes.FakeStockRepo,
	validate func(t *testing.T, consumer *fakes.FakeMqConsumer, stockRepo *fakes.FakeStockRepo)) {
	t.Helper()

	if stockRepo == nil {
		stockRepo = &fakes.FakeStockRepo{}
	}

	startTime := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)

	clockClient := &fakes.FakeClockClient{
		NowFn: func(ctx context.Context) (*time.Time, error) {
			return &startTime, nil
		},
	}

	consumer := &fakes.FakeMqConsumer{
		ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
			ch := make(chan amqp.Delivery, 1)

			if config, ok := washTable[request.Item]; ok {
				finishTime := startTime.Add(time.Duration(config.cycleDurationInHours * float64(time.Hour)))
				ch <- amqp.Delivery{Body: mustMarshalTimeEvent(t, finishTime)}
			}

			close(ch)
			return ch, nil
		},
	}

	service, err := NewLaundererService(
		[]string{"Alice"},
		clockClient,
		consumer,
		stockRepo,
		&stocker{stockRepo: stockRepo},
	)
	if err != nil {
		t.Fatalf("NewLaundererService() error = %v", err)
	}

	requests := make(chan domain.WashRequest, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		service.Work(ctx, requests)
		close(done)
	}()

	requests <- request

	select {
	case <-done:
		t.Fatal("Work returned before cancellation")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	select {
	case <-done:
		validate(t, consumer, stockRepo)
	case <-time.After(time.Second):
		t.Fatal("Work did not return after cancellation")
	}
}

func mustMarshalTimeEvent(t *testing.T, eventTime time.Time) []byte {
	t.Helper()

	payload, err := proto.Marshal(&dto.TimeEvent{
		Time: timestamppb.New(eventTime),
	})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return payload
}
