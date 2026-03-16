package app

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/transport/grpc/clock"
)

func TestHousekeeperServiceRunStopsOnContextCancellation(t *testing.T) {
	t.Parallel()

	service, err := NewHousekeeperService([]string{"alice", "bob"}, clock.Clock{}, nil, nil)
	if err != nil {
		t.Fatalf("NewHousekeeperService() error = %v", err)
	}

	cleaningRequests := make(chan domain.CleaningRequest)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})

	go func() {
		service.Run(ctx, cleaningRequests)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Run returned before shutdown")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestHousekeeperServiceMetrics(t *testing.T) {
	t.Parallel()

	service, err := NewHousekeeperService([]string{"alice", "bob"}, clock.Clock{}, nil, nil)
	if err != nil {
		t.Fatalf("NewHousekeeperService() error = %v", err)
	}

	started := make(chan string, 2)
	release := make(chan struct{})
	service.work = func(ctx context.Context, workerName string, cleaningRequest domain.CleaningRequest) {
		started <- workerName
		<-release
	}

	cleaningRequests := make(chan domain.CleaningRequest, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		service.Run(ctx, cleaningRequests)
		close(done)
	}()

	cleaningRequests <- domain.CleaningRequest{RoomName: "A"}
	cleaningRequests <- domain.CleaningRequest{RoomName: "B"}

	workerOne := <-started
	workerTwo := <-started

	idleCount := service.IdleEmployeeCount()
	if idleCount != 0 {
		t.Fatalf("IdleEmployeeCount() = %d, want 0", idleCount)
	}

	metrics := service.EmployeeMetrics()
	if metrics[workerOne].HandledRequests != 1 {
		t.Fatalf("%s handled requests = %d, want 1", workerOne, metrics[workerOne].HandledRequests)
	}

	if metrics[workerTwo].HandledRequests != 1 {
		t.Fatalf("%s handled requests = %d, want 1", workerTwo, metrics[workerTwo].HandledRequests)
	}

	if metrics[workerOne].IsIdle {
		t.Fatalf("%s should be busy", workerOne)
	}

	if metrics[workerTwo].IsIdle {
		t.Fatalf("%s should be busy", workerTwo)
	}

	close(release)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after metrics test shutdown")
	}

	finalMetrics := service.EmployeeMetrics()
	if finalMetrics[workerOne].HandledRequests != 1 {
		t.Fatalf("%s final handled requests = %d, want 1", workerOne, finalMetrics[workerOne].HandledRequests)
	}

	if finalMetrics[workerTwo].HandledRequests != 1 {
		t.Fatalf("%s final handled requests = %d, want 1", workerTwo, finalMetrics[workerTwo].HandledRequests)
	}
}
