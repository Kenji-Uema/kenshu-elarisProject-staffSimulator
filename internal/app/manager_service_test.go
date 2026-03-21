package app

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/fakes"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

func TestNewManagerService(t *testing.T) {
	t.Parallel()

	t.Run("returns error when cleaning consumer is nil", func(t *testing.T) {
		var cleaningConsumer *fakes.FakeMqConsumer

		service, err := NewManagerService(cleaningConsumer, &fakeManagerTimeEventRegistry{}, make(chan domain.CleaningRequest), nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when day change registry is nil", func(t *testing.T) {
		var registry *fakeManagerTimeEventRegistry

		service, err := NewManagerService(&fakes.FakeMqConsumer{}, registry, make(chan domain.CleaningRequest), nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("returns error when cleaning channel is nil", func(t *testing.T) {
		service, err := NewManagerService(&fakes.FakeMqConsumer{}, &fakeManagerTimeEventRegistry{}, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if service != nil {
			t.Fatal("expected nil service on constructor error")
		}
	})

	t.Run("creates service when required dependencies are configured", func(t *testing.T) {
		service, err := NewManagerService(&fakes.FakeMqConsumer{}, &fakeManagerTimeEventRegistry{}, make(chan domain.CleaningRequest), nil)
		if err != nil {
			t.Fatalf("NewManagerService() error = %v", err)
		}
		if service == nil {
			t.Fatal("expected service to be created")
		}
	})
}

func TestManagerServiceStart(t *testing.T) {
	t.Parallel()

	t.Run("nacks invalid cleaning request without requeue", func(t *testing.T) {
		validate := func(t *testing.T, cleaningAck *fakes.FakeAcknowledger, dayRegistry *fakeManagerTimeEventRegistry, cleaningCh <-chan domain.CleaningRequest, stockerCh <-chan domain.RestockRequest) {
			t.Helper()

			if cleaningAck.NackCalls != 1 {
				t.Fatalf("NackCalls = %d, want 1", cleaningAck.NackCalls)
			}
			if cleaningAck.LastNackRequeue {
				t.Fatal("expected nack requeue to be false")
			}
			if cleaningAck.AckCalls != 0 {
				t.Fatalf("AckCalls = %d, want 0", cleaningAck.AckCalls)
			}
			if dayRegistry.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", dayRegistry.RegisterCallCount)
			}
			if dayRegistry.LastRegisteredEventType != timeEventDayChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", dayRegistry.LastRegisteredEventType, timeEventDayChange)
			}
			if dayRegistry.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", dayRegistry.UnregisterCallCount)
			}
			if dayRegistry.LastUnregisteredEventType != timeEventDayChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", dayRegistry.LastUnregisteredEventType, timeEventDayChange)
			}

			select {
			case got := <-cleaningCh:
				t.Fatalf("unexpected cleaning request: %+v", got)
			default:
			}

			select {
			case got := <-stockerCh:
				t.Fatalf("unexpected restock request: %+v", got)
			default:
			}
		}

		managerStart(t, []byte("invalid-protobuf"), nil, nil, make(chan domain.RestockRequest, 1), validate)
	})

	t.Run("does not forward day change when stocker channel is not configured", func(t *testing.T) {
		validate := func(t *testing.T, cleaningAck *fakes.FakeAcknowledger, dayRegistry *fakeManagerTimeEventRegistry, cleaningCh <-chan domain.CleaningRequest, stockerCh <-chan domain.RestockRequest) {
			t.Helper()

			if cleaningAck.NackCalls != 0 {
				t.Fatalf("cleaningAck.NackCalls = %d, want 0", cleaningAck.NackCalls)
			}
			if dayRegistry.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", dayRegistry.RegisterCallCount)
			}
			if dayRegistry.LastRegisteredEventType != timeEventDayChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", dayRegistry.LastRegisteredEventType, timeEventDayChange)
			}
			if dayRegistry.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", dayRegistry.UnregisterCallCount)
			}
			if dayRegistry.LastUnregisteredEventType != timeEventDayChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", dayRegistry.LastUnregisteredEventType, timeEventDayChange)
			}

			select {
			case got := <-cleaningCh:
				t.Fatalf("unexpected cleaning request: %+v", got)
			default:
			}

			if stockerCh != nil {
				select {
				case got := <-stockerCh:
					t.Fatalf("unexpected restock request: %+v", got)
				default:
				}
			}
		}

		managerStart(t, nil, &time.Time{}, nil, nil, validate)
	})

	t.Run("acks valid cleaning request and forwards it", func(t *testing.T) {
		validate := func(t *testing.T, cleaningAck *fakes.FakeAcknowledger, dayRegistry *fakeManagerTimeEventRegistry, cleaningCh <-chan domain.CleaningRequest, stockerCh <-chan domain.RestockRequest) {
			t.Helper()

			if cleaningAck.AckCalls != 1 {
				t.Fatalf("AckCalls = %d, want 1", cleaningAck.AckCalls)
			}
			if cleaningAck.LastAckTag != 1 {
				t.Fatalf("LastAckTag = %d, want 1", cleaningAck.LastAckTag)
			}
			if cleaningAck.LastAckMultiple {
				t.Fatal("expected ack multiple to be false")
			}
			if cleaningAck.NackCalls != 0 {
				t.Fatalf("NackCalls = %d, want 0", cleaningAck.NackCalls)
			}
			if dayRegistry.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", dayRegistry.RegisterCallCount)
			}
			if dayRegistry.LastRegisteredEventType != timeEventDayChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", dayRegistry.LastRegisteredEventType, timeEventDayChange)
			}
			if dayRegistry.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", dayRegistry.UnregisterCallCount)
			}
			if dayRegistry.LastUnregisteredEventType != timeEventDayChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", dayRegistry.LastUnregisteredEventType, timeEventDayChange)
			}

			select {
			case got := <-cleaningCh:
				want := domain.CleaningRequest{RoomName: "A", RequestType: "DAILY_CLEANING"}
				if got != want {
					t.Fatalf("cleaning request = %+v, want %+v", got, want)
				}
			default:
				t.Fatal("expected cleaning request")
			}

			select {
			case got := <-stockerCh:
				t.Fatalf("unexpected restock request: %+v", got)
			default:
			}
		}

		managerStart(t, mustMarshalManagerCleaningRequest(t, &dto.CleaningRequest{
			RoomName: "A",
			Request:  dto.RequestType_DAILY_CLEANING,
		}), nil, nil, make(chan domain.RestockRequest, 1), validate)
	})

	t.Run("forwards restock request on day change event", func(t *testing.T) {
		validate := func(t *testing.T, cleaningAck *fakes.FakeAcknowledger, dayRegistry *fakeManagerTimeEventRegistry, cleaningCh <-chan domain.CleaningRequest, stockerCh <-chan domain.RestockRequest) {
			t.Helper()

			if cleaningAck.AckCalls != 0 {
				t.Fatalf("cleaningAck.AckCalls = %d, want 0", cleaningAck.AckCalls)
			}
			if cleaningAck.NackCalls != 0 {
				t.Fatalf("cleaningAck.NackCalls = %d, want 0", cleaningAck.NackCalls)
			}
			if dayRegistry.RegisterCallCount != 1 {
				t.Fatalf("RegisterCallCount = %d, want 1", dayRegistry.RegisterCallCount)
			}
			if dayRegistry.LastRegisteredEventType != timeEventDayChange {
				t.Fatalf("LastRegisteredEventType = %q, want %q", dayRegistry.LastRegisteredEventType, timeEventDayChange)
			}
			if dayRegistry.UnregisterCallCount != 1 {
				t.Fatalf("UnregisterCallCount = %d, want 1", dayRegistry.UnregisterCallCount)
			}
			if dayRegistry.LastUnregisteredEventType != timeEventDayChange {
				t.Fatalf("LastUnregisteredEventType = %q, want %q", dayRegistry.LastUnregisteredEventType, timeEventDayChange)
			}

			select {
			case got := <-stockerCh:
				if len(got.ItemsName) == 0 {
					t.Fatal("expected at least one item to restock")
				}

				seen := make(map[string]struct{}, len(got.ItemsName))
				for _, item := range got.ItemsName {
					if !slices.Contains(documents.ManagedStockItemNames, item) {
						t.Fatalf("restock item %q is not managed", item)
					}
					if _, ok := seen[item]; ok {
						t.Fatalf("restock item %q was duplicated", item)
					}
					seen[item] = struct{}{}
				}
			default:
				t.Fatal("expected restock request")
			}

			select {
			case got := <-cleaningCh:
				t.Fatalf("unexpected cleaning request: %+v", got)
			default:
			}
		}

		now := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
		managerStart(t, nil, &now, nil, make(chan domain.RestockRequest, 1), validate)
	})
}

