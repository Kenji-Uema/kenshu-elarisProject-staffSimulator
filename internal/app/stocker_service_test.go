package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/fakes"
)

func TestNewStockerService(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no employees are configured", func(t *testing.T) {
		service, err := NewStockerService(nil, &fakes.FakeStockRepo{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when stock repo is nil", func(t *testing.T) {
		var stockRepo *fakes.FakeStockRepo

		service, err := NewStockerService([]string{"Alice"}, stockRepo)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("creates service with configured dependencies", func(t *testing.T) {
		service, err := NewStockerService([]string{"Alice", "Bob"}, &fakes.FakeStockRepo{})
		if err != nil {
			t.Fatalf("NewStockerService() error = %v", err)
		}

		if service.employeeCount != 2 {
			t.Fatalf("employeeCount = %d, want 2", service.employeeCount)
		}

		if service.work == nil {
			t.Fatal("expected stocker work handler to be configured")
		}
	})
}

func TestStockerWork(t *testing.T) {
	t.Parallel()

	t.Run("returns when no items are requested", func(t *testing.T) {
		t.Parallel()

		stockRepo := &fakes.FakeStockRepo{}
		stocker := &stocker{stockRepo: stockRepo}

		stocker.work(context.Background(), "Alice", domain.RestockRequest{})

		if stockRepo.GetStockCallCount != 0 {
			t.Fatalf("GetStockCallCount = %d, want 0", stockRepo.GetStockCallCount)
		}
		if stockRepo.RestockItemCallCount != 0 {
			t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
		}
	})

	t.Run("returns when loading stock fails", func(t *testing.T) {
		t.Parallel()

		stockRepo := &fakes.FakeStockRepo{
			GetStockFn: func(ctx context.Context) (map[string]documents.StockItem, error) {
				return nil, errors.New("db down")
			},
		}
		stocker := &stocker{stockRepo: stockRepo}

		stocker.work(context.Background(), "Alice", domain.RestockRequest{
			ItemsName: []string{"teaBags"},
		})

		if stockRepo.GetStockCallCount != 1 {
			t.Fatalf("GetStockCallCount = %d, want 1", stockRepo.GetStockCallCount)
		}
		if stockRepo.RestockItemCallCount != 0 {
			t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
		}
	})

	t.Run("restocks items below minimum quantity", func(t *testing.T) {
		t.Parallel()

		stockRepo := &fakes.FakeStockRepo{
			GetStockFn: func(ctx context.Context) (map[string]documents.StockItem, error) {
				return map[string]documents.StockItem{
					"teaBags": {Name: "teaBags", Quantity: minimumStockQuantity - 1},
				}, nil
			},
		}
		stocker := &stocker{stockRepo: stockRepo}

		stocker.work(context.Background(), "Alice", domain.RestockRequest{
			ItemsName: []string{"teaBags"},
		})

		if stockRepo.GetStockCallCount != 1 {
			t.Fatalf("GetStockCallCount = %d, want 1", stockRepo.GetStockCallCount)
		}
		if stockRepo.RestockItemCallCount != 1 {
			t.Fatalf("RestockItemCallCount = %d, want 1", stockRepo.RestockItemCallCount)
		}
		if stockRepo.LastRestockItemName != "teaBags" {
			t.Fatalf("LastRestockItemName = %q, want %q", stockRepo.LastRestockItemName, "teaBags")
		}
		if stockRepo.LastRestockQuantity != stockReplenishmentQuantity {
			t.Fatalf("LastRestockQuantity = %d, want %d", stockRepo.LastRestockQuantity, stockReplenishmentQuantity)
		}
	})

	t.Run("does not restock items already above minimum quantity", func(t *testing.T) {
		t.Parallel()

		stockRepo := &fakes.FakeStockRepo{
			GetStockFn: func(ctx context.Context) (map[string]documents.StockItem, error) {
				return map[string]documents.StockItem{
					"teaBags": {Name: "teaBags", Quantity: minimumStockQuantity},
				}, nil
			},
		}
		stocker := &stocker{stockRepo: stockRepo}

		stocker.work(context.Background(), "Alice", domain.RestockRequest{
			ItemsName: []string{"teaBags"},
		})

		if stockRepo.GetStockCallCount != 1 {
			t.Fatalf("GetStockCallCount = %d, want 1", stockRepo.GetStockCallCount)
		}
		if stockRepo.RestockItemCallCount != 0 {
			t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
		}
	})

	t.Run("skips items not present in stock snapshot", func(t *testing.T) {
		t.Parallel()

		stockRepo := &fakes.FakeStockRepo{
			GetStockFn: func(ctx context.Context) (map[string]documents.StockItem, error) {
				return map[string]documents.StockItem{
					"teaBags": {Name: "teaBags", Quantity: 1},
				}, nil
			},
		}
		stocker := &stocker{stockRepo: stockRepo}

		stocker.work(context.Background(), "Alice", domain.RestockRequest{
			ItemsName: []string{"unknown"},
		})

		if stockRepo.GetStockCallCount != 1 {
			t.Fatalf("GetStockCallCount = %d, want 1", stockRepo.GetStockCallCount)
		}
		if stockRepo.RestockItemCallCount != 0 {
			t.Fatalf("RestockItemCallCount = %d, want 0", stockRepo.RestockItemCallCount)
		}
	})
}
