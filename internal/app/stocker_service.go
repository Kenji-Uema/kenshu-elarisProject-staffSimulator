package app

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"go.opentelemetry.io/otel/attribute"
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
	workCtx, span := telemetry.StartSpan(ctx, "staff.stocker.restock_request",
		telemetry.RequestAttributes(employeeName),
		attribute.Int("staff.restock.item_count", len(request.ItemsName)),
	)
	defer span.End()

	slog.InfoContext(workCtx,
		"stocker started to workFn on restock request",
		"employeeName", employeeName,
		"itemsName", request.ItemsName,
	)

	if len(request.ItemsName) == 0 {
		slog.InfoContext(workCtx, "no items requested for restock", "employeeName", employeeName)
		return
	}

	stock, err := s.stockRepo.GetStock(workCtx)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		slog.ErrorContext(workCtx, "failed to get stock snapshot", "employeeName", employeeName, "error", err)
		return
	}

	for _, itemName := range request.ItemsName {
		itemCtx, itemSpan := telemetry.StartSpan(workCtx, "staff.stocker.restock_item",
			telemetry.RequestAttributes(employeeName),
			attribute.String("staff.item_name", itemName),
		)
		item, ok := stock[itemName]
		if !ok {
			slog.WarnContext(itemCtx, "requested item not found in stock snapshot",
				"employeeName", employeeName,
				"itemName", itemName,
			)
			itemSpan.End()
			continue
		}

		if item.Quantity >= minimumStockQuantity {
			slog.InfoContext(itemCtx, "item already has enough stock",
				"employeeName", employeeName,
				"itemName", itemName,
				"currentQuantity", item.Quantity,
			)
			itemSpan.End()
			continue
		}

		if err := s.stockRepo.RestockItem(itemCtx, itemName, stockReplenishmentQuantity); err != nil {
			telemetry.RecordSpanError(itemSpan, err)
			slog.ErrorContext(itemCtx, "failed to restock item",
				"employeeName", employeeName,
				"itemName", itemName,
				"currentQuantity", item.Quantity,
				"error", err,
			)
			itemSpan.End()
			continue
		}

		slog.InfoContext(itemCtx, "item restocked",
			"employeeName", employeeName,
			"itemName", itemName,
			"currentQuantity", item.Quantity,
			"addedQuantity", stockReplenishmentQuantity,
		)
		itemSpan.End()
	}

	slog.InfoContext(workCtx,
		"stocker finished to workFn on restock request",
		"employeeName", employeeName,
		"itemsName", request.ItemsName,
	)
}

func (s *stocker) ImmediateRestock(ctx context.Context, item string) error {
	spanCtx, span := telemetry.StartSpan(ctx, "staff.stocker.immediate_restock",
		attribute.String("staff.item_name", item),
		attribute.Int("staff.item_quantity", stockReplenishmentQuantity),
	)
	defer span.End()

	err := s.stockRepo.RestockItem(spanCtx, item, stockReplenishmentQuantity)
	if err != nil {
		telemetry.RecordSpanError(span, err)
	}
	return err
}

func (s *StockerService) ImmediateRestock(ctx context.Context, item string) error {
	if s == nil || s.immediateRestocker == nil {
		return nil
	}

	return s.immediateRestocker.ImmediateRestock(ctx, item)
}
