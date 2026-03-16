package mdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type stockRepo struct {
	collection *mongo.Collection
	mu         sync.Mutex
}

const defaultStockCollectionName = "Stock"

func NewStockRepo(db *mongo.Database) *stockRepo {
	return &stockRepo{collection: db.Collection(defaultStockCollectionName)}
}

func (s *stockRepo) GetStock(ctx context.Context, roomName string) (documents.Stock, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validation.New().NotBlank("roomName", roomName).Validate(); err != nil {
		return documents.Stock{}, err
	}

	filter := bson.M{"name": roomName}

	var stock documents.Stock
	if err := s.collection.FindOne(ctx, filter).Decode(&stock); err != nil {
		return documents.Stock{}, fmt.Errorf("%w: could not find stock for roomName=%s: %v",
			dbErrors.ErrStockRepo, roomName, err)
	}

	return stock, nil
}

func (s *stockRepo) ConsumeItem(ctx context.Context, itemName string, quantity int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validation.New().NotBlank("itemName", itemName).PositiveValue("quantity", quantity).Validate(); err != nil {
		return err
	}

	itemQuantityField, err := stockItemQuantityField(itemName)
	if err != nil {
		return err
	}

	filter := bson.M{itemQuantityField: bson.M{"$gte": quantity}}
	update := bson.M{"$inc": bson.M{itemQuantityField: -quantity}}

	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%w: could not consume stock item itemName=%s quantity=%d: %v",
			dbErrors.ErrStockRepo, itemName, quantity, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: item unavailable or insufficient quantity itemName=%s quantity=%d",
			dbErrors.ErrStockRepo, itemName, quantity)
	}

	return nil
}

func (s *stockRepo) RestockItem(ctx context.Context, itemName string, quantity int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validation.New().NotBlank("itemName", itemName).PositiveValue("quantity", quantity).Validate(); err != nil {
		return err
	}

	itemQuantityField, err := stockItemQuantityField(itemName)
	if err != nil {
		return err
	}

	update := bson.M{"$inc": bson.M{itemQuantityField: quantity}}

	result, err := s.collection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return fmt.Errorf("%w: could not restock item itemName=%s quantity=%d: %v",
			dbErrors.ErrStockRepo, itemName, quantity, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: stock document does not exist for restock itemName=%s",
			dbErrors.ErrStockRepo, itemName)
	}

	return nil
}

func stockItemQuantityField(itemName string) (string, error) {
	switch itemName {
	case "cleaning_items", "cleaning_item", "cleaningItem":
		return "cleaning_items.quantity", nil
	case "bathroom_amenities", "bathroomAmenities":
		return "bathroom_amenities.quantity", nil
	case "aroma_candles", "aroma_candle", "aromaCandle":
		return "aroma_candles.quantity", nil
	case "water_bottle", "waterBottle":
		return "water_bottle.quantity", nil
	case "wine_bottle", "wineBottle", "wine":
		return "wine_bottle.quantity", nil
	case "tea_bags", "teaBags":
		return "tea_bags.quantity", nil
	case "sweets":
		return "sweets.quantity", nil
	case "chips":
		return "chips.quantity", nil
	default:
		return "", fmt.Errorf("unsupported stock item: %s", itemName)
	}
}
