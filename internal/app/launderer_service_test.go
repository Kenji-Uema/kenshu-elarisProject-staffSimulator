package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/fakes"
)

func TestNewLaundererService(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no employees are configured", func(t *testing.T) {
		service, err := NewLaundererService(nil, &fakes.FakeClockClient{}, &fakeLaundererTimeEventRegistry{}, &fakes.FakeStockRepo{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when hour change notifier is nil", func(t *testing.T) {
		var notifier *fakeLaundererTimeEventRegistry

		service, err := NewLaundererService([]string{"Alice"}, &fakes.FakeClockClient{}, notifier, &fakes.FakeStockRepo{}, nil)
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
			&fakeLaundererTimeEventRegistry{},
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
		validationFn := func(t *testing.T, notifier *fakeLaundererTimeEventRegistry, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if notifier.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", notifier.RegisterCallCount)
			}
			if notifier.LastRegisteredEventType != timeEventHourChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", notifier.LastRegisteredEventType, timeEventHourChange)
			}
			if notifier.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", notifier.UnregisterCallCount)
			}
			if notifier.LastUnregisteredEventType != timeEventHourChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", notifier.LastUnregisteredEventType, timeEventHourChange)
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
		validationFn := func(t *testing.T, notifier *fakeLaundererTimeEventRegistry, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if notifier.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", notifier.RegisterCallCount)
			}
			if notifier.LastRegisteredEventType != timeEventHourChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", notifier.LastRegisteredEventType, timeEventHourChange)
			}
			if notifier.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", notifier.UnregisterCallCount)
			}
			if notifier.LastUnregisteredEventType != timeEventHourChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", notifier.LastUnregisteredEventType, timeEventHourChange)
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
		validationFn := func(t *testing.T, notifier *fakeLaundererTimeEventRegistry, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if notifier.RegisterCallCount != 0 {
				t.Fatalf("RegisterCallCount = %d, want 0", notifier.RegisterCallCount)
			}
			if notifier.UnregisterCallCount != 0 {
				t.Fatalf("UnregisterCallCount = %d, want 0", notifier.UnregisterCallCount)
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

		validationFn := func(t *testing.T, notifier *fakeLaundererTimeEventRegistry, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if notifier.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", notifier.RegisterCallCount)
			}
			if notifier.LastRegisteredEventType != timeEventHourChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", notifier.LastRegisteredEventType, timeEventHourChange)
			}
			if notifier.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", notifier.UnregisterCallCount)
			}
			if notifier.LastUnregisteredEventType != timeEventHourChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", notifier.LastUnregisteredEventType, timeEventHourChange)
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

		validationFn := func(t *testing.T, notifier *fakeLaundererTimeEventRegistry, stockRepo *fakes.FakeStockRepo) {
			t.Helper()

			if notifier.RegisterCallCount != 0 {
				t.Fatalf("RegisterCallCount = %d, want 0", notifier.RegisterCallCount)
			}
			if notifier.UnregisterCallCount != 0 {
				t.Fatalf("UnregisterCallCount = %d, want 0", notifier.UnregisterCallCount)
			}
			if stockRepo.RestockItemCallCount != 0 {
				t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
			}
			if stockRepo.ConsumeItemCallCount != 1 {
				t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
			}
		}

		laundererRun(t, domain.WashRequest{RoomName: "A", Item: "linens"}, stockRepo, validationFn)
	})
}

func TestLaundererDoesNotRegisterWithoutWashRequests(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	clockClient := &fakes.FakeClockClient{
		NowFn: func(ctx context.Context) (*time.Time, error) {
			return &startTime, nil
		},
	}

	notifier := &fakeLaundererTimeEventRegistry{}
	service, err := NewLaundererService(
		[]string{"Alice"},
		clockClient,
		notifier,
		&fakes.FakeStockRepo{},
		nil,
	)
	if err != nil {
		t.Fatalf("NewLaundererService() error = %v", err)
	}

	requests := make(chan domain.WashRequest)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Work(ctx, requests)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	if notifier.RegisterCallCount != 0 {
		t.Fatalf("RegisterCallCount = %d, want 0", notifier.RegisterCallCount)
	}
	if notifier.UnregisterCallCount != 0 {
		t.Fatalf("UnregisterCallCount = %d, want 0", notifier.UnregisterCallCount)
	}

	cancel()
	<-done
}

func laundererRun(t *testing.T, request domain.WashRequest, stockRepo *fakes.FakeStockRepo,
	validate func(t *testing.T, notifier *fakeLaundererTimeEventRegistry, stockRepo *fakes.FakeStockRepo)) {
	t.Helper()

	if stockRepo == nil {
		stockRepo = &fakes.FakeStockRepo{}
	}

	startTime := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	finishTime := startTime
	if config, ok := washTable[request.Item]; ok {
		finishTime = startTime.Add(time.Duration(config.cycleDurationInHours * float64(time.Hour)))
	}

	clockClient := &fakes.FakeClockClient{
		NowFn: func(ctx context.Context) (*time.Time, error) {
			return &startTime, nil
		},
	}

	notifier := &fakeLaundererTimeEventRegistry{
		RegisterFn: func(eventType timeEventType, ch chan<- time.Time) {
			if eventType != timeEventHourChange {
				t.Fatalf("Register() eventType = %q, want %q", eventType, timeEventHourChange)
			}
			ch <- finishTime
		},
	}

	service, err := NewLaundererService(
		[]string{"Alice"},
		clockClient,
		notifier,
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
		validate(t, notifier, stockRepo)
	case <-time.After(time.Second):
		t.Fatal("Work did not return after cancellation")
	}
}

type fakeLaundererTimeEventRegistry struct {
	RegisterFn   func(timeEventType, chan<- time.Time)
	UnregisterFn func(timeEventType, chan<- time.Time)

	RegisterCallCount         int
	UnregisterCallCount       int
	LastRegisteredEventType   timeEventType
	LastUnregisteredEventType timeEventType
	LastRegisteredCh          chan<- time.Time
	LastUnregisteredCh        chan<- time.Time
}

func (f *fakeLaundererTimeEventRegistry) Register(eventType timeEventType, ch chan<- time.Time) {
	f.RegisterCallCount++
	f.LastRegisteredEventType = eventType
	f.LastRegisteredCh = ch

	if f.RegisterFn != nil {
		f.RegisterFn(eventType, ch)
	}
}

func (f *fakeLaundererTimeEventRegistry) Unregister(eventType timeEventType, ch chan<- time.Time) {
	f.UnregisterCallCount++
	f.LastUnregisteredEventType = eventType
	f.LastUnregisteredCh = ch

	if f.UnregisterFn != nil {
		f.UnregisterFn(eventType, ch)
	}
}
