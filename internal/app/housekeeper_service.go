package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

type HousekeeperService struct {
	employeeService[domain.CleaningRequest]
}

type immediateRestocker interface {
	ImmediateRestock(ctx context.Context, item string) error
}

type housekeeper struct {
	clock                port.Clock
	cottageRepo          port.CottageRepo
	stockRepo            port.StockRepo
	stoker               immediateRestocker
	laundererRequestChan chan<- domain.WashRequest
}

func NewHousekeeperService(employeeNames []string, clock port.Clock, cottageRepo port.CottageRepo,
	stockRepo port.StockRepo, stocker immediateRestocker, launderRequestCh chan<- domain.WashRequest) (*HousekeeperService, error) {
	employeeCount := len(employeeNames)

	if err := validation.New().
		NotZeroValue("cottageRepo", cottageRepo).
		NotZeroValue("stockRepo", stockRepo).
		PositiveValue("employeeCount", employeeCount).Validate(); err != nil {
		return nil, err
	}

	employees := make(map[string]*employeeStatus[domain.CleaningRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.CleaningRequest]{isIdle: true}
	}

	housekeeper := &housekeeper{
		clock:                clock,
		cottageRepo:          cottageRepo,
		stockRepo:            stockRepo,
		stoker:               stocker,
		laundererRequestChan: launderRequestCh,
	}

	return &HousekeeperService{
		employeeService: employeeService[domain.CleaningRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          housekeeper.work,
		},
	}, nil
}

func (h *housekeeper) work(ctx context.Context, employeeName string, request domain.CleaningRequest) {
	startTime, err := h.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get current time", "error", err)
		return
	}

	slog.InfoContext(ctx,
		"housekeeper started to workFn on cleaning request",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"request", request.RequestType,
		"startTime", startTime,
	)

	switch request.RequestType {
	case "PREPARE_FOR_GUEST":
		h.actionConsumeItemRetry(ctx, "added wine bottle", documents.WineBottle, 1, employeeName, request.RoomName)
		h.action(ctx, "added a vase of flower", employeeName, request.RoomName)
		h.action(ctx, "added a welcoming message", employeeName, request.RoomName)
		h.actionConsumeItemRetry(ctx, "added aroma candle", documents.AromaCandle, 1, employeeName, request.RoomName)
		h.updateCleaningStatus(ctx, employeeName, request.RoomName, "PREPARED_FOR_GUEST")
	case "DAILY_CLEANING":
		h.action(ctx, "trash taken out", employeeName, request.RoomName)
		h.action(ctx, "bed arranged", employeeName, request.RoomName)
		h.actionConsumeItemRetry(ctx, "bathroom cleaned", documents.CleaningItem, 5, employeeName, request.RoomName)
		h.actionLaundry(ctx, "towels changed", "towels", employeeName, request.RoomName)
		h.action(ctx, "vacuum cleaned", employeeName, request.RoomName)
		h.updateCleaningStatus(ctx, employeeName, request.RoomName, "DAILY_CLEANED")
	case "FULL_CLEANING":
		h.action(ctx, "trash taken out", employeeName, request.RoomName)
		h.actionLaundry(ctx, "towels changed", "towels", employeeName, request.RoomName)
		h.actionLaundry(ctx, "linens changed", "linens", employeeName, request.RoomName)
		h.actionConsumeItemRetry(ctx, "bathroom cleaned", documents.CleaningItem, 10, employeeName, request.RoomName)
		h.action(ctx, "bedroom detailed cleaned", employeeName, request.RoomName)
		h.updateCleaningStatus(ctx, employeeName, request.RoomName, "FULLY_CLEANED")
	case "PREPARE_FOR_SLEEP":
		h.action(ctx, "bed arranged for sleep", employeeName, request.RoomName)
		h.actionConsumeItemRetry(ctx, "added calm tea bags", documents.TeaBags, 1, employeeName, request.RoomName)
		h.actionConsumeItemRetry(ctx, "added aroma candle", documents.AromaCandle, 1, employeeName, request.RoomName)
		h.updateCleaningStatus(ctx, employeeName, request.RoomName, "PREPARED_FOR_SLEEP")
	default:
		slog.ErrorContext(ctx, "unknown cleaning request", "request", request.RequestType)
	}

	finishTime, err := h.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get current time", "error", err)
		return
	}

	slog.InfoContext(ctx,
		"housekeeper finished to workFn on cleaning request",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"request", request.RequestType,
		"startTime", startTime,
		"finishTime", finishTime,
	)
}

func (h *housekeeper) action(ctx context.Context, action string, employeeName string, roomName string) {
	slog.InfoContext(ctx,
		action,
		"employeeName", employeeName,
		"roomName", roomName,
	)
}

func (h *housekeeper) actionConsumeItemRetry(ctx context.Context, action string, item string, quantity int,
	employeeName string, roomName string) {
	err := h.actionConsumeItem(ctx, action, item, quantity, employeeName, roomName)
	if err == nil {
		return
	}

	var stockErr *dbErrors.StockInsufficientQuantityErr
	if !errors.As(err, &stockErr) {
		slog.ErrorContext(ctx, "failed to consume item", "employeeName", employeeName, "roomName", roomName, "error", err)
		return
	}

	slog.WarnContext(ctx,
		"stock item unavailable for housekeeper action, requesting restock",
		"employeeName", employeeName,
		"roomName", roomName,
		"itemName", stockErr.ItemName,
		"requestedQuantity", stockErr.Quantity,
	)

	if err := h.stoker.ImmediateRestock(ctx, item); err != nil {
		slog.ErrorContext(ctx, "failed to restock item", "employeeName", employeeName, "roomName", roomName, "error", err)
		return
	}

	if err := h.actionConsumeItem(ctx, action, item, quantity, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to consume item after restock", "employeeName", employeeName, "roomName", roomName, "error", err)
	}
}

func (h *housekeeper) actionConsumeItem(ctx context.Context, action string, item string, quantity int,
	employeeName string, roomName string) error {

	err := h.stockRepo.ConsumeItem(ctx, item, quantity)
	if err == nil {
		slog.InfoContext(ctx, action, "employeeName", employeeName, "roomName", roomName)

		return nil
	}

	return err
}

func (h *housekeeper) updateCleaningStatus(ctx context.Context, employeeName string, roomName string, cleaningStatus string) {
	if err := h.cottageRepo.UpdateCleaningStatus(ctx, roomName, cleaningStatus); err != nil {
		slog.ErrorContext(ctx,
			"failed to update cleaning status",
			"employeeName", employeeName,
			"roomName", roomName,
			"cleaningStatus", cleaningStatus,
			"error", err,
		)
	}
}

func (h *housekeeper) actionLaundry(ctx context.Context, action string, laundryItem string, employeeName string, roomName string) {
	laundryRequest := domain.WashRequest{
		RoomName: roomName,
		Item:     laundryItem,
	}

	h.laundererRequestChan <- laundryRequest

	slog.InfoContext(ctx,
		action,
		"laundryItem", laundryItem,
		"employeeName", employeeName,
		"roomName", roomName,
	)
}
