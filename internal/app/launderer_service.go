package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"github.com/Kenji-Uema/staffSimulator/internal/transport/grpc/clock"
	"google.golang.org/protobuf/proto"
)

const linensCycleDurationInHours = 2
const towelsCycleDurationInHours = 1
const soapUsedToWashLinens = 20
const soapUsedToWashTowels = 10

type LaundererService struct {
	EmployeeService[domain.WashRequest]
}

type Launderer struct {
	clock            clock.Clock
	hourChangeClient port.MqConsumer
	stockRepo        port.StockRepo
}

func NewLaundererService(employeeNames []string,
	clock clock.Clock, hourChangeClient port.MqConsumer, stockRepo port.StockRepo) (*LaundererService, error) {

	employeeCount := len(employeeNames)
	if employeeCount == 0 {
		return nil, fmt.Errorf("worker count must be greater than 0")
	}

	employees := make(map[string]*employeeStatus[domain.WashRequest], employeeCount)
	for _, workerName := range employeeNames {
		employees[workerName] = &employeeStatus[domain.WashRequest]{isIdle: true}
	}

	launderer := &Launderer{
		clock:            clock,
		hourChangeClient: hourChangeClient,
		stockRepo:        stockRepo,
	}

	return &LaundererService{
		EmployeeService: EmployeeService[domain.WashRequest]{
			employeeCount: employeeCount,
			employees:     employees,
			work:          launderer.work,
		},
	}, nil
}

func (l *Launderer) work(ctx context.Context, employeeName string, request domain.WashRequest) {
	startTime, err := l.clock.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get current time", "error", err)
		return
	}

	type washTask struct {
		item         string
		duration     float64
		soapQuantity int
	}

	var tasks []washTask
	if request.Linens {
		tasks = append(tasks, washTask{item: "linens", duration: linensCycleDurationInHours, soapQuantity: soapUsedToWashLinens})
	}
	if request.Towels {
		tasks = append(tasks, washTask{item: "towels", duration: towelsCycleDurationInHours, soapQuantity: soapUsedToWashTowels})
	}

	for _, task := range tasks {
		if err := l.wash(ctx, task.duration, task.soapQuantity, employeeName, request.RoomName, task.item, startTime); err != nil {
			slog.ErrorContext(ctx, "failed to wash item", "item", task.item, "error", err)
		}
	}
}

func (l *Launderer) wash(ctx context.Context, cycleDurationInHours float64, soapQuantity int,
	employeeName string, roomName string, item string, startTime *time.Time) error {

	slog.InfoContext(ctx, "launderer is working",
		"employeeName", employeeName,
		"roomName", roomName,
		"washing", item,
		"startTime", startTime,
	)

	if err := l.stockRepo.ConsumeItem(ctx, item, soapQuantity); err != nil {
		return err
	}

	finishTime, err := l.washingCycle(ctx, *startTime, cycleDurationInHours)
	if err != nil {
		slog.ErrorContext(ctx, "failed to wash", "error", err)
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

func (l *Launderer) unmarshalTimeEvent(ctx context.Context, body []byte) (time.Time, error) {
	var timeEvent dto.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		slog.WarnContext(ctx, "invalid day.changed payload", "error", err)
		return time.Time{}, err

	}

	if timeEvent.GetTime() == nil {
		slog.WarnContext(ctx, "invalid day.changed payload: missing time")
		return time.Time{}, errors.New("missing time")
	}

	tomorrow := timeEvent.GetTime().AsTime().AddDate(0, 0, 1)
	return tomorrow, nil
}

func (l *Launderer) washingCycle(ctx context.Context, startTime time.Time, cycleDurationInHours float64) (finishTime time.Time, err error) {
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

			currentTime, err := l.unmarshalTimeEvent(ctx, delivery.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal day-change event", "error", err)
				return time.Time{}, err
			}

			if currentTime.Sub(startTime).Hours() > cycleDurationInHours {
				return currentTime, nil
			}
		}
	}
}
