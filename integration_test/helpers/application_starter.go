package helpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type ApplicationConfig struct {
	AppPort          int
	ClockHost        string
	ClockPort        string
	MongoURI         string
	MongoDatabase    string
	RabbitHost       string
	RabbitPort       int
	CleaningExchange string
	DayExchange      string
	HourExchange     string
}

func ApplicationStart(t TestReporter, cfg ApplicationConfig) (stop func(), runErr <-chan error) {
	t.Helper()

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)

	cmd := exec.CommandContext(runCtx, "go", "run", "./internal")
	cmd.Dir = "/home/kenjiuema/Documents/projects/staffSimulator"
	cmd.Env = envWithOverrides(os.Environ(), applicationEnv(cfg))

	go func() {
		err := cmd.Run()
		if runCtx.Err() != nil {
			runErrCh <- nil
			return
		}
		runErrCh <- err
	}()

	stop = func() {
		cancelRun()
	}

	return stop, runErrCh
}

func applicationEnv(cfg ApplicationConfig) map[string]string {
	return map[string]string{
		"SERVICE_NAME":                      "staff-simulator",
		"VERSION":                           "integration-test",
		"SERVICE_HOST":                      "127.0.0.1",
		"SERVICE_PORT":                      fmt.Sprintf("%d", cfg.AppPort),
		"LOG_LEVEL":                         "0",
		"CLEANING_EMPLOYEES_NAMES":          "alice",
		"LAUNDERING_EMPLOYEES_NAMES":        "bob",
		"STOCKERS_EMPLOYEES_NAMES":          "carol",
		"MONGO_INITDB_ROOT_USERNAME":        "test_user",
		"MONGO_INITDB_ROOT_PASSWORD":        "test_pass",
		"MONGO_HOST":                        cfg.MongoURI,
		"MONGO_DATABASE":                    cfg.MongoDatabase,
		"COTTAGE_COLLECTION":                "cottage",
		"STOCK_COLLECTION":                  "Stock",
		"RABBITMQ_USERNAME":                 "test_user",
		"RABBITMQ_PASSWORD":                 "test_pass",
		"RABBITMQ_HOST":                     cfg.RabbitHost,
		"RABBITMQ_PORT":                     fmt.Sprintf("%d", cfg.RabbitPort),
		"CLEANING_QUEUE_NAME":               fmt.Sprintf("integration.cleaning.queue.%d", cfg.AppPort),
		"CLEANING_QUEUE_AUTO_DELETE":        "true",
		"CLEANING_BINDING_EXCHANGE_NAME":    cfg.CleaningExchange,
		"CLEANING_BINDING_ROUTING_KEY":      "cleaning.request",
		"DAY_CHANGE_QUEUE_NAME":             fmt.Sprintf("integration.day.queue.%d", cfg.AppPort),
		"DAY_CHANGE_QUEUE_AUTO_DELETE":      "true",
		"DAY_CHANGE_BINDING_EXCHANGE_NAME":  cfg.DayExchange,
		"DAY_CHANGE_BINDING_ROUTING_KEY":    "day.change",
		"HOUR_CHANGE_QUEUE_NAME":            fmt.Sprintf("integration.hour.queue.%d", cfg.AppPort),
		"HOUR_CHANGE_QUEUE_AUTO_DELETE":     "true",
		"HOUR_CHANGE_BINDING_EXCHANGE_NAME": cfg.HourExchange,
		"HOUR_CHANGE_BINDING_ROUTING_KEY":   "hour.change",
		"CLOCK_EMU_GRPC_HOST":               cfg.ClockHost,
		"CLOCK_EMU_GRPC_PORT":               cfg.ClockPort,
	}
}

func envWithOverrides(baseEnv []string, overrides map[string]string) []string {
	result := make([]string, 0, len(baseEnv)+len(overrides))
	seen := make(map[string]struct{}, len(baseEnv))

	for _, entry := range baseEnv {
		key, _, ok := stringsCut(entry)
		if !ok {
			result = append(result, entry)
			continue
		}

		if override, ok := overrides[key]; ok {
			result = append(result, fmt.Sprintf("%s=%s", key, override))
		} else {
			result = append(result, entry)
		}
		seen[key] = struct{}{}
	}

	for key, value := range overrides {
		if _, ok := seen[key]; ok {
			continue
		}
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}

func stringsCut(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return "", "", false
}
