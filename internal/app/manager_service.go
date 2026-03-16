package app

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"google.golang.org/protobuf/proto"
)

type ManagerService struct {
	cleaningEventConsumer port.MqConsumer
	timeEventConsumer     port.MqConsumer
	cleaningCh            chan<- domain.CleaningRequest
	stockerCh             chan<- domain.RestockRequest
}

var dailyRestockItems = append([]string(nil), managedStockItems...)

func NewManagerService(
	cleaningEventConsumer port.MqConsumer,
	timeEventConsumer port.MqConsumer,
	cleaningCh chan<- domain.CleaningRequest,
	stockerCh chan<- domain.RestockRequest,
) *ManagerService {
	return &ManagerService{
		cleaningEventConsumer: cleaningEventConsumer,
		timeEventConsumer:     timeEventConsumer,
		cleaningCh:            cleaningCh,
		stockerCh:             stockerCh,
	}
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

			cleaningRequest, err := s.unmarshal(ctx, cleaningDelivery.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal cleaning request", "error", err)
				continue
			}

			s.cleaningCh <- cleaningRequest

			if err := cleaningDelivery.Ack(false); err != nil {
				slog.ErrorContext(ctx, "failed to ack cleaningRequest", "error", err, "routingKey", cleaningDelivery.RoutingKey)
			}
		case dayChangeEvent, ok := <-dayChangeEvents:
			if !ok {
				slog.Info("time event consumer closed")
				return
			}

			if s.stockerCh == nil {
				slog.WarnContext(ctx, "stocker channel is not configured; skipping daily restock request")
				continue
			}

			s.stockerCh <- domain.RestockRequest{ItemsName: dailyRestockItems}

			if err := dayChangeEvent.Ack(false); err != nil {
				slog.ErrorContext(ctx, "failed to ack dayChangeEvent", "error", err, "routingKey", dayChangeEvent.RoutingKey)
			}
		}
	}
}

func (s *ManagerService) unmarshal(ctx context.Context, body []byte) (domain.CleaningRequest, error) {
	var cleaningRequest dto.CleaningRequest
	if err := proto.Unmarshal(body, &cleaningRequest); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal cleaning request", "error", err)
		return domain.CleaningRequest{}, err
	}

	if err := validation.New().
		NotBlank("roomName", cleaningRequest.GetRoomName()).
		NotBlank("request", cleaningRequest.GetRequest().String()).Validate(); err != nil {

		return domain.CleaningRequest{}, err
	}

	return domain.CleaningRequest{
		RoomName: cleaningRequest.GetRoomName(),
		R:        cleaningRequest.GetRequest().String(),
	}, nil
}
