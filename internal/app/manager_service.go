package app

import (
	"context"
	"log/slog"
	"math/rand/v2"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ManagerService struct {
	cleaningEventConsumer port.MqConsumer
	timeEventConsumer     port.MqConsumer
	cleaningCh            chan<- domain.CleaningRequest
	stockerCh             chan<- domain.RestockRequest
}

func NewManagerService(
	cleaningEventConsumer port.MqConsumer,
	timeEventConsumer port.MqConsumer,
	cleaningCh chan<- domain.CleaningRequest,
	stockerCh chan<- domain.RestockRequest) (*ManagerService, error) {

	if err := validation.New().
		NotZeroValue("cleaningEventConsumer", cleaningEventConsumer).
		NotZeroValue("timeEventConsumer", timeEventConsumer).
		NotZeroValue("cleaningCh", cleaningCh).
		Validate(); err != nil {
		return nil, err
	}

	return &ManagerService{
		cleaningEventConsumer: cleaningEventConsumer,
		timeEventConsumer:     timeEventConsumer,
		cleaningCh:            cleaningCh,
		stockerCh:             stockerCh,
	}, nil
}

func (s *ManagerService) Start(ctx context.Context) {
	cleaningRequests, err := s.cleaningEventConsumer.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume cleaning cleaningRequests", "error", err)
		return
	}

	dayChangeEvents, err := s.timeEventConsumer.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume time events", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case cleaningDelivery, ok := <-cleaningRequests:
			if !ok {
				slog.Info("cleaning event consumer closed")
				return
			}

			cleaningRequest, err := UnmarshalCleaningRequest(ctx, cleaningDelivery.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshalCleaningRequest cleaning request", "error", err)
				s.nackDelivery(ctx, cleaningDelivery, "cleaningRequest")
				continue
			}
			s.cleaningCh <- cleaningRequest
			s.ackDelivery(ctx, cleaningDelivery, "cleaningRequest")

		case dayChangeEvent, ok := <-dayChangeEvents:
			if !ok {
				slog.Info("time event consumer closed")
				return
			}

			if s.stockerCh == nil {
				slog.ErrorContext(ctx, "stocker channel is not configured")
				s.nackDelivery(ctx, dayChangeEvent, "dayChangeEvent")
				continue
			}

			s.stockerCh <- domain.RestockRequest{ItemsName: s.selectItemsToRestock()}
			s.ackDelivery(ctx, dayChangeEvent, "dayChangeEvent")
		}
	}
}

func (s *ManagerService) selectItemsToRestock() []string {
	items := append([]string(nil), documents.ManagedStockItemNames...)
	rand.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})

	selectedCount := rand.IntN(len(items)) + 1
	return append([]string(nil), items[:selectedCount]...)
}

func (s *ManagerService) ackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "failed to ack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func (s *ManagerService) nackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Nack(false, false); err != nil {
		slog.ErrorContext(ctx, "failed to nack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}
