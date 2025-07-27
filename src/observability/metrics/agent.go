package metrics

import (
	"coffee-and-running/src/config"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alexcesaro/statsd"
	"go.uber.org/zap"
)

type Agent interface {
	Increment(bucket string)
	Count(bucket string, n interface{})
	Timing(bucket string, value interface{})
	Gauge(bucket string, value interface{})
	Close()
	IsEnabled() bool
}

type agent struct {
	config *config.MetricsConfig
	client *statsd.Client
	logger *zap.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Close implements Agent.
func (a *agent) Close() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
	if a.client != nil {
		a.client.Close()
	}
}

// Count implements Agent.
func (a *agent) Count(bucket string, n interface{}) {
	a.client.Count(bucket, n)
}

// Gauge implements Agent.
func (a *agent) Gauge(bucket string, value interface{}) {
	a.client.Gauge(bucket, value)
}

// Increment implements Agent.
func (a *agent) Increment(bucket string) {
	a.client.Increment(bucket)
}

// IsEnabled implements Agent.
func (a *agent) IsEnabled() bool {
	return a.config.Enabled
}

// Timing implements Agent.
func (a *agent) Timing(bucket string, value interface{}) {
	panic("unimplemented")
}

func NewAgent(cfg *config.MetricsConfig, logger *zap.Logger) (Agent, error) {
	if !cfg.Enabled {
		logger.Info("metrics disabled, using no-op client")
		return &agent{
			config: cfg,
			logger: logger,
		}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	agent := &agent{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
	client, err := agent.createClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}
	agent.client = client
	// Start periodic reporting if configured
	if cfg.ReportInterval > 0 {
		agent.startPeriodicReporting()
	}
	logger.Info("metrics agent initialized",
		zap.String("type", cfg.Type),
		zap.String("address", cfg.Address),
		zap.String("prefix", cfg.Prefix),
		zap.Strings("tags", cfg.Tags),
	)
	return agent, nil
}

// createClient creates the appropriate client based on configuration
func (a *agent) createClient() (*statsd.Client, error) {
	opts := []statsd.Option{}

	if a.config.Address != "" {
		opts = append(opts, statsd.Address(a.config.Address))
	}

	if a.config.Prefix != "" {
		opts = append(opts, statsd.Prefix(a.config.Prefix))
	}

	if len(a.config.Tags) > 0 {
		opts = append(opts, statsd.TagsFormat(statsd.InfluxDB))
	}

	return statsd.New(opts...)
}

// startPeriodicReporting starts a goroutine for periodic metric reporting
func (a *agent) startPeriodicReporting() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		ticker := time.NewTicker(a.config.ReportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				a.reportSystemMetrics()
			}
		}
	}()
}

// reportSystemMetrics reports system-level metrics
func (a *agent) reportSystemMetrics() {
	// Example system metrics - you can extend this
	a.client.Gauge("system.uptime", time.Now().Unix())
	a.client.Increment("system.heartbeat")

	a.logger.Debug("system metrics reported")
}
