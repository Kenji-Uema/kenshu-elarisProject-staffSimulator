package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

type washConfig struct {
	soap                 int
	cycleDurationInHours float64
}

var washTable = map[string]washConfig{
	"linens": {
		soap:                 20,
		cycleDurationInHours: 2,
	},
	"towels": {
		soap:                 10,
		cycleDurationInHours: 1,
	},
}

type LaundererService struct {
	employeeService[domain.WashRequest]
}

type launderer struct {
	clock            port.Clock
	hourChangeClient port.MqConsumer
	stockRepo        port.StockRepo
	stocker          immediateRestocker
}

func NewLaundererService(employeeNames []string,
	clock port.Clock, hourChangeClient port.MqConsumer, stockRepo port.StockRepo, stocker immediateRestocker) (*LaundererService, error) {

	employeeCount := len(employeeNames)
	if err := validation.New().
		NotZeroValue("clock", clock).
		NotZeroValue("hourChangeClient", hourChangeClient).
		NotZeroValue("stockRepo", stockRepo).
		PositiveValue("employeeCount", employeeCount).Validate(); err != nil {
		return nil, err
	}

	employees := make(map[string]*employeeStatus[domain.WashRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.WashRequest]{isIdle: true}
	}

	launderer := &launderer{
		clock:            clock,
		hourChangeClient: hourChangeClient,
		stockRepo:        stockRepo,
		stocker:          stocker,
	}

	return &LaundererService{
		employeeService: employeeService[domain.WashRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          launderer.work,
		},
	}, nil
}

func (l *launderer) work(ctx context.Context, employeeName string, request domain.WashRequest) {
	startTime, err := l.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get current time", "error", err)
		return
	}

	slog.InfoContext(ctx,
		"launderer started to workFn on washing request",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"item", request.Item,
		"startTime", startTime,
	)

	err = l.wash(ctx, request.Item, employeeName, request.RoomName, startTime)
	if err == nil {
		return
	}

	var stockErr *dbErrors.StockInsufficientQuantityErr
	if !errors.As(err, &stockErr) {
		slog.ErrorContext(ctx, "failed to wash item", "item", request.Item, "employeeName", employeeName, "roomName", request.RoomName, "error", err)
		return
	}

	slog.WarnContext(ctx,
		"insufficient stock to wash item, requesting restock",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"itemName", stockErr.ItemName,
		"requestedQuantity", stockErr.Quantity,
	)

	if restockErr := l.stocker.ImmediateRestock(ctx, request.Item); restockErr != nil {
		slog.ErrorContext(ctx, "failed to restock item", "employeeName", employeeName, "roomName", request.RoomName, "error", restockErr)
		return
	}

	if err := l.wash(ctx, request.Item, employeeName, request.RoomName, startTime); err != nil {
		slog.ErrorContext(ctx, "failed to wash item after restock", "employeeName", employeeName, "roomName", request.RoomName, "error", err)
	}
}

func (l *launderer) wash(ctx context.Context, item string, employeeName string, roomName string, startTime *time.Time) error {
	washConfig, ok := washTable[item]
	if !ok {
		return fmt.Errorf("unsupported wash item: %s", item)
	}

	err := l.stockRepo.ConsumeItem(ctx, item, washConfig.soap)
	if err == nil {
		finishTime, err := l.washingCycle(ctx, *startTime, washConfig.cycleDurationInHours)
		if err != nil {
			return err
		}

		slog.InfoContext(ctx, "launderer is done",
			"employeeName", employeeName,
			"roomName", roomName,
			"washing", item,
			"startTime", startTime,
			"finishTime", finishTime,
		)

		return nil
	}

	return err
}

func (l *launderer) washingCycle(ctx context.Context, startTime time.Time, cycleDurationInHours float64) (finishTime time.Time, err error) {
	deliveries, err := l.hourChangeClient.Consume(ctx)
	if err != nil {
		return time.Time{}, err
	}

	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return time.Time{}, errors.New("delivery channel closed")
			}

			currentTime, err := unmarshalTimeEvent(ctx, delivery.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshalCleaningRequest hour-change event", "error", err)
				return time.Time{}, err
			}

			if currentTime.Sub(startTime).Hours() >= cycleDurationInHours {
				return currentTime, nil
			}
		}
	}
}
