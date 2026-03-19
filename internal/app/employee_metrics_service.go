package app

import "github.com/Kenji-Uema/staffSimulator/internal/domain"

type EmployeeMetrics[R domain.Request] struct {
	IsIdle          bool
	CurrentRequest  R
	HandledRequests int
}

func (s *employeeService[R]) IdleEmployeeCount() int {
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

func (s *employeeService[R]) EmployeeMetrics() map[string]EmployeeMetrics[R] {
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
