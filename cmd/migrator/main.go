package main

import (
	"coffee-and-running/src/config"
	"coffee-and-running/src/migrations"
	"coffee-and-running/src/observability/logger"
	"coffee-and-running/src/observability/metrics"
	"coffee-and-running/src/storage"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	configFileEnv        = "CONFIG_FILE"
	defaultMigrationsDir = "scripts/migrations"
)

func main() {
	var (
		command       = flag.String("command", "", "Migration command: up, down, status, reset")
		configFile    = flag.String("config", "", "Path to config file (overrides CONFIG_FILE env var)")
		migrationsDir = flag.String("migrations-dir", defaultMigrationsDir, "Path to migrations directory")
		timeout       = flag.Duration("timeout", 30*time.Second, "Migration timeout")
	)
	flag.Parse()

	if *command == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -command=<up|down|status|reset> [options]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Get config file path
	cfgPath := *configFile
	if cfgPath == "" {
		cfgPath = os.Getenv(configFileEnv)
	}
	if cfgPath == "" {
		log.Fatalf("config file not specified. Use -config flag or set %s env var", configFileEnv)
	}

	// Load configuration
	cfg, err := config.LoadFromFile(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Setup logger
	lgr, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer lgr.Sync()

	// Setup metrics (can be no-op for migrations)
	metricsAgent, err := metrics.NewAgent(cfg.Metrics, lgr)
	if err != nil {
		log.Fatalf("failed to create metrics agent: %v", err)
	}
	defer metricsAgent.Close()

	// Setup database engine
	engine, err := storage.NewEngine(cfg.Database, lgr, metricsAgent)
	if err != nil {
		log.Fatalf("failed to create database engine: %v", err)
	}
	defer engine.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := engine.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Create migrator
	migrator := migrations.NewMigrator(engine, lgr, *migrationsDir)

	// Execute command with timeout
	ctx, cancel = context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	switch *command {
	case "up":
		err = migrator.Up(ctx)
	case "down":
		err = migrator.Down(ctx)
	case "status":
		err = migrator.Status(ctx)
	case "reset":
		fmt.Print("This will reset ALL migrations and drop all data. Are you sure? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Migration reset cancelled")
			return
		}
		err = migrator.Reset(ctx)
	default:
		log.Fatalf("unknown command: %s. Use: up, down, status, or reset", *command)
	}

	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Printf("Migration command '%s' completed successfully\n", *command)
}
