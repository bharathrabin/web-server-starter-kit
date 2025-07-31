package main

import (
	"coffee-and-running/src/app"
	"coffee-and-running/src/config"
	"coffee-and-running/src/observability/logger"
	"coffee-and-running/src/observability/metrics"
	"coffee-and-running/src/server"
	"coffee-and-running/src/storage"
	"fmt"
	"log"
	"os"
)

const configFile = "CONFIG_FILE"

func main() {
	fPath, ok := os.LookupEnv(configFile)
	if !ok {
		log.Fatalf("please set %s env var", configFile)
	}
	cfg, err := config.LoadFromFile(fPath)
	if err != nil {
		log.Fatalf("failed to read config file: %s", err.Error())
	}
	app, err := buildApp(cfg)
	if err != nil {
		log.Fatalf("failed to build application: %s", err.Error())
	}
	app.Run()
}

func buildApp(cfg *config.Config) (app.Application, error) {
	lgr, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build app logger: %w", err)
	}
	metricsAgent, err := metrics.NewAgent(cfg.Metrics, lgr)
	if err != nil {
		return nil, fmt.Errorf("failed to buuld app metrics agent: %w", err)
	}
	engine, err := storage.NewEngine(cfg.Database, lgr, metricsAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to build app storage engine: %w", err)
	}
	srv := server.New(cfg.Server)

	return app.New(cfg, lgr, metricsAgent, engine, srv), nil
}
