package app

import (
	"coffee-and-running/src/config"
	"coffee-and-running/src/observability/metrics"
	"coffee-and-running/src/storage"
	"net/http"

	"go.uber.org/zap"
)

type Application struct {
	Config *config.Config
	Logger *zap.Logger
	Engine *storage.Engine
	Server *http.Server
	Stats  metrics.Agent
}

func New(config *config.Config, logger *zap.Logger, engine *storage.Engine, server *http.Server, stats metrics.Agent) *Application {
	return &Application{
		Config: config,
		Logger: logger,
		Engine: engine,
		Server: server,
		Stats:  stats,
	}
}
