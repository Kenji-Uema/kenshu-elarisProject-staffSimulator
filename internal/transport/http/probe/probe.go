package probe

import (
	"log/slog"
	"net/http"

	"github.com/Kenji-Uema/staffSimulator/internal/infra/mdb"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mq"
)

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func ReadinessHandler(rabbitMqClient *mq.RabbitMqConnection, client *mdb.Mdb) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rabbitMqClient.IsConnectionOpen() {
			slog.ErrorContext(r.Context(), "rabbitmq connection is not open")
			http.Error(w, "rabbitmq connection closed", http.StatusServiceUnavailable)
			return
		}

		if err := client.Ping(); err != nil {
			slog.ErrorContext(r.Context(), "db ping failed", "error", err)
			http.Error(w, "db ping failed", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
