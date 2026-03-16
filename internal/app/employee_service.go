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

type EmployeeMetrics[R domain.Request] struct {
	IsIdle          bool
	CurrentRequest  R
	HandledRequests int
}

type EmployeeService[R domain.Request] struct {
	employeeCount int
	employees     map[string]*employeeStatus[R]
	employeeMu    sync.RWMutex
	work          func(ctx context.Context, employeeName string, request R)
}

func (s *EmployeeService[R]) Run(ctx context.Context, requests <-chan R) {
	var wg sync.WaitGroup

	for name := range s.employees {
		wg.Add(1)

		go func(employeeName string) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					slog.Info("employee shutting down", "employeeName", employeeName)
					return
				case request, ok := <-requests:
					if !ok {
						return
					}

					s.setEmployeeStatus(employeeName, false, request, true)

					s.handleWork(ctx, employeeName, request)

					var zero R
					s.setEmployeeStatus(employeeName, true, zero, false)
				}
			}
		}(name)
	}

	wg.Wait()
}

func (s *EmployeeService[R]) handleWork(ctx context.Context, employeeName string, request R) {
	if s.work != nil {
		s.work(ctx, employeeName, request)
		return
	}

	slog.Info(
		"cleaning request received",
		"workerName", employeeName,
		"request", request,
	)
}

func (s *EmployeeService[R]) setEmployeeStatus(employeeName string, isIdle bool, request R, incrementHandledRequests bool) {
	s.employeeMu.Lock()
	defer s.employeeMu.Unlock()

	status, ok := s.employees[employeeName]
	if !ok {
		return
	}

	status.isIdle = isIdle
	status.currentRequest = request
	if incrementHandledRequests {
		status.handledRequests++
	}
}

func (s *EmployeeService[R]) IdleEmployeeCount() int {
	s.employeeMu.Lock()
	defer s.employeeMu.Unlock()

	idleEmployees := 0
	for _, status := range s.employees {
		if status.isIdle {
			idleEmployees++
		}
	}

	return idleEmployees
}

func (s *EmployeeService[R]) EmployeeMetrics() map[string]EmployeeMetrics[R] {
	s.employeeMu.Lock()
	defer s.employeeMu.Unlock()

	metrics := make(map[string]EmployeeMetrics[R], len(s.employees))
	for workerName, status := range s.employees {
		metrics[workerName] = EmployeeMetrics[R]{
			IsIdle:          status.isIdle,
			CurrentRequest:  status.currentRequest,
			HandledRequests: status.handledRequests,
		}
	}

	return metrics
}
