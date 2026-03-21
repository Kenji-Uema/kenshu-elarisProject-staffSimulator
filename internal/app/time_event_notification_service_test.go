package app

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/fakes"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewTimeEventNotificationService(t *testing.T) {
	t.Parallel()

	t.Run("returns error when hour change consumer is nil", func(t *testing.T) {
		var hourConsumer *fakes.FakeMqConsumer

		service, err := NewTimeEventNotificationService(hourConsumer, &fakes.FakeMqConsumer{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when day change consumer is nil", func(t *testing.T) {
		var dayConsumer *fakes.FakeMqConsumer

		service, err := NewTimeEventNotificationService(&fakes.FakeMqConsumer{}, dayConsumer)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("creates service with configured dependencies", func(t *testing.T) {
		service, err := NewTimeEventNotificationService(&fakes.FakeMqConsumer{}, &fakes.FakeMqConsumer{})
		if err != nil {
			t.Fatalf("NewTimeEventNotificationService() error = %v", err)
		}
		if service == nil {
			t.Fatal("expected service to be created")
		}
	})
}

func TestTimeEventNotificationServiceStart(t *testing.T) {
	t.Parallel()

	t.Run("publishes valid hour change events to registered channels and acks", func(t *testing.T) {
		eventTime := time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		hourConsumer := &fakes.FakeMqConsumer{}
		hourConsumer.ConsumeFn = func(ctx context.Context) (<-chan amqp.Delivery, error) {
			ch := make(chan amqp.Delivery, 1)
			ack := &fakes.FakeAcknowledger{}
			hourConsumer.LastAck = ack
			ch <- amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  1,
				Body:         mustMarshalTimeEvent(t, eventTime),
			}
			close(ch)
			return ch, nil
		}
		dayConsumer := &fakes.FakeMqConsumer{
			ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
				ch := make(chan amqp.Delivery)
				go func() {
					<-ctx.Done()
					close(ch)
				}()
				return ch, nil
			},
		}

		service, err := NewTimeEventNotificationService(hourConsumer, dayConsumer)
		if err != nil {
			t.Fatalf("NewTimeEventNotificationService() error = %v", err)
		}

		firstCh := make(chan time.Time, 1)
		secondCh := make(chan time.Time, 1)
		service.Register(timeEventHourChange, firstCh)
		service.Register(timeEventHourChange, secondCh)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			service.Start(ctx)
			close(done)
		}()

		assertPublishedTime(t, firstCh, eventTime)
		assertPublishedTime(t, secondCh, eventTime)

		cancel()
		<-done

		if hourConsumer.ConsumeCallCount != 1 {
			t.Fatalf("ConsumeCallCount = %d, want 1", hourConsumer.ConsumeCallCount)
		}
		if hourConsumer.LastAck == nil {
			t.Fatal("expected acknowledger to be captured")
		}
		if hourConsumer.LastAck.AckCalls != 1 {
			t.Fatalf("AckCalls = %d, want 1", hourConsumer.LastAck.AckCalls)
		}
		if hourConsumer.LastAck.NackCalls != 0 {
			t.Fatalf("NackCalls = %d, want 0", hourConsumer.LastAck.NackCalls)
		}
	})

	t.Run("publishes valid day change events to registered channels and acks", func(t *testing.T) {
		eventTime := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
		hourConsumer := &fakes.FakeMqConsumer{
			ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
				ch := make(chan amqp.Delivery)
				go func() {
					<-ctx.Done()
					close(ch)
				}()
				return ch, nil
			},
		}
		dayConsumer := &fakes.FakeMqConsumer{}
		dayConsumer.ConsumeFn = func(ctx context.Context) (<-chan amqp.Delivery, error) {
			ch := make(chan amqp.Delivery, 1)
			ack := &fakes.FakeAcknowledger{}
			dayConsumer.LastAck = ack
			ch <- amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  1,
				Body:         mustMarshalTimeEvent(t, eventTime),
			}
			close(ch)
			return ch, nil
		}

		service, err := NewTimeEventNotificationService(hourConsumer, dayConsumer)
		if err != nil {
			t.Fatalf("NewTimeEventNotificationService() error = %v", err)
		}

		dayCh := make(chan time.Time, 1)
		service.Register(timeEventDayChange, dayCh)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			service.Start(ctx)
			close(done)
		}()

		assertPublishedTime(t, dayCh, eventTime)

		cancel()
		<-done

		if dayConsumer.ConsumeCallCount != 1 {
			t.Fatalf("ConsumeCallCount = %d, want 1", dayConsumer.ConsumeCallCount)
		}
		if dayConsumer.LastAck == nil {
			t.Fatal("expected acknowledger to be captured")
		}
		if dayConsumer.LastAck.AckCalls != 1 {
			t.Fatalf("AckCalls = %d, want 1", dayConsumer.LastAck.AckCalls)
		}
		if dayConsumer.LastAck.NackCalls != 0 {
			t.Fatalf("NackCalls = %d, want 0", dayConsumer.LastAck.NackCalls)
		}
	})

	t.Run("nacks invalid day change events", func(t *testing.T) {
		hourConsumer := &fakes.FakeMqConsumer{
			ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
				ch := make(chan amqp.Delivery)
				go func() {
					<-ctx.Done()
					close(ch)
				}()
				return ch, nil
			},
		}
		dayConsumer := &fakes.FakeMqConsumer{}
		dayConsumer.ConsumeFn = func(ctx context.Context) (<-chan amqp.Delivery, error) {
			ch := make(chan amqp.Delivery, 1)
			ack := &fakes.FakeAcknowledger{}
			dayConsumer.LastAck = ack
			ch <- amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  1,
				Body:         []byte("not-protobuf"),
			}
			close(ch)
			return ch, nil
		}

		service, err := NewTimeEventNotificationService(hourConsumer, dayConsumer)
		if err != nil {
			t.Fatalf("NewTimeEventNotificationService() error = %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			service.Start(ctx)
			close(done)
		}()

		waitForAckState(t, time.Second, func() bool {
			return dayConsumer.LastAck != nil && dayConsumer.LastAck.NackCalls == 1
		})

		cancel()
		<-done

		if dayConsumer.LastAck == nil {
			t.Fatal("expected acknowledger to be captured")
		}
		if dayConsumer.LastAck.AckCalls != 0 {
			t.Fatalf("AckCalls = %d, want 0", dayConsumer.LastAck.AckCalls)
		}
		if dayConsumer.LastAck.NackCalls != 1 {
			t.Fatalf("NackCalls = %d, want 1", dayConsumer.LastAck.NackCalls)
		}
		if dayConsumer.LastAck.LastNackRequeue {
			t.Fatal("expected Nack requeue to be false")
		}
	})

	t.Run("stops publishing day change events to channel after unregister", func(t *testing.T) {
		firstEvent := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
		secondEvent := firstEvent.Add(24 * time.Hour)

		hourConsumer := &fakes.FakeMqConsumer{
			ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
				ch := make(chan amqp.Delivery)
				go func() {
					<-ctx.Done()
					close(ch)
				}()
				return ch, nil
			},
		}
		dayDeliveries := make(chan amqp.Delivery, 2)
		dayConsumer := &fakes.FakeMqConsumer{
			ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
				return dayDeliveries, nil
			},
		}

		service, err := NewTimeEventNotificationService(hourConsumer, dayConsumer)
		if err != nil {
			t.Fatalf("NewTimeEventNotificationService() error = %v", err)
		}

		dayCh := make(chan time.Time, 2)
		service.Register(timeEventDayChange, dayCh)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			service.Start(ctx)
			close(done)
		}()

		dayDeliveries <- amqp.Delivery{
			Acknowledger: &fakes.FakeAcknowledger{},
			DeliveryTag:  1,
			Body:         mustMarshalTimeEvent(t, firstEvent),
		}

		assertPublishedTime(t, dayCh, firstEvent)
		service.Unregister(timeEventDayChange, dayCh)

		secondAck := &fakes.FakeAcknowledger{}
		dayDeliveries <- amqp.Delivery{
			Acknowledger: secondAck,
			DeliveryTag:  2,
			Body:         mustMarshalTimeEvent(t, secondEvent),
		}

		select {
		case got := <-dayCh:
			t.Fatalf("received unexpected event after unregister: %v", got)
		case <-time.After(50 * time.Millisecond):
		}

		if secondAck.AckCalls != 1 {
			t.Fatalf("AckCalls for second event = %d, want 1", secondAck.AckCalls)
		}

		cancel()
		close(dayDeliveries)
		<-done
	})
}

func assertPublishedTime(t *testing.T, ch <-chan time.Time, want time.Time) {
	t.Helper()

	select {
	case got := <-ch:
		if !got.Equal(want) {
			t.Fatalf("published time = %v, want %v", got, want)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for published time")
	}
}

func waitForAckState(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition not met before timeout")
}

func mustMarshalTimeEvent(t *testing.T, eventTime time.Time) []byte {
	t.Helper()

	payload, err := proto.Marshal(&dto.TimeEvent{
		Time: timestamppb.New(eventTime),
	})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return payload
}
