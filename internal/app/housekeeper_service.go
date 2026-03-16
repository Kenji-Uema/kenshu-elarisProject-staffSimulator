package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"github.com/Kenji-Uema/staffSimulator/internal/transport/grpc/clock"
)

type HousekeeperService struct {
	EmployeeService[domain.CleaningRequest]
}

type Housekeeper struct {
	clock                clock.Clock
	stockRepo            port.StockRepo
	laundererRequestChan chan<- domain.WashRequest
}

func NewHousekeeperService(employeeNames []string, clock clock.Clock, stockRepo port.StockRepo, launderRequestCh chan<- domain.WashRequest) (*HousekeeperService, error) {
	employeeCount := len(employeeNames)
	if employeeCount == 0 {
		return nil, fmt.Errorf("worker count must be greater than 0")
	}

	employees := make(map[string]*employeeStatus[domain.CleaningRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.CleaningRequest]{isIdle: true}
	}

	housekeeper := &Housekeeper{
		clock:                clock,
		stockRepo:            stockRepo,
		laundererRequestChan: launderRequestCh,
	}

	return &HousekeeperService{
		EmployeeService: EmployeeService[domain.CleaningRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          housekeeper.work,
		},
	}, nil
}

func (h *Housekeeper) work(ctx context.Context, employeeName string, request domain.CleaningRequest) {
	startTime, err := h.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get current time", "error", err)
		return
	}

	slog.InfoContext(ctx,
		"housekeeper started to work on cleaning request",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"request", request.R,
		"startTime", startTime,
	)

	switch request.R {
	case "PREPARE_FOR_GUEST":
		h.prepareForGuest(ctx, employeeName, request.RoomName)
	case "DAILY_CLEANING":
		h.dailyCleaning(ctx, employeeName, request.RoomName)
	case "FULL_CLEANING":
		h.fullCleaning(ctx, employeeName, request.RoomName)
	case "PREPARE_FOR_SLEEP":
		h.prepareForSleep(ctx, employeeName, request.RoomName)
	default:
		slog.ErrorContext(ctx, "unknown cleaning request", "request", request.R)
	}

	finishTime, err := h.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get current time", "error", err)
		return
	}

	slog.InfoContext(ctx,
		"housekeeper finished to work on cleaning request",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"request", request.R,
		"startTime", startTime,
		"finishTime", finishTime,
	)
}

func (h *Housekeeper) prepareForGuest(ctx context.Context, employeeName string, roomName string) {
	if err := h.actionConsumeItem(ctx, "added wine bottle", "wine", 1, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to add wine bottle", "error", err)
	}

	h.action(ctx, "added a vase of flower", employeeName, roomName)

	h.action(ctx, "added a welcoming message", employeeName, roomName)

	if err := h.actionConsumeItem(ctx, "added aroma candle", "aromaCandle", 1, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to add aroma candle", "error", err)
	}
}

func (h *Housekeeper) dailyCleaning(ctx context.Context, employeeName string, roomName string) {
	h.action(ctx, "trash taken out", employeeName, roomName)

	h.action(ctx, "bed arranged", employeeName, roomName)

	if err := h.actionConsumeItem(ctx, "bathroom cleaned", "cleaningItem", 5, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to clean bathroom", "error", err)
	}

	h.actionLaundry(ctx, "towels changed", "towels", employeeName, roomName)

	if err := h.actionRestockAmenities(ctx, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to restock amenities", "error", err)
	}

	h.action(ctx, "vacuum cleaned", employeeName, roomName)
}

func (h *Housekeeper) fullCleaning(ctx context.Context, employeeName string, roomName string) {
	h.action(ctx, "trash taken out", employeeName, roomName)

	h.actionLaundry(ctx, "towels changed", "towels", employeeName, roomName)

	h.actionLaundry(ctx, "linens changed", "linens", employeeName, roomName)

	if err := h.actionConsumeItem(ctx, "bathroom cleaned", "cleaningItem", 10, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to clean bathroom", "error", err)
	}

	h.action(ctx, "bedroom detailed cleaned", employeeName, roomName)

	if err := h.actionRestockAmenities(ctx, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to restock amenities", "error", err)
	}
}

func (h *Housekeeper) prepareForSleep(ctx context.Context, employeeName string, roomName string) {
	h.action(ctx, "bed arranged for sleep", employeeName, roomName)

	if err := h.actionConsumeItem(ctx, "added calm tea bags", "teaBags", 1, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to add calm tea bags", "error", err)
	}

	if err := h.actionConsumeItem(ctx, "added aroma candle", "aromaCandle", 1, employeeName, roomName); err != nil {
		slog.ErrorContext(ctx, "failed to add aroma candle", "error", err)
	}
}

func (h *Housekeeper) action(ctx context.Context, action string, employeeName string, roomName string) {
	slog.InfoContext(ctx,
		action,
		"employeeName", employeeName,
		"roomName", roomName,
	)
}

func (h *Housekeeper) actionConsumeItem(ctx context.Context, action string, item string, quantity int,
	employeeName string, roomName string) error {

	if err := h.stockRepo.ConsumeItem(ctx, item, quantity); err != nil {
		return err
	}

	slog.InfoContext(ctx,
		action,
		"employeeName", employeeName,
		"roomName", roomName,
	)

	return nil
}

func (h *Housekeeper) actionLaundry(ctx context.Context, action string, laundryItem string, employeeName string, roomName string) {
	laundryRequest := domain.WashRequest{
		RoomName: roomName,
	}

	if laundryItem != "linens" {
		laundryRequest.Linens = true
	}

	if laundryItem != "towels" {
		laundryRequest.Towels = true
	}

	h.laundererRequestChan <- laundryRequest

	slog.InfoContext(ctx,
		action,
		"laundryItem", laundryItem,
		"employeeName", employeeName,
		"roomName", roomName,
	)
}

func (h *Housekeeper) actionRestockAmenities(ctx context.Context, employeeName string, roomName string) error {

	var restockErr error
	if err := h.stockRepo.ConsumeItem(ctx, "waterBottle", 2); err != nil {
		restockErr = errors.Join(restockErr, err)
	}
	if err := h.stockRepo.ConsumeItem(ctx, "teaBags", 2); err != nil {
		restockErr = errors.Join(restockErr, err)
	}
	if err := h.stockRepo.ConsumeItem(ctx, "sweets", 3); err != nil {
		restockErr = errors.Join(restockErr, err)
	}
	if err := h.stockRepo.ConsumeItem(ctx, "chips", 1); err != nil {
		restockErr = errors.Join(restockErr, err)
	}
	if err := h.stockRepo.ConsumeItem(ctx, "bathroomAmenities", 5); err != nil {
		restockErr = errors.Join(restockErr, err)
	}

	if restockErr != nil {
		return restockErr
	}

	slog.InfoContext(ctx,
		"amenities restocked",
		"employeeName", employeeName,
		"roomName", roomName,
	)

	return nil
}
