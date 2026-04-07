package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"go.opentelemetry.io/otel/attribute"
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
	clock      port.Clock
	hourChange timeEventRegistry
	stockRepo  port.StockRepo
	stocker    immediateRestocker
}

func NewLaundererService(employeeNames []string,
	clock port.Clock, hourChange timeEventRegistry, stockRepo port.StockRepo, stocker immediateRestocker) (*LaundererService, error) {

	employeeCount := len(employeeNames)
	if err := validation.New().
		NotZeroValue("clock", clock).
		NotZeroValue("hourChange", hourChange).
		NotZeroValue("stockRepo", stockRepo).
		PositiveValue("employeeCount", employeeCount).Validate(); err != nil {
		return nil, err
	}

	employees := make(map[string]*employeeStatus[domain.WashRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.WashRequest]{isIdle: true}
	}

	launderer := &launderer{
		clock:      clock,
		hourChange: hourChange,
		stockRepo:  stockRepo,
		stocker:    stocker,
	}

	return &LaundererService{
		employeeService: employeeService[domain.WashRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          launderer.work,
		},
	}, nil
}

func (s *LaundererService) Work(ctx context.Context, requests <-chan domain.WashRequest) {
	if s == nil {
		return
	}

	s.employeeService.Work(ctx, requests)
}

func (l *launderer) work(ctx context.Context, employeeName string, request domain.WashRequest) {
	spanCtx, span := telemetry.StartSpan(ctx, "staff.launderer.wash_request",
		telemetry.RequestAttributes(employeeName),
		attribute.String("staff.room_name", request.RoomName),
		attribute.String("staff.laundry.item", request.Item),
	)
	defer span.End()

	startTime, err := l.clock.Now(spanCtx)
	if err != nil {
		telemetry.RecordSpanError(span, err)
		slog.ErrorContext(spanCtx, "failed to get current time", "error", err)
		return
	}

	slog.InfoContext(spanCtx,
		"launderer started to workFn on washing request",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"item", request.Item,
		"startTime", startTime,
	)

	err = l.wash(spanCtx, request.Item, employeeName, request.RoomName, startTime)
	if err == nil {
		return
	}

	var stockErr *dbErrors.StockInsufficientQuantityErr
	if !errors.As(err, &stockErr) {
		telemetry.RecordSpanError(span, err)
		slog.ErrorContext(spanCtx, "failed to wash item", "item", request.Item, "employeeName", employeeName, "roomName", request.RoomName, "error", err)
		return
	}

	slog.WarnContext(spanCtx,
		"insufficient stock to wash item, requesting restock",
		"employeeName", employeeName,
		"roomName", request.RoomName,
		"itemName", stockErr.ItemName,
		"requestedQuantity", stockErr.Quantity,
	)

	if restockErr := l.stocker.ImmediateRestock(spanCtx, stockErr.ItemName); restockErr != nil {
		telemetry.RecordSpanError(span, restockErr)
		slog.ErrorContext(spanCtx, "failed to restock item", "employeeName", employeeName, "roomName", request.RoomName, "error", restockErr)
		return
	}

	if err := l.wash(spanCtx, request.Item, employeeName, request.RoomName, startTime); err != nil {
		telemetry.RecordSpanError(span, err)
		slog.ErrorContext(spanCtx, "failed to wash item after restock", "employeeName", employeeName, "roomName", request.RoomName, "error", err)
	}
}

func (l *launderer) wash(ctx context.Context, item string, employeeName string, roomName string, startTime *time.Time) error {
	spanCtx, span := telemetry.StartSpan(ctx, "staff.launderer.wash",
		telemetry.RequestAttributes(employeeName),
		attribute.String("staff.room_name", roomName),
		attribute.String("staff.laundry.item", item),
	)
	defer span.End()

	washConfig, ok := washTable[item]
	if !ok {
		err := fmt.Errorf("unsupported wash item: %s", item)
		telemetry.RecordSpanError(span, err)
		return err
	}

	err := l.stockRepo.ConsumeItem(spanCtx, documents.Soap, washConfig.soap)
	if err == nil {
		finishTime, err := l.washingCycle(spanCtx, *startTime, washConfig.cycleDurationInHours, employeeName, roomName, item)
		if err != nil {
			telemetry.RecordSpanError(span, err)
			return err
		}

		slog.InfoContext(spanCtx, "launderer is done",
			"employeeName", employeeName,
			"roomName", roomName,
			"washing", item,
			"startTime", startTime,
			"finishTime", finishTime,
		)

		return nil
	}

	telemetry.RecordSpanError(span, err)
	return err
}

func (l *launderer) washingCycle(ctx context.Context, startTime time.Time, cycleDurationInHours float64, employeeName string, roomName string, item string) (finishTime time.Time, err error) {
	spanCtx, span := telemetry.StartSpan(ctx, "staff.launderer.washing_cycle",
		telemetry.RequestAttributes(employeeName),
		attribute.String("staff.room_name", roomName),
		attribute.String("staff.laundry.item", item),
		attribute.Float64("staff.laundry.cycle_hours", cycleDurationInHours),
	)
	defer span.End()

	hourChangeCh := make(chan time.Time, 1)
	l.hourChange.Register(timeEventHourChange, hourChangeCh)
	defer l.hourChange.Unregister(timeEventHourChange, hourChangeCh)

	for {
		select {
		case <-spanCtx.Done():
			return
		case currentTime := <-hourChangeCh:
			if currentTime.Sub(startTime).Hours() >= cycleDurationInHours {
				return currentTime, nil
			}
		}
	}
}
