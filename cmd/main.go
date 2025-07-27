package cmd

import (
	"coffee-and-running/src/app"
	"coffee-and-running/src/config"
	"coffee-and-running/src/observability/logger"
	"coffee-and-running/src/observability/metrics"
	"fmt"
	"log"
	"os"
)

const configFile = "CONFIG_FILE"

func main() {
	fPath, ok := os.LookupEnv(configFile)
	if !ok {
		log.Panicf("please set %s env var", configFile)
	}
	cfg, err := config.LoadFromFile(fPath)
	if err != nil {
		log.Panicf("failed to read config file: %s", err.Error())
	}
	app, err := BuildApplication(cfg)
	if err != nil {
		log.Panicf("failed to build application: %s", err.Error())
	}
	app.Run()
}

func BuildApplication(cfg *config.Config) (*app.Application, error) {
	lgr, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build app logger: %w", err)
	}
	metricsAgent, err := metrics.NewAgent(cfg.Metrics, lgr)
	if err != nil {
		return nil, fmt.Errorf("failed to buuld app metrics agent: %w", err)
	}

}
