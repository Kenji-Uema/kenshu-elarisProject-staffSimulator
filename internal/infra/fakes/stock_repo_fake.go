package fakes

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

var _ port.StockRepo = (*FakeStockRepo)(nil)

type FakeStockRepo struct {
	GetStockFn    func(ctx context.Context) (map[string]documents.StockItem, error)
	ConsumeItemFn func(ctx context.Context, itemName string, quantity int) error
	RestockItemFn func(ctx context.Context, itemName string, quantity int) error

	GetStockCallCount    int
	ConsumeItemCallCount int
	RestockItemCallCount int

	LastGetStockCtx     context.Context
	LastConsumeItemCtx  context.Context
	LastConsumeItemName string
	LastConsumeQuantity int
	LastRestockItemCtx  context.Context
	LastRestockItemName string
	LastRestockQuantity int
}

func (f *FakeStockRepo) GetStock(ctx context.Context) (map[string]documents.StockItem, error) {
	f.GetStockCallCount++
	f.LastGetStockCtx = ctx

	if f.GetStockFn != nil {
		return f.GetStockFn(ctx)
	}

	return map[string]documents.StockItem{}, nil
}

func (f *FakeStockRepo) ConsumeItem(ctx context.Context, itemName string, quantity int) error {
	f.ConsumeItemCallCount++
	f.LastConsumeItemCtx = ctx
	f.LastConsumeItemName = itemName
	f.LastConsumeQuantity = quantity

	if f.ConsumeItemFn != nil {
		return f.ConsumeItemFn(ctx, itemName, quantity)
	}

	return nil
}

func (f *FakeStockRepo) RestockItem(ctx context.Context, itemName string, quantity int) error {
	f.RestockItemCallCount++
	f.LastRestockItemCtx = ctx
	f.LastRestockItemName = itemName
	f.LastRestockQuantity = quantity

	if f.RestockItemFn != nil {
		return f.RestockItemFn(ctx, itemName, quantity)
	}

	return nil
}
