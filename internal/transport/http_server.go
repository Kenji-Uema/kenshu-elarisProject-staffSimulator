package transport

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mdb"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/staffSimulator/internal/transport/http/probe"
)

func StartHTTPServer(appCfg config.AppConfig, rabbitMqClient *mq.RabbitMqConnection, mongoClient *mdb.Mdb) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", probe.HealthHandler)
	mux.HandleFunc("/readyz", probe.ReadinessHandler(rabbitMqClient, mongoClient))

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", appCfg.Server.Host, appCfg.Server.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("pprof http listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http serve", "error", err)
		}
	}()

	return server
}
