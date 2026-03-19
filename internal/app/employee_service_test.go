package app

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
)

func TestEmployeeServiceRun(t *testing.T) {
	t.Parallel()

	t.Run("stops when context is cancelled", func(t *testing.T) {
		service := newTestEmployeeService([]string{"Alice", "Bob"})
		requests := make(chan domain.CleaningRequest)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Work the service and use done to observe when it exits.
		done := make(chan struct{})
		go func() {
			service.Work(ctx, requests)
			close(done)
		}()

		// Verify Work stays blocked until the context is cancelled.
		select {
		case <-done:
			t.Fatal("Work returned before cancellation")
		case <-time.After(50 * time.Millisecond):
		}

		// Cancel the service context to trigger shutdown.
		cancel()

		// After cancellation, Work should return and close done.
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Work did not return after cancellation")
		}
	})

	t.Run("marks workers busy and idle around handled request", func(t *testing.T) {
		service := newTestEmployeeService([]string{"Alice", "Bob"})

		started := make(chan string, 2)
		release := make(chan struct{})
		service.work = func(ctx context.Context, employeeName string, request domain.CleaningRequest) {
			// Signal that this worker started, then block so the test can
			// inspect the in-progress worker state.
			started <- employeeName
			<-release
		}

		requests := make(chan domain.CleaningRequest, 2)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			service.Work(ctx, requests)
			close(done)
		}()

		// Queue two requests so both workers become busy.
		requests <- domain.CleaningRequest{RoomName: "A", RequestType: "DAILY_CLEANING"}
		requests <- domain.CleaningRequest{RoomName: "B", RequestType: "FULL_CLEANING"}

		// Wait until both workers have started processing requests.
		workerOne := <-started
		workerTwo := <-started

		// While the workers are blocked in work, both should be marked busy.
		if idleCount := service.IdleEmployeeCount(); idleCount != 0 {
			t.Fatalf("IdleEmployeeCount() = %d, want 0", idleCount)
		}

		metrics := service.EmployeeMetrics()
		if metrics[workerOne].IsIdle {
			t.Fatalf("%s should be busy", workerOne)
		}
		if metrics[workerTwo].IsIdle {
			t.Fatalf("%s should be busy", workerTwo)
		}
		if metrics[workerOne].HandledRequests != 1 {
			t.Fatalf("%s handled requests = %d, want 1", workerOne, metrics[workerOne].HandledRequests)
		}
		if metrics[workerTwo].HandledRequests != 1 {
			t.Fatalf("%s handled requests = %d, want 1", workerTwo, metrics[workerTwo].HandledRequests)
		}

		// Let the workers finish, then cancel the service loop.
		close(release)
		cancel()

		// Work should return once both workers exit.
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Work did not return after worker release")
		}

		// After finishing the request, both workers should be idle again.
		finalMetrics := service.EmployeeMetrics()
		if !finalMetrics[workerOne].IsIdle {
			t.Fatalf("%s should be idle after finishing work", workerOne)
		}
		if !finalMetrics[workerTwo].IsIdle {
			t.Fatalf("%s should be idle after finishing work", workerTwo)
		}
	})

	t.Run("stops when request channel closes", func(t *testing.T) {
		service := newTestEmployeeService([]string{"alice"})
		requests := make(chan domain.CleaningRequest)

		done := make(chan struct{})
		go func() {
			service.Work(context.Background(), requests)
			close(done)
		}()

		// Closing the request channel should make Work exit without context cancellation.
		close(requests)

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Work did not return after request channel closed")
		}
	})
}

func newTestEmployeeService(employeeNames []string) *employeeService[domain.CleaningRequest] {
	employees := make(map[string]*employeeStatus[domain.CleaningRequest], len(employeeNames))
	for _, name := range employeeNames {
		employees[name] = &employeeStatus[domain.CleaningRequest]{isIdle: true}
	}

	return &employeeService[domain.CleaningRequest]{
		employeeCount: len(employeeNames),
		employees:     employees,
		work:          func(context.Context, string, domain.CleaningRequest) {},
	}
}
