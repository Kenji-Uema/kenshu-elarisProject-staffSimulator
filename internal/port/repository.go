package port

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
)

type CottageRepo interface {
	UpdateCleaningStatus(ctx context.Context, cottageName string, cleaningStatus string) error
}

type StockRepo interface {
	GetStock(ctx context.Context) (map[string]documents.StockItem, error)
	ConsumeItem(ctx context.Context, itemName string, quantity int) error
	RestockItem(ctx context.Context, itemName string, quantity int) error
}