func TestManagerServiceSelectItemsToRestock(t *testing.T) {
	t.Parallel()

	manager, err := NewManagerService(&fakes.FakeMqConsumer{}, &fakeManagerTimeEventRegistry{}, make(chan domain.CleaningRequest), nil)
	if err != nil {
		t.Fatalf("NewManagerService() error = %v", err)
	}

	selectedItems := manager.selectItemsToRestock()
	if len(selectedItems) == 0 {
		t.Fatal("expected at least one item to be selected")
	}
	if len(selectedItems) > len(documents.ManagedStockItemNames) {
		t.Fatalf("selected item count = %d, want <= %d", len(selectedItems), len(documents.ManagedStockItemNames))
	}

	seen := make(map[string]struct{}, len(selectedItems))
	for _, item := range selectedItems {
		if !slices.Contains(documents.ManagedStockItemNames, item) {
			t.Fatalf("selected item %q is not managed", item)
		}
		if _, ok := seen[item]; ok {
			t.Fatalf("selected item %q was duplicated", item)
		}
		seen[item] = struct{}{}
	}
}

func mustMarshalManagerCleaningRequest(t *testing.T, request *dto.CleaningRequest) []byte {
	t.Helper()

	body, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return body
}

func managerStart(t *testing.T, cleaningBody []byte, dayChangeTime *time.Time, cleaningCh chan domain.CleaningRequest, stockerCh chan domain.RestockRequest,
	validate func(t *testing.T, cleaningAck *fakes.FakeAcknowledger, dayRegistry *fakeManagerTimeEventRegistry, cleaningCh <-chan domain.CleaningRequest, stockerCh <-chan domain.RestockRequest)) {
	t.Helper()

	cleaningAck := &fakes.FakeAcknowledger{}

	if cleaningCh == nil {
		cleaningCh = make(chan domain.CleaningRequest, 1)
	}

	cleaningConsumer := &fakes.FakeMqConsumer{
		ConsumeFn: func(ctx context.Context) (<-chan amqp.Delivery, error) {
			if cleaningBody == nil {
				return nil, nil
			}

			ch := make(chan amqp.Delivery, 1)
			ch <- amqp.Delivery{
				Body:         cleaningBody,
				Acknowledger: cleaningAck,
				DeliveryTag:  1,
				RoutingKey:   "cleaning.request",
			}

			return ch, nil
		},
	}

	dayRegistry := &fakeManagerTimeEventRegistry{}
	manager, err := NewManagerService(cleaningConsumer, dayRegistry, cleaningCh, stockerCh)
	if err != nil {
		t.Fatalf("NewManagerService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		manager.Start(ctx)
		close(done)
	}()

	if dayChangeTime != nil {
		waitFor(t, time.Second, func() bool {
			return dayRegistry.LastRegisteredCh != nil
		})
		dayRegistry.LastRegisteredCh <- *dayChangeTime
	}

	waitFor(t, time.Second, func() bool {
		if cleaningAck.AckCalls > 0 || cleaningAck.NackCalls > 0 {
			return true
		}
		if dayChangeTime == nil || stockerCh == nil {
			return dayRegistry.RegisterCallCount > 0
		}
		return len(stockerCh) > 0
	})

	cancel()
	waitForDone(t, done)

	validate(t, cleaningAck, dayRegistry, cleaningCh, stockerCh)
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
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

func waitForDone(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("manager did not return after cancellation")
	}
}

type fakeManagerTimeEventRegistry struct {
	RegisterCallCount         int
	UnregisterCallCount       int
	LastRegisteredEventType   timeEventType
	LastUnregisteredEventType timeEventType
	LastRegisteredCh          chan<- time.Time
	LastUnregisteredCh        chan<- time.Time
}

func (f *fakeManagerTimeEventRegistry) Register(eventType timeEventType, ch chan<- time.Time) {
	f.RegisterCallCount++
	f.LastRegisteredEventType = eventType
	f.LastRegisteredCh = ch
}

func (f *fakeManagerTimeEventRegistry) Unregister(eventType timeEventType, ch chan<- time.Time) {
	f.UnregisterCallCount++
	f.LastUnregisteredEventType = eventType
	f.LastUnregisteredCh = ch
}
