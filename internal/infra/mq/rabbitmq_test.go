package mq

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/mqErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"google.golang.org/protobuf/proto"
)

func TestRabbitmqConsumerBindQueue(t *testing.T) {
	t.Parallel()

	setupAndRun("bind queue edge cases", t, func(t *testing.T, consumer port.MqConsumer, _ port.MqPublisher) {
		t.Run("returns error when queue is not declared", func(t *testing.T) {
			err := consumer.BindQueue(context.Background(), config.BindingConfig{
				ExchangeName: "exchange.does.not.matter",
				RoutingKey:   "rk.test",
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("returns error when exchange name is empty", func(t *testing.T) {
			queueName := fmt.Sprintf("consumer.bind.queue.%d", time.Now().UnixNano())
			if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
				t.Fatalf("DeclareQueue() error = %v", err)
			}

			err := consumer.BindQueue(context.Background(), config.BindingConfig{
				ExchangeName: "",
				RoutingKey:   "rk.test",
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

func TestRabbitmqProducerPublish(t *testing.T) {
	t.Parallel()

	setupAndRun("publish edge cases", t, func(t *testing.T, _ port.MqConsumer, producer port.MqPublisher) {
		exchangeName := fmt.Sprintf("producer.publish.exchange.%d", time.Now().UnixNano())
		if err := producer.DeclareExchange(config.ExchangeConfig{Name: exchangeName, Kind: "direct"}); err != nil {
			t.Fatalf("DeclareExchange() error = %v", err)
		}

		t.Run("returns unexpected error when message cannot be marshaled", func(t *testing.T) {
			invalidMessage := &dto.CleaningRequest{
				RoomName: "invalid-\xff",
			}

			err := producer.Publish(context.Background(), invalidMessage, "cleaning.request")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unexpectedErr *mqErrors.UnexpectedErr
			if !errors.As(err, &unexpectedErr) {
				t.Fatalf("error type = %T, want %T", err, unexpectedErr)
			}
		})
	})
}

func TestRabbitmqConsumerConsume(t *testing.T) {
	t.Parallel()

	setupAndRun("consume flow", t, func(t *testing.T, consumer port.MqConsumer, producer port.MqPublisher) {
		exchangeName := fmt.Sprintf("consumer.consume.exchange.%d", time.Now().UnixNano())
		queueName := fmt.Sprintf("consumer.consume.queue.%d", time.Now().UnixNano())
		routingKey := "cleaning.request"

		if err := producer.DeclareExchange(config.ExchangeConfig{Name: exchangeName, Kind: "direct"}); err != nil {
			t.Fatalf("DeclareExchange() error = %v", err)
		}
		if err := consumer.DeclareQueue(context.Background(), config.QueueConfig{Name: queueName, AutoDelete: true}); err != nil {
			t.Fatalf("DeclareQueue() error = %v", err)
		}
		if err := consumer.BindQueue(context.Background(), config.BindingConfig{ExchangeName: exchangeName, RoutingKey: routingKey}); err != nil {
			t.Fatalf("BindQueue() error = %v", err)
		}

		consumeCtx, cancelConsume := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelConsume()

		deliveries, err := consumer.Consume(consumeCtx)
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}

		msg := &dto.CleaningRequest{
			RoomName: "A",
			Request:  dto.RequestType_DAILY_CLEANING,
		}
		wantBody, err := proto.Marshal(msg)
		if err != nil {
			t.Fatalf("proto.Marshal() error = %v", err)
		}

		if err := producer.Publish(context.Background(), msg, routingKey); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		select {
		case delivery, ok := <-deliveries:
			if !ok {
				t.Fatal("deliveries channel closed before receiving message")
			}
			if delivery.RoutingKey != routingKey {
				t.Fatalf("RoutingKey = %q, want %q", delivery.RoutingKey, routingKey)
			}
			if delivery.ContentType != "application/protobuf" {
				t.Fatalf("ContentType = %q, want %q", delivery.ContentType, "application/protobuf")
			}
			if string(delivery.Body) != string(wantBody) {
				t.Fatalf("Body = %s, want %s", string(delivery.Body), string(wantBody))
			}
			if err := delivery.Ack(false); err != nil {
				t.Fatalf("Ack() error = %v", err)
			}
		case <-consumeCtx.Done():
			t.Fatalf("timed out waiting for delivery: %v", consumeCtx.Err())
		}

		cancelConsume()
		select {
		case _, ok := <-deliveries:
			if ok {
				t.Fatal("expected deliveries channel to close after cancel")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for deliveries channel to close after cancel")
		}
	})
}
