package app

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

const minimumStockQuantity = 10
const stockReplenishmentQuantity = 50

type StockerService struct {
	employeeService[domain.RestockRequest]
	immediateRestocker *stocker
}

type stocker struct {
	stockRepo port.StockRepo
}

func NewStockerService(employeeNames []string, stockRepo port.StockRepo) (*StockerService, error) {
	employeeCount := len(employeeNames)
	if err := validation.New().
		NotZeroValue("stockRepo", stockRepo).
		PositiveValue("employeeCount", employeeCount).Validate(); err != nil {
		return nil, err
	}

	employees := make(map[string]*employeeStatus[domain.RestockRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.RestockRequest]{isIdle: true}
	}

	stocker := &stocker{stockRepo: stockRepo}

	return &StockerService{
		employeeService: employeeService[domain.RestockRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          stocker.work,
		},
		immediateRestocker: stocker,
	}, nil
}

func (s *stocker) work(ctx context.Context, employeeName string, request domain.RestockRequest) {
	slog.InfoContext(ctx,
		"stocker started to workFn on restock request",
		"employeeName", employeeName,
		"itemsName", request.ItemsName,
	)

	if len(request.ItemsName) == 0 {
		slog.InfoContext(ctx, "no items requested for restock", "employeeName", employeeName)
		return
	}

	stock, err := s.stockRepo.GetStock(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get stock snapshot", "employeeName", employeeName, "error", err)
		return
	}

	for _, itemName := range request.ItemsName {
		item, ok := stock[itemName]
		if !ok {
			slog.WarnContext(ctx, "requested item not found in stock snapshot",
				"employeeName", employeeName,
				"itemName", itemName,
			)
			continue
		}

		if item.Quantity >= minimumStockQuantity {
			slog.InfoContext(ctx, "item already has enough stock",
				"employeeName", employeeName,
				"itemName", itemName,
				"currentQuantity", item.Quantity,
			)
			continue
		}

		if err := s.stockRepo.RestockItem(ctx, itemName, stockReplenishmentQuantity); err != nil {
			slog.ErrorContext(ctx, "failed to restock item",
				"employeeName", employeeName,
				"itemName", itemName,
				"currentQuantity", item.Quantity,
				"error", err,
			)
			continue
		}

		slog.InfoContext(ctx, "item restocked",
			"employeeName", employeeName,
			"itemName", itemName,
			"currentQuantity", item.Quantity,
			"addedQuantity", stockReplenishmentQuantity,
		)
	}

	slog.InfoContext(ctx,
		"stocker finished to workFn on restock request",
		"employeeName", employeeName,
		"itemsName", request.ItemsName,
	)
}

func (s *stocker) ImmediateRestock(ctx context.Context, item string) error {
	return s.stockRepo.RestockItem(ctx, item, stockReplenishmentQuantity)
}

func (s *StockerService) ImmediateRestock(ctx context.Context, item string) error {
	if s == nil || s.immediateRestocker == nil {
		return nil
	}

	return s.immediateRestocker.ImmediateRestock(ctx, item)
}
