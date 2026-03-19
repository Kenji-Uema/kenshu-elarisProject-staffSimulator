package app

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
)

type employeeStatus[R domain.Request] struct {
	isIdle          bool
	currentRequest  R
	handledRequests int
}

type employeeService[R domain.Request] struct {
	employeeCount int
	employees     map[string]*employeeStatus[R]
	employeeMu    sync.RWMutex
	work          func(ctx context.Context, employeeName string, request R)
	workFn        func(ctx context.Context, employeeName string, request R)
}

func (s *employeeService[R]) Work(ctx context.Context, requests <-chan R) {
	var wg sync.WaitGroup

	slog.InfoContext(ctx, "employee service started", "employeeCount", s.employeeCount)

	for name := range s.employees {
		wg.Add(1)

		go func(employeeName string) {
			defer wg.Done()

			slog.DebugContext(ctx, "employee worker started", "employeeName", employeeName)

			for {
				select {
				case <-ctx.Done():
					slog.InfoContext(ctx, "employee worker stopping", "employeeName", employeeName, "reason", "context canceled")
					return
				case request, ok := <-requests:
					if !ok {
						slog.InfoContext(ctx, "employee worker stopping", "employeeName", employeeName, "reason", "request channel closed")
						return
					}

					handledRequests := s.setEmployeeWorkingStatus(employeeName, request)
					slog.InfoContext(ctx,
						"employee started request",
						"employeeName", employeeName,
						"handledRequests", handledRequests,
						"request", request,
					)

					s.work(ctx, employeeName, request)

					s.setEmployeeIdleStatus(employeeName)
					slog.InfoContext(ctx,
						"employee finished request",
						"employeeName", employeeName,
						"handledRequests", handledRequests,
						"request", request,
					)
				}
			}
		}(name)
	}

	wg.Wait()
	slog.InfoContext(ctx, "employee service stopped", "employeeCount", s.employeeCount)
}

func (s *employeeService[R]) setEmployeeWorkingStatus(employeeName string, request R) int {
	s.employeeMu.Lock()
	defer s.employeeMu.Unlock()

	status, ok := s.employees[employeeName]
	if !ok {
		return 0
	}

	status.isIdle = false
	status.handledRequests++
	status.currentRequest = request

	return status.handledRequests
}

func (s *employeeService[R]) setEmployeeIdleStatus(employeeName string) {
	s.employeeMu.Lock()
	defer s.employeeMu.Unlock()

	status, ok := s.employees[employeeName]
	if !ok {
		return
	}

	var emptyRequest R
	status.isIdle = true
	status.currentRequest = emptyRequest
}
