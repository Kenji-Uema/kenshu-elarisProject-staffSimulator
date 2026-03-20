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

func TestNewHousekeeperService(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no employees are configured", func(t *testing.T) {
		service, err := NewHousekeeperService(nil, &fakes.FakeClockClient{}, &fakes.FakeCottageRepo{}, &fakes.FakeStockRepo{}, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when cottage repo is nil", func(t *testing.T) {
		var cottageRepo *fakes.FakeCottageRepo

		service, err := NewHousekeeperService([]string{"Alice"}, &fakes.FakeClockClient{}, cottageRepo, &fakes.FakeStockRepo{}, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when stock repo is nil", func(t *testing.T) {
		var stockRepo *fakes.FakeStockRepo

		service, err := NewHousekeeperService([]string{"Alice"}, &fakes.FakeClockClient{}, &fakes.FakeCottageRepo{}, stockRepo, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("creates service with configured employees", func(t *testing.T) {
		service, err := NewHousekeeperService(
			[]string{"Alice", "Bob"},
			&fakes.FakeClockClient{},
			&fakes.FakeCottageRepo{},
			&fakes.FakeStockRepo{},
			nil,
			nil,
		)
		if err != nil {
			t.Fatalf("NewHousekeeperService() error = %v", err)
		}

		if service.employeeCount != 2 {
			t.Fatalf("employeeCount = %d, want 2", service.employeeCount)
		}

		if service.work == nil {
			t.Fatal("expected housekeeper work handler to be configured")
		}
	})
}

func TestHousekeeperServiceRun(t *testing.T) {
	t.Parallel()

	t.Run("prepare room for new guest", func(t *testing.T) {
		validationFn := func(t *testing.T, cottageRepo *fakes.FakeCottageRepo, stockRepo *fakes.FakeStockRepo, laundryRequests <-chan domain.WashRequest) {
			t.Helper()

			if cottageRepo.UpdateCleaningStatusCallCount != 1 {
				t.Fatalf("UpdateCleaningStatusCallCount = %d, want 1", cottageRepo.UpdateCleaningStatusCallCount)
			}
			if cottageRepo.LastUpdateCleaningStatusRoom != "A" {
				t.Fatalf("LastUpdateCleaningStatusRoom = %q, want %q", cottageRepo.LastUpdateCleaningStatusRoom, "A")
			}
			if cottageRepo.LastCleaningStatus != "PREPARED_FOR_GUEST" {
				t.Fatalf("LastCleaningStatus = %q, want %q", cottageRepo.LastCleaningStatus, "PREPARED_FOR_GUEST")
			}
			if stockRepo.ConsumeItemCallCount != 2 {
				t.Fatalf("ConsumeItemCallCount = %d, want 2", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.LastConsumeItemName != "aromaCandle" {
				t.Fatalf("LastConsumeItemName = %q, want %q", stockRepo.LastConsumeItemName, "aromaCandle")
			}
			if stockRepo.LastConsumeQuantity != 1 {
				t.Fatalf("LastConsumeQuantity = %d, want 1", stockRepo.LastConsumeQuantity)
			}

			select {
			case req := <-laundryRequests:
				t.Fatalf("unexpected laundry request: %+v", req)
			default:
			}
		}

		housekeeperRun(t, "PREPARE_FOR_GUEST", validationFn)
	})

	t.Run("daily cleaning", func(t *testing.T) {
		validationFn := func(t *testing.T, cottageRepo *fakes.FakeCottageRepo, stockRepo *fakes.FakeStockRepo, laundryRequests <-chan domain.WashRequest) {
			t.Helper()

			if cottageRepo.UpdateCleaningStatusCallCount != 1 {
				t.Fatalf("UpdateCleaningStatusCallCount = %d, want 1", cottageRepo.UpdateCleaningStatusCallCount)
			}
			if cottageRepo.LastUpdateCleaningStatusRoom != "A" {
				t.Fatalf("LastUpdateCleaningStatusRoom = %q, want %q", cottageRepo.LastUpdateCleaningStatusRoom, "A")
			}
			if cottageRepo.LastCleaningStatus != "DAILY_CLEANED" {
				t.Fatalf("LastCleaningStatus = %q, want %q", cottageRepo.LastCleaningStatus, "DAILY_CLEANED")
			}
			if stockRepo.ConsumeItemCallCount != 1 {
				t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.LastConsumeItemName != "cleaningItem" {
				t.Fatalf("LastConsumeItemName = %q, want %q", stockRepo.LastConsumeItemName, "cleaningItem")
			}
			if stockRepo.LastConsumeQuantity != 5 {
				t.Fatalf("LastConsumeQuantity = %d, want 5", stockRepo.LastConsumeQuantity)
			}

			select {
			case got := <-laundryRequests:
				want := domain.WashRequest{RoomName: "A", Item: "towels"}
				if got != want {
					t.Fatalf("laundry request = %+v, want %+v", got, want)
				}
			default:
				t.Fatal("expected laundry request")
			}
		}

		housekeeperRun(t, "DAILY_CLEANING", validationFn)
	})

	t.Run("full cleaning", func(t *testing.T) {
		validationFn := func(t *testing.T, cottageRepo *fakes.FakeCottageRepo, stockRepo *fakes.FakeStockRepo, laundryRequests <-chan domain.WashRequest) {
			t.Helper()

			if cottageRepo.UpdateCleaningStatusCallCount != 1 {
				t.Fatalf("UpdateCleaningStatusCallCount = %d, want 1", cottageRepo.UpdateCleaningStatusCallCount)
			}
			if cottageRepo.LastUpdateCleaningStatusRoom != "A" {
				t.Fatalf("LastUpdateCleaningStatusRoom = %q, want %q", cottageRepo.LastUpdateCleaningStatusRoom, "A")
			}
			if cottageRepo.LastCleaningStatus != "FULLY_CLEANED" {
				t.Fatalf("LastCleaningStatus = %q, want %q", cottageRepo.LastCleaningStatus, "FULLY_CLEANED")
			}
			if stockRepo.ConsumeItemCallCount != 1 {
				t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.LastConsumeItemName != "cleaningItem" {
				t.Fatalf("LastConsumeItemName = %q, want %q", stockRepo.LastConsumeItemName, "cleaningItem")
			}
			if stockRepo.LastConsumeQuantity != 10 {
				t.Fatalf("LastConsumeQuantity = %d, want 10", stockRepo.LastConsumeQuantity)
			}

			wantRequests := []domain.WashRequest{
				{RoomName: "A", Item: "towels"},
				{RoomName: "A", Item: "linens"},
			}
			for _, want := range wantRequests {
				select {
				case got := <-laundryRequests:
					if got != want {
						t.Fatalf("laundry request = %+v, want %+v", got, want)
					}
				default:
					t.Fatalf("expected laundry request: %+v", want)
				}
			}
		}

		housekeeperRun(t, "FULL_CLEANING", validationFn)
	})

	t.Run("prepare room for sleep", func(t *testing.T) {
		validationFn := func(t *testing.T, cottageRepo *fakes.FakeCottageRepo, stockRepo *fakes.FakeStockRepo, laundryRequests <-chan domain.WashRequest) {
			t.Helper()

			if cottageRepo.UpdateCleaningStatusCallCount != 1 {
				t.Fatalf("UpdateCleaningStatusCallCount = %d, want 1", cottageRepo.UpdateCleaningStatusCallCount)
			}
			if cottageRepo.LastUpdateCleaningStatusRoom != "A" {
				t.Fatalf("LastUpdateCleaningStatusRoom = %q, want %q", cottageRepo.LastUpdateCleaningStatusRoom, "A")
			}
			if cottageRepo.LastCleaningStatus != "PREPARED_FOR_SLEEP" {
				t.Fatalf("LastCleaningStatus = %q, want %q", cottageRepo.LastCleaningStatus, "PREPARED_FOR_SLEEP")
			}
			if stockRepo.ConsumeItemCallCount != 2 {
				t.Fatalf("ConsumeItemCallCount = %d, want 2", stockRepo.ConsumeItemCallCount)
			}
			if stockRepo.LastConsumeItemName != "aromaCandle" {
				t.Fatalf("LastConsumeItemName = %q, want %q", stockRepo.LastConsumeItemName, "aromaCandle")
			}
			if stockRepo.LastConsumeQuantity != 1 {
				t.Fatalf("LastConsumeQuantity = %d, want 1", stockRepo.LastConsumeQuantity)
			}

			select {
			case req := <-laundryRequests:
				t.Fatalf("unexpected laundry request: %+v", req)
			default:
			}
		}

		housekeeperRun(t, "PREPARE_FOR_SLEEP", validationFn)
	})
}

func housekeeperRun(t *testing.T, requestType string,
	validate func(t *testing.T, cottageRepo *fakes.FakeCottageRepo, stockRepo *fakes.FakeStockRepo, laundryRequests <-chan domain.WashRequest)) {

	t.Helper()

	cottageRepo := &fakes.FakeCottageRepo{}
	stockRepo := &fakes.FakeStockRepo{}
	laundryRequests := make(chan domain.WashRequest, 2)

	service, err := NewHousekeeperService(
		[]string{"Alice"},
		&fakes.FakeClockClient{},
		cottageRepo,
		stockRepo,
		nil,
		laundryRequests,
	)
	if err != nil {
		t.Fatalf("NewHousekeeperService() error = %v", err)
	}

	requests := make(chan domain.CleaningRequest, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		service.Work(ctx, requests)
		close(done)
	}()

	requests <- domain.CleaningRequest{RoomName: "A", RequestType: requestType}

	select {
	case <-done:
		t.Fatal("Work returned before cancellation")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	select {
	case <-done:
		validate(t, cottageRepo, stockRepo, laundryRequests)
	case <-time.After(time.Second):
		t.Fatal("Work did not return after cancellation")
	}

}

func TestHousekeeperActionConsumeItemRetry(t *testing.T) {
	t.Parallel()

	t.Run("restocks and retries when stock is insufficient", func(t *testing.T) {
		consumeCalls := 0
		stockRepo := &fakes.FakeStockRepo{
			ConsumeItemFn: func(ctx context.Context, itemName string, quantity int) error {
				if itemName != "wineBottle" {
					return nil
				}

				consumeCalls++
				if consumeCalls == 1 {
					return &dbErrors.StockInsufficientQuantityErr{ItemName: itemName, Quantity: quantity}
				}

				return nil
			},
		}

		housekeeper := &housekeeper{
			stockRepo: stockRepo,
			stoker:    &stocker{stockRepo: stockRepo},
		}

		housekeeper.actionConsumeItemRetry(context.Background(), "added wine bottle", "wineBottle", 1, "Alice", "A")

		if consumeCalls != 2 {
			t.Fatalf("consume calls = %d, want 2", consumeCalls)
		}

		if stockRepo.RestockItemCallCount != 1 {
			t.Fatalf("RestockItemCallCount = %d, want 1", stockRepo.RestockItemCallCount)
		}
	})

	t.Run("does not restock for non-stock errors", func(t *testing.T) {
		stockRepo := &fakes.FakeStockRepo{
			ConsumeItemFn: func(ctx context.Context, itemName string, quantity int) error {
				return errors.New("db down")
			},
		}

		housekeeper := &housekeeper{
			stockRepo: stockRepo,
			stoker:    &stocker{stockRepo: stockRepo},
		}

		housekeeper.actionConsumeItemRetry(context.Background(), "added wine bottle", "wineBottle", 1, "Alice", "A")

		if stockRepo.RestockItemCallCount != 0 {
			t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
		}

		if stockRepo.ConsumeItemCallCount != 1 {
			t.Fatalf("ConsumeItemCallCount = %d, want 1", stockRepo.ConsumeItemCallCount)
		}
	})
}
