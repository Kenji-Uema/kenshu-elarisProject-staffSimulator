package mdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestStockRepoGetStock(t *testing.T) {
	t.Parallel()

	t.Run("returns seeded stock", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &stockRepo{collection: collection}

			got, err := repo.GetStock(ctx)
			if err != nil {
				t.Fatalf("GetStock() error = %v", err)
			}

			if len(got) != len(documents.ManagedStockItemNames) {
				t.Fatalf("len(GetStock()) = %d, want %d", len(got), len(documents.ManagedStockItemNames))
			}
			if got[documents.CleaningItem].Quantity != 100 {
				t.Fatalf("cleaningItem quantity = %d, want 100", got[documents.CleaningItem].Quantity)
			}
			if got[documents.Soap].Quantity != 100 {
				t.Fatalf("soap quantity = %d, want 100", got[documents.Soap].Quantity)
			}
			if got[documents.AromaCandle].Name != documents.AromaCandle {
				t.Fatalf("aromaCandle name = %q, want %q", got[documents.AromaCandle].Name, documents.AromaCandle)
			}
		})
	})

	t.Run("returns not found when stock document does not exist", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := collection.Drop(ctx); err != nil {
				t.Fatalf("Drop() error = %v", err)
			}

			repo := &stockRepo{collection: collection}

			_, err := repo.GetStock(ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.ErrStockDoesNotExist
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})

	t.Run("returns corrupted data for malformed stock document", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := collection.Drop(ctx); err != nil {
				t.Fatalf("Drop() error = %v", err)
			}

			_, err := collection.InsertOne(ctx, bson.M{
				"cleaning_items": bson.M{"name": "cleaningItem", "quantity": 100},
				"soap":           "invalid-shape",
			})
			if err != nil {
				t.Fatalf("InsertOne() error = %v", err)
			}

			repo := &stockRepo{collection: collection}

			_, err = repo.GetStock(ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.CorruptedDataErr
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})
}

func TestStockRepoConsumeItem(t *testing.T) {
	t.Parallel()

	t.Run("consumes quantity from existing stock item", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &stockRepo{collection: collection}

			if err := repo.ConsumeItem(ctx, documents.CleaningItem, 5); err != nil {
				t.Fatalf("ConsumeItem() error = %v", err)
			}
			if err := repo.ConsumeItem(ctx, documents.Soap, 5); err != nil {
				t.Fatalf("ConsumeItem() error = %v", err)
			}

			got := mustFindStock(t, ctx, collection)
			if got.CleaningItems.Quantity != 95 {
				t.Fatalf("CleaningItems.Quantity = %d, want 95", got.CleaningItems.Quantity)
			}
			if got.Soap.Quantity != 95 {
				t.Fatalf("Soap.Quantity = %d, want 95", got.Soap.Quantity)
			}
		})
	})

	t.Run("consumes quantity from soap stock item", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &stockRepo{collection: collection}

			if err := repo.ConsumeItem(ctx, documents.Soap, 5); err != nil {
				t.Fatalf("ConsumeItem() error = %v", err)
			}

			got := mustFindStock(t, ctx, collection)
			if got.Soap.Quantity != 95 {
				t.Fatalf("Soap.Quantity = %d, want 95", got.Soap.Quantity)
			}
		})
	})

	t.Run("returns insufficient quantity when stock is too low", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &stockRepo{collection: collection}

			err := repo.ConsumeItem(ctx, documents.CleaningItem, 101)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.StockInsufficientQuantityErr
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})

	t.Run("returns insufficient quantity when soap stock is too low", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &stockRepo{collection: collection}

			err := repo.ConsumeItem(ctx, documents.Soap, 101)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.StockInsufficientQuantityErr
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})

	t.Run("returns not found when stock document does not exist", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := collection.Drop(ctx); err != nil {
				t.Fatalf("Drop() error = %v", err)
			}

			repo := &stockRepo{collection: collection}

			err := repo.ConsumeItem(ctx, documents.CleaningItem, 1)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.ErrStockDoesNotExist
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})

	t.Run("returns not found when stock document does not exist for soap", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := collection.Drop(ctx); err != nil {
				t.Fatalf("Drop() error = %v", err)
			}

			repo := &stockRepo{collection: collection}

			err := repo.ConsumeItem(ctx, documents.Soap, 1)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.ErrStockDoesNotExist
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})
}

func TestStockRepoRestockItem(t *testing.T) {
	t.Parallel()

	t.Run("adds quantity to existing stock item", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &stockRepo{collection: collection}

			if err := repo.RestockItem(ctx, documents.TeaBags, 15); err != nil {
				t.Fatalf("RestockItem() error = %v", err)
			}

			got := mustFindStock(t, ctx, collection)
			if got.TeaBags.Quantity != 115 {
				t.Fatalf("TeaBags.Quantity = %d, want 115", got.TeaBags.Quantity)
			}
		})
	})

	t.Run("returns not found when stock document does not exist", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, _ *mongo.Collection, collection *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := collection.Drop(ctx); err != nil {
				t.Fatalf("Drop() error = %v", err)
			}

			repo := &stockRepo{collection: collection}

			err := repo.RestockItem(ctx, documents.TeaBags, 1)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.ErrStockDoesNotExist
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})
}

func mustFindStock(t *testing.T, ctx context.Context, collection *mongo.Collection) documents.Stock {
	t.Helper()

	var stock documents.Stock
	if err := collection.FindOne(ctx, bson.M{}).Decode(&stock); err != nil {
		t.Fatalf("FindOne().Decode() error = %v", err)
	}

	return stock
}
