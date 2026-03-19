package mdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type stockRepo struct {
	collection *mongo.Collection
	mu         sync.Mutex
}

func NewStockRepo(db *mongo.Database) port.StockRepo {
	return &stockRepo{collection: db.Collection("Stock")}
}

func (s *stockRepo) GetStock(ctx context.Context) (map[string]documents.StockItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stock documents.Stock
	result := s.collection.FindOne(ctx, bson.M{})
	if err := result.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			slog.WarnContext(ctx, "stock not found")
			return nil, &dbErrors.ErrStockDoesNotExist{}
		}

		slog.ErrorContext(ctx, "failed to find stock", "error", err)
		return nil, &dbErrors.UnexpectedErr{Msg: "failed to find stock", Err: err}
	}

	if err := result.Decode(&stock); err != nil {
		slog.ErrorContext(ctx, "failed to decode stock", "error", err)
		return nil, &dbErrors.CorruptedDataErr{Err: err}
	}

	return stock.AsMap(), nil
}

func (s *stockRepo) ConsumeItem(ctx context.Context, itemName string, quantity int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validation.New().
		NotBlank("itemName", itemName).
		PositiveValue("quantity", quantity).Validate(); err != nil {

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
		return &dbErrors.UnexpectedErr{
			Msg: fmt.Sprintf("could not consume stock item itemName=%s quantity=%d", itemName, quantity),
			Err: err,
		}
	}

	if result.MatchedCount == 0 {
		exists, err := s.stockDocumentExists(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return &dbErrors.ErrStockDoesNotExist{}
		}

		return &dbErrors.StockInsufficientQuantityErr{ItemName: itemName, Quantity: quantity}
	}

	if result.ModifiedCount == 0 {
		return &dbErrors.StockItemQuantityNotUpdatedErr{ItemName: itemName}
	}

	return nil
}

func (s *stockRepo) stockDocumentExists(ctx context.Context) (bool, error) {
	err := s.collection.FindOne(ctx, bson.M{}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}

	return false, &dbErrors.UnexpectedErr{Msg: "failed to find stock", Err: err}
}

func (s *stockRepo) RestockItem(ctx context.Context, itemName string, quantity int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validation.New().
		NotBlank("itemName", itemName).
		PositiveValue("quantity", quantity).Validate(); err != nil {

		return err
	}

	itemQuantityField, err := stockItemQuantityField(itemName)
	if err != nil {
		return err
	}

	update := bson.M{"$inc": bson.M{itemQuantityField: quantity}}

	result, err := s.collection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return &dbErrors.UnexpectedErr{
			Msg: fmt.Sprintf("could not restock stock item itemName=%s quantity=%d", itemName, quantity),
			Err: err,
		}
	}

	if result.MatchedCount == 0 {
		return &dbErrors.ErrStockDoesNotExist{}
	}

	if result.ModifiedCount == 0 {
		return &dbErrors.StockItemQuantityNotUpdatedErr{ItemName: itemName}
	}

	return nil
}

func stockItemQuantityField(itemName string) (string, error) {
	switch itemName {
	case documents.CleaningItem:
		return "cleaning_items.quantity", nil
	case documents.BathroomAmenities:
		return "bathroom_amenities.quantity", nil
	case documents.AromaCandle:
		return "aroma_candles.quantity", nil
	case documents.WaterBottle:
		return "water_bottle.quantity", nil
	case documents.WineBottle:
		return "wine_bottle.quantity", nil
	case documents.TeaBags:
		return "tea_bags.quantity", nil
	case documents.Sweets:
		return "sweets.quantity", nil
	case documents.Chips:
		return "chips.quantity", nil
	default:
		return "", &dbErrors.StockItemQuantityNotUpdatedErr{ItemName: itemName}
	}
}
