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

func TestCottageRepoUpdateCleaningStatus(t *testing.T) {
	t.Parallel()

	t.Run("updates existing cottage cleaning status", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, collection *mongo.Collection, _ *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &cottageRepo{collection: collection}

			if err := repo.UpdateCleaningStatus(ctx, "A", "PREPARED_FOR_GUEST"); err != nil {
				t.Fatalf("UpdateCleaningStatus() error = %v", err)
			}

			got := mustFindCottageByName(t, ctx, collection, "A")
			if got.CleaningStatus != "PREPARED_FOR_GUEST" {
				t.Fatalf("CleaningStatus = %q, want %q", got.CleaningStatus, "PREPARED_FOR_GUEST")
			}
		})
	})

	t.Run("returns not found when cottage does not exist", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, collection *mongo.Collection, _ *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &cottageRepo{collection: collection}

			err := repo.UpdateCleaningStatus(ctx, "missing", "PREPARED_FOR_GUEST")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.ErrCottageDoesNotExist
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})

	t.Run("returns not updated when cleaning status is unchanged", func(t *testing.T) {
		setupAndRun(t, func(t *testing.T, collection *mongo.Collection, _ *mongo.Collection) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			repo := &cottageRepo{collection: collection}

			err := repo.UpdateCleaningStatus(ctx, "B", "DAILY_CLEANED")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var target *dbErrors.CottageCleaningStatusNotUpdatedErr
			if !errors.As(err, &target) {
				t.Fatalf("error type = %T, want %T", err, target)
			}
		})
	})
}

func mustFindCottageByName(t *testing.T, ctx context.Context, collection *mongo.Collection, name string) documents.Cottage {
	t.Helper()

	var cottage documents.Cottage
	if err := collection.FindOne(ctx, bson.M{"name": name}).Decode(&cottage); err != nil {
		t.Fatalf("FindOne().Decode() error = %v", err)
	}

	return cottage
}
