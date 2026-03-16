package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

const minimumStockQuantity = 50
const stockReplenishmentQuantity = 50
const defaultStockName = "stock"

var managedStockItems = []string{
	"cleaningItem",
	"bathroomAmenities",
	"aromaCandle",
	"waterBottle",
	"wineBottle",
	"teaBags",
	"sweets",
	"chips",
}

type StockerService struct {
	EmployeeService[domain.RestockRequest]
}

type Stocker struct {
	stockRepo port.StockRepo
}

func NewStockerService(employeeNames []string, stockRepo port.StockRepo) (*StockerService, error) {
	employeeCount := len(employeeNames)
	if employeeCount == 0 {
		return nil, fmt.Errorf("worker count must be greater than 0")
	}

	employees := make(map[string]*employeeStatus[domain.RestockRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.RestockRequest]{isIdle: true}
	}

	stocker := &Stocker{stockRepo: stockRepo}

	return &StockerService{
		EmployeeService: EmployeeService[domain.RestockRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          stocker.work,
		},
	}, nil
}

func (s *Stocker) work(ctx context.Context, employeeName string, request domain.RestockRequest) {
	if len(request.ItemsName) == 0 {
		slog.InfoContext(ctx, "no items requested for restock", "employeeName", employeeName)
		return
	}

	stock, err := s.stockRepo.GetStock(ctx, defaultStockName)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get stock snapshot", "employeeName", employeeName, "stockName", defaultStockName, "error", err)
		return
	}

	for _, itemName := range request.ItemsName {
		quantity, ok := stockQuantityByItemName(stock, itemName)
		if !ok {
			slog.WarnContext(ctx, "requested item not found in stock snapshot",
				"employeeName", employeeName,
				"stockName", defaultStockName,
				"itemName", itemName,
			)
			continue
		}

		if quantity >= minimumStockQuantity {
			continue
		}

		if err := s.stockRepo.RestockItem(ctx, itemName, stockReplenishmentQuantity); err != nil {
			slog.ErrorContext(ctx, "failed to restock item",
				"employeeName", employeeName,
				"stockName", defaultStockName,
				"itemName", itemName,
				"currentQuantity", quantity,
				"error", err,
			)
			continue
		}

		slog.InfoContext(ctx, "item restocked",
			"employeeName", employeeName,
			"stockName", defaultStockName,
			"itemName", itemName,
			"currentQuantity", quantity,
			"addedQuantity", stockReplenishmentQuantity,
		)
	}
}

func (s *Stocker) immediateRestock(ctx context.Context, item string) error {
	return s.stockRepo.RestockItem(ctx, item, stockReplenishmentQuantity)
}

func stockQuantityByItemName(stock documents.Stock, itemName string) (int, bool) {
	switch itemName {
	case "cleaningItem", "cleaning_item", "cleaning_items":
		return stock.CleaningItems.Quantity, true
	case "bathroomAmenities", "bathroom_amenities":
		return stock.BathroomAmenities.Quantity, true
	case "aromaCandle", "aroma_candle", "aroma_candles":
		return stock.AromaCandles.Quantity, true
	case "waterBottle", "water_bottle":
		return stock.WaterBottle.Quantity, true
	case "wineBottle", "wine_bottle", "wine":
		return stock.WineBottle.Quantity, true
	case "teaBags", "tea_bags":
		return stock.TeaBags.Quantity, true
	case "sweets":
		return stock.Sweets.Quantity, true
	case "chips":
		return stock.Chips.Quantity, true
	default:
		return 0, false
	}
}
