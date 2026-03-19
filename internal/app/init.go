package app

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/infra"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
)

var channels = struct {
	cleaning  chan domain.CleaningRequest
	launderer chan domain.WashRequest
	stocker   chan domain.RestockRequest
}{
	cleaning:  make(chan domain.CleaningRequest, 32),
	launderer: make(chan domain.WashRequest, 32),
	stocker:   make(chan domain.RestockRequest, 16),
}

type Services struct {
	HousekeeperService *HousekeeperService
	LaundererService   *LaundererService
	StockerService     *StockerService
	ManagerService     *ManagerService
}

func NewServices(configs config.AppConfig, mongo infra.Mongo, rabbitmq infra.Rabbitmq, clock port.Clock) (Services, error) {
	stockerService, err := NewStockerService(configs.Employees.Stockers, mongo.StockRepo)
	if err != nil {
		return Services{}, err
	}

	housekeeperService, err := NewHousekeeperService(
		configs.Employees.Housekeepers,
		clock,
		mongo.CottageRepo,
		mongo.StockRepo,
		stockerService,
		channels.launderer,
	)
	if err != nil {
		return Services{}, err
	}

	laundererService, err := NewLaundererService(
		configs.Employees.Launderers,
		clock,
		rabbitmq.HourChangeConsumer,
		mongo.StockRepo,
		stockerService,
	)
	if err != nil {
		return Services{}, err
	}

	managerService, err := NewManagerService(rabbitmq.CleaningConsumer, rabbitmq.DayChangeConsumer, channels.cleaning, channels.stocker)
	if err != nil {
		return Services{}, err
	}
	return Services{
		HousekeeperService: housekeeperService,
		LaundererService:   laundererService,
		StockerService:     stockerService,
		ManagerService:     managerService,
	}, nil
}

func (s Services) Start(ctx context.Context) {
	go s.HousekeeperService.Work(ctx, channels.cleaning)
	go s.LaundererService.Work(ctx, channels.launderer)
	go s.StockerService.Work(ctx, channels.stocker)
	s.ManagerService.Start(ctx)
}
