package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   *ServerConfig   `json:"server" yaml:"server"`
	Database *DatabaseConfig `json:"database" yaml:"database"`
	Logger   *LoggerConfig   `json:"logger" yaml:"logger"`
	Metrics  *MetricsConfig  `json:"metrics" yaml:"metrics"`
	App      *AppConfig      `json:"app" yaml:"app"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
	TLS             *TLSConfig    `json:"tls" yaml:"tls"`
	CORS            *CORSConfig   `json:"cors" yaml:"cors"`
}

// GetAddress returns the full server address
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers" yaml:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers" yaml:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials" yaml:"allow_credentials"`
	MaxAge           int      `json:"max_age" yaml:"max_age"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver             string        `json:"driver" yaml:"driver"`
	Host               string        `json:"host" yaml:"host"`
	Port               int           `json:"port" yaml:"port"`
	Name               string        `json:"name" yaml:"name"`
	User               string        `json:"user" yaml:"user"`
	Password           string        `json:"password" yaml:"password"`
	SSLMode            string        `json:"ssl_mode" yaml:"ssl_mode"`
	ConnectTimeout     time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	MaxOpenConns       int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns       int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	LogSlowQueries     bool          `json:"log_slow_queries" yaml:"log_slow_queries"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold" yaml:"slow_query_threshold"`
}

// GetDSN returns the database connection string
func (d DatabaseConfig) GetDSN() string {
	switch d.Driver {
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
			d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, int(d.ConnectTimeout.Seconds()))
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s",
			d.User, d.Password, d.Host, d.Port, d.Name, d.ConnectTimeout)
	case "sqlite", "sqlite3":
		return d.Name
	default:
		return ""
	}
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level             string `json:"level" yaml:"level"`
	Format            string `json:"format" yaml:"format"` // json, console
	Output            string `json:"output" yaml:"output"` // stdout, stderr, file
	File              string `json:"file" yaml:"file"`
	MaxSize           int    `json:"max_size" yaml:"max_size"` // MB
	MaxBackups        int    `json:"max_backups" yaml:"max_backups"`
	MaxAge            int    `json:"max_age" yaml:"max_age"` // days
	Compress          bool   `json:"compress" yaml:"compress"`
	Development       bool   `json:"development" yaml:"development"`
	DisableCaller     bool   `json:"disable_caller" yaml:"disable_caller"`
	DisableStacktrace bool   `json:"disable_stacktrace" yaml:"disable_stacktrace"`
	ComponentLogging  bool   `json:"component_logging" yaml:"component_logging"`
}

// MetricsConfig holds metrics/StatsD configuration
type MetricsConfig struct {
	Enabled        bool          `json:"enabled" yaml:"enabled"`
	Type           string        `json:"type" yaml:"type"`                       // alexcesaro, custom, mock, buffered
	Address        string        `json:"address" yaml:"address"`                 // localhost:8125
	Prefix         string        `json:"prefix" yaml:"prefix"`                   // myapp
	UnderlyingType string        `json:"underlying_type" yaml:"underlying_type"` // for buffered client
	BufferSize     int           `json:"buffer_size" yaml:"buffer_size"`         // for buffered client
	FlushInterval  time.Duration `json:"flush_interval" yaml:"flush_interval"`   // for buffered client
	ReportInterval time.Duration `json:"report_interval" yaml:"report_interval"` // for periodic stats
	Tags           []string      `json:"tags" yaml:"tags"`                       // global tags
}

// AppConfig holds general application configuration
type AppConfig struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Environment string `json:"environment" yaml:"environment"` // development, staging, production
	Debug       bool   `json:"debug" yaml:"debug"`
	Region      string `json:"region" yaml:"region"`
	InstanceID  string `json:"instance_id" yaml:"instance_id"`
}

// IsProduction returns true if running in production environment
func (a AppConfig) IsProduction() bool {
	return strings.ToLower(a.Environment) == "production"
}

// IsDevelopment returns true if running in development environment
func (a AppConfig) IsDevelopment() bool {
	return strings.ToLower(a.Environment) == "development"
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filename string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	return config, nil
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {

	return &Config{
		Server: &ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			TLS: &TLSConfig{
				Enabled: false,
			},
			CORS: &CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
				MaxAge:         86400,
			},
		},
		Database: &DatabaseConfig{
			Driver:             "postgres",
			Host:               "localhost",
			Port:               5432,
			Name:               "myapp",
			User:               "postgres",
			Password:           "",
			SSLMode:            "disable",
			ConnectTimeout:     30 * time.Second,
			MaxOpenConns:       25,
			MaxIdleConns:       5,
			ConnMaxLifetime:    5 * time.Minute,
			ConnMaxIdleTime:    5 * time.Minute,
			LogSlowQueries:     true,
			SlowQueryThreshold: 500 * time.Millisecond,
		},
		Logger: &LoggerConfig{
			Level:             "info",
			Format:            "json",
			Output:            "stdout",
			Development:       false,
			ComponentLogging:  true,
			DisableCaller:     false,
			DisableStacktrace: false,
		},
		Metrics: &MetricsConfig{
			Enabled:        true,
			Type:           "alexcesaro",
			Address:        "localhost:8125",
			Prefix:         "myapp",
			BufferSize:     100,
			FlushInterval:  5 * time.Second,
			ReportInterval: 30 * time.Second,
		},
		App: &AppConfig{
			Name:        "myapp",
			Version:     "1.0.0",
			Environment: "development",
			Debug:       true,
			Region:      "us-east-1",
		},
	}
}

// String returns a string representation of the config (with sensitive data masked)
func (c *Config) String() string {
	masked := *c
	masked.Database.Password = "***"

	data, _ := yaml.Marshal(masked)
	return string(data)
}
