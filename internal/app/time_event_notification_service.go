package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
)

type timeEventType string

const timeEventHourChange timeEventType = "hour_change"
const timeEventDayChange timeEventType = "day_change"

type timeEventRegistry interface {
	Register(eventType timeEventType, ch chan<- time.Time)
	Unregister(eventType timeEventType, ch chan<- time.Time)
}

type TimeEventNotificationService struct {
	hourChangeClient port.MqConsumer
	dayChangeClient  port.MqConsumer

	hourChangeChannelsMu sync.RWMutex
	hourChangeChannels   *domain.Set[chan<- time.Time]
	dayChangeChannelsMu  sync.RWMutex
	dayChangeChannels    *domain.Set[chan<- time.Time]
}

func NewTimeEventNotificationService(hourChangeClient port.MqConsumer, dayChangeClient port.MqConsumer) (*TimeEventNotificationService, error) {
	if err := validation.New().
		NotZeroValue("hourChangeClient", hourChangeClient).
		NotZeroValue("dayChangeClient", dayChangeClient).
		Validate(); err != nil {
		return nil, err
	}

	return &TimeEventNotificationService{
		hourChangeClient:   hourChangeClient,
		dayChangeClient:    dayChangeClient,
		hourChangeChannels: domain.NewSet[chan<- time.Time](),
		dayChangeChannels:  domain.NewSet[chan<- time.Time](),
	}, nil
}

func (s *TimeEventNotificationService) Start(ctx context.Context) {
	hourChangeDeliveries, err := s.hourChangeClient.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume hour change events", "error", err)
		return
	}

	dayChangeDeliveries, err := s.dayChangeClient.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume day change events", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case hourChange, ok := <-hourChangeDeliveries:
			if !ok {
				slog.InfoContext(ctx, "hour change consumer closed")
				return
			}

			deliveryCtx, span := telemetry.StartConsumerSpan(ctx, hourChange, "hour.change")
			currentTime, err := unmarshalTimeEvent(deliveryCtx, hourChange.Body)
			if err != nil {
				telemetry.RecordDeliveryError(span, err)
				slog.ErrorContext(deliveryCtx, "failed to unmarshal hour change event", "error", err)
				s.nackDelivery(deliveryCtx, hourChange, "hourChangeEvent")
				span.End()
				continue
			}

			s.ackDelivery(deliveryCtx, hourChange, "hourChangeEvent")
			s.notifyHourChange(deliveryCtx, currentTime)
			span.End()
		case dayChange, ok := <-dayChangeDeliveries:
			if !ok {
				slog.InfoContext(ctx, "day change consumer closed")
				return
			}

			deliveryCtx, span := telemetry.StartConsumerSpan(ctx, dayChange, "day.change")
			currentTime, err := unmarshalTimeEvent(deliveryCtx, dayChange.Body)
			if err != nil {
				telemetry.RecordDeliveryError(span, err)
				slog.ErrorContext(deliveryCtx, "failed to unmarshal day change event", "error", err)
				s.nackDelivery(deliveryCtx, dayChange, "dayChangeEvent")
				span.End()
				continue
			}

			s.ackDelivery(deliveryCtx, dayChange, "dayChangeEvent")
			s.notifyDayChange(deliveryCtx, currentTime)
			span.End()
		}
	}
}

func (s *TimeEventNotificationService) Register(eventType timeEventType, ch chan<- time.Time) {
	if ch == nil {
		return
	}

	switch eventType {
	case timeEventHourChange:
		s.hourChangeChannelsMu.Lock()
		defer s.hourChangeChannelsMu.Unlock()
		s.hourChangeChannels.Add(ch)
	case timeEventDayChange:
		s.dayChangeChannelsMu.Lock()
		defer s.dayChangeChannelsMu.Unlock()
		s.dayChangeChannels.Add(ch)
	}
}

func (s *TimeEventNotificationService) Unregister(eventType timeEventType, ch chan<- time.Time) {
	if ch == nil {
		return
	}

	switch eventType {
	case timeEventHourChange:
		s.hourChangeChannelsMu.Lock()
		defer s.hourChangeChannelsMu.Unlock()
		s.hourChangeChannels.Remove(ch)
	case timeEventDayChange:
		s.dayChangeChannelsMu.Lock()
		defer s.dayChangeChannelsMu.Unlock()
		s.dayChangeChannels.Remove(ch)
	}
}

func (s *TimeEventNotificationService) notifyHourChange(ctx context.Context, currentTime time.Time) {
	s.hourChangeChannelsMu.RLock()
	channels := s.hourChangeChannels.Values()
	s.hourChangeChannelsMu.RUnlock()

	s.notify(ctx, currentTime, channels, "hour change")
}

func (s *TimeEventNotificationService) notifyDayChange(ctx context.Context, currentTime time.Time) {
	s.dayChangeChannelsMu.RLock()
	channels := s.dayChangeChannels.Values()
	s.dayChangeChannelsMu.RUnlock()

	s.notify(ctx, currentTime, channels, "day change")
}

func (s *TimeEventNotificationService) notify(ctx context.Context, currentTime time.Time, channels []chan<- time.Time, eventName string) {
	notifyCtx, span := telemetry.StartSpan(ctx, "staff.time_event.notify",
		attribute.String("staff.time_event.name", eventName),
		attribute.Int("staff.time_event.subscriber_count", len(channels)),
	)
	defer span.End()

	for _, ch := range channels {
		select {
		case ch <- currentTime:
			slog.DebugContext(notifyCtx, "published "+eventName+" event to subscriber", "currentTime", currentTime)
		default:
			slog.WarnContext(notifyCtx, "skipped "+eventName+" event for busy subscriber", "currentTime", currentTime)
		}
	}
}

func (s *TimeEventNotificationService) ackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "failed to ack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func (s *TimeEventNotificationService) nackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Nack(false, false); err != nil {
		slog.ErrorContext(ctx, "failed to nack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}
