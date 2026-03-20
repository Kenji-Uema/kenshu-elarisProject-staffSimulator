package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func SeedMongoFromFixtures(t TestReporter, mongoHost string, database string, cottagesFixturePath string, stocksFixturePath string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(fmt.Sprintf("mongodb://test_user:test_pass@%s", mongoHost)))
	if err != nil {
		t.Fatalf("connect mongo for fixture seed: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	if err := waitForMongoReady(ctx, client); err != nil {
		t.Fatalf("wait for mongo readiness: %v", err)
	}

	db := client.Database(database)
	insertFixture[documents.Cottage](t, ctx, db.Collection("cottage"), cottagesFixturePath)
	insertFixture[documents.Stock](t, ctx, db.Collection("Stock"), stocksFixturePath)
}

func waitForMongoReady(ctx context.Context, client *mongo.Client) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := client.Database("admin").RunCommand(pingCtx, bson.M{"ping": 1}).Err()
		cancel()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func insertFixture[T documents.Cottage | documents.Stock](t TestReporter, ctx context.Context, collection *mongo.Collection, fixturePath string) {
	t.Helper()

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixturePath, err)
	}

	var docs []T
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("unmarshal fixture %q: %v", fixturePath, err)
	}

	items := make([]any, len(docs))
	for i, doc := range docs {
		items[i] = doc
	}

	if _, err := collection.InsertMany(ctx, items); err != nil {
		t.Fatalf("insert fixture docs into %q: %v", collection.Name(), err)
	}
}
