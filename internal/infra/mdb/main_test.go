package mdb

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	mongoC      *mongodb.MongoDBContainer
	mongoClient *mongo.Client
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var err error
	mongoC, err = mongodb.Run(
		ctx,
		"mongo:latest",
		mongodb.WithUsername("test_user"),
		mongodb.WithPassword("test_pass"),
	)
	if err != nil {
		log.Fatalf("failed to start MongoDB container: %v", err)
	}

	uri, err := mongoC.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	mongoClient, err = mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("failed to connect mongo client: %v", err)
	}

	code := m.Run()

	_ = mongoClient.Disconnect(ctx)
	_ = testcontainers.TerminateContainer(mongoC)

	os.Exit(code)
}

func setupAndRun(t *testing.T, test func(t *testing.T, ct *mongo.Collection, st *mongo.Collection)) {
	t.Helper()

	dbName := "test_" + bson.NewObjectID().Hex()
	db := mongoClient.Database(dbName)

	cottageCollection := db.Collection("cottage")
	stockCollection := db.Collection("Stock")

	seed[documents.Cottage](t, cottageCollection, "../../../test_data/cottages.json")
	seed[documents.Stock](t, stockCollection, "../../../test_data/stocks.json")

	t.Cleanup(func() {
		_ = cottageCollection.Drop(context.Background())
		_ = stockCollection.Drop(context.Background())
		_ = db.Drop(context.Background())
	})

	test(t, cottageCollection, stockCollection)
}

func seed[D documents.Cottage | documents.Stock](t *testing.T, collection *mongo.Collection, filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	var items []D
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatal(err)
	}

	docs := make([]interface{}, len(items))
	for i, g := range items {
		docs[i] = g
	}
	if _, err := collection.InsertMany(context.Background(), docs); err != nil {
		t.Fatal(err)
	}
}
