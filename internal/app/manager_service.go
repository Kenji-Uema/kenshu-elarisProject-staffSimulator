package app

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

type ManagerService struct {
	cleaningEventConsumer port.MqConsumer
	timeEventRegistry     timeEventRegistry
	cleaningCh            chan<- domain.CleaningRequest
	stockerCh             chan<- domain.RestockRequest
}

func NewManagerService(
	cleaningEventConsumer port.MqConsumer,
	timeEventRegistry timeEventRegistry,
	cleaningCh chan<- domain.CleaningRequest,
	stockerCh chan<- domain.RestockRequest) (*ManagerService, error) {

	if err := validation.New().
		NotZeroValue("cleaningEventConsumer", cleaningEventConsumer).
		NotZeroValue("timeEventRegistry", timeEventRegistry).
		NotZeroValue("cleaningCh", cleaningCh).
		Validate(); err != nil {
		return nil, err
	}

	return &ManagerService{
		cleaningEventConsumer: cleaningEventConsumer,
		timeEventRegistry:     timeEventRegistry,
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

	dayChangeCh := make(chan time.Time, 1)
	s.timeEventRegistry.Register(timeEventDayChange, dayChangeCh)
	defer s.timeEventRegistry.Unregister(timeEventDayChange, dayChangeCh)

	for {
		select {
		case <-ctx.Done():
			return
		case cleaningDelivery, ok := <-cleaningRequests:
			if !ok {
				slog.InfoContext(ctx, "cleaning event consumer closed")
				return
			}

			deliveryCtx, span := telemetry.StartConsumerSpan(ctx, cleaningDelivery, "cleaning.request")
			cleaningRequest, err := UnmarshalCleaningRequest(deliveryCtx, cleaningDelivery.Body)
			if err != nil {
				telemetry.RecordDeliveryError(span, err)
				slog.ErrorContext(deliveryCtx, "failed to unmarshalCleaningRequest cleaning request", "error", err)
				s.nackDelivery(deliveryCtx, cleaningDelivery, "cleaningRequest")
				span.End()
				continue
			}
			_, dispatchSpan := telemetry.StartSpan(deliveryCtx, "staff.manager.dispatch_cleaning_request",
				attribute.String("staff.room_name", cleaningRequest.RoomName),
				attribute.String("staff.cleaning.request_type", cleaningRequest.RequestType),
			)
			s.cleaningCh <- cleaningRequest
			s.ackDelivery(deliveryCtx, cleaningDelivery, "cleaningRequest")
			dispatchSpan.End()
			span.End()

		case <-dayChangeCh:
			if s.stockerCh == nil {
				slog.ErrorContext(ctx, "stocker channel is not configured")
				continue
			}

			items := s.selectItemsToRestock()
			dispatchCtx, dispatchSpan := telemetry.StartSpan(ctx, "staff.manager.dispatch_restock_request",
				attribute.Int("staff.restock.item_count", len(items)),
			)
			s.stockerCh <- domain.RestockRequest{ItemsName: items}
			slog.DebugContext(dispatchCtx, "manager dispatched restock request", "itemsName", items)
			dispatchSpan.End()
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
