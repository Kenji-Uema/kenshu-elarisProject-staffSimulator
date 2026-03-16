package port

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type CottageRepo interface {
	GetByName(ctx context.Context, roomName string) (documents.Cottage, error)
	UpdateCurrentGuest(ctx context.Context, roomName string, guestId bson.ObjectID) error
	ClearCurrentGuest(ctx context.Context, roomName string) error
}

type StockRepo interface {
	GetStock(ctx context.Context, roomName string) (documents.Stock, error)
	ConsumeItem(ctx context.Context, itemName string, quantity int) error
	RestockItem(ctx context.Context, itemName string, quantity int) error
}
