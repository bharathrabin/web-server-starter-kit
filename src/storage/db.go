package storage

import (
	"coffee-and-running/src/config"
	"coffee-and-running/src/observability/metrics"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Engine interface {
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Begin(ctx context.Context) (*InstrumentedTx, error)
	Prepare(ctx context.Context, query string) (*InstrumentedStmt, error)
	Ping(ctx context.Context) error
	Close() error
	Stats() sql.DBStats
}

// Engine is the app's storage engine wrapped with a logger and metrics
type engine struct {
	logger *zap.Logger
	db     *sql.DB
	stats  metrics.Agent
}

// NewEngineWithComponent creates a new instrumented database engine with custom component name
func NewEngine(cfg *config.DatabaseConfig, logger *zap.Logger, stats metrics.Agent) (Engine, error) {
	componentLogger := logger.With(zap.String("component", "db_engine"))

	// Get the DSN from the config
	dsn := cfg.GetDSN()
	if dsn == "" {
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
	db, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		componentLogger.Error("failed to open database connection",
			zap.Error(err),
			zap.String("driver", cfg.Driver))
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// Configure connection pool settings
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		componentLogger.Error("failed to ping database",
			zap.Error(err),
			zap.String("driver", cfg.Driver))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	componentLogger.Info("database connection established successfully",
		zap.String("driver", cfg.Driver),
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name))

	return &engine{
		logger: componentLogger,
		db:     db,
		stats:  stats,
	}, nil
}

// Query executes a query with logging and metrics
func (e *engine) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	e.logger.Debug("executing query",
		zap.String("query", query),
		zap.Any("args", args),
	)

	rows, err := e.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	// Log the result
	if err != nil {
		e.logger.Error("query failed",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		e.stats.Increment("db.query.error")
	} else {
		e.logger.Debug("query completed",
			zap.String("query", query),
			zap.Duration("duration", duration),
		)
		e.stats.Increment("db.query.success")
	}

	e.stats.Timing("db.query.duration", duration)
	return rows, err
}

// QueryRow executes a single row query with logging and metrics
func (e *engine) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()

	e.logger.Debug("executing query row",
		zap.String("query", query),
		zap.Any("args", args),
	)

	row := e.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	e.logger.Debug("query row completed",
		zap.String("query", query),
		zap.Duration("duration", duration),
	)

	e.stats.Timing("db.queryrow.duration", duration)
	e.stats.Increment("db.queryrow.count")

	return row
}

// Exec executes a statement with logging and metrics
func (e *engine) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()

	e.logger.Debug("executing statement",
		zap.String("query", query),
		zap.Any("args", args),
	)

	result, err := e.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("statement execution failed",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		e.stats.Increment("db.exec.error")
	} else {
		rowsAffected, _ := result.RowsAffected()
		e.logger.Debug("statement completed",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Int64("rows_affected", rowsAffected),
		)
		e.stats.Increment("db.exec.success")
		e.stats.Count("db.rows_affected", rowsAffected)
	}

	e.stats.Timing("db.exec.duration", duration)
	return result, err
}

// Begin starts a transaction with logging and metrics
func (e *engine) Begin(ctx context.Context) (*InstrumentedTx, error) {
	start := time.Now()

	e.logger.Debug("beginning transaction")

	tx, err := e.db.BeginTx(ctx, nil)
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("failed to begin transaction",
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		e.stats.Increment("db.transaction.begin.error")
		return nil, err
	}

	e.logger.Debug("transaction began",
		zap.Duration("duration", duration),
	)
	e.stats.Increment("db.transaction.begin.success")
	e.stats.Timing("db.transaction.begin.duration", duration)

	return &InstrumentedTx{
		tx:     tx,
		logger: e.logger,
		stats:  e.stats,
		start:  start,
	}, nil
}

// Prepare creates a prepared statement with logging and metrics
func (e *engine) Prepare(ctx context.Context, query string) (*InstrumentedStmt, error) {
	start := time.Now()

	e.logger.Debug("preparing statement",
		zap.String("query", query),
	)

	stmt, err := e.db.PrepareContext(ctx, query)
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("failed to prepare statement",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		e.stats.Increment("db.prepare.error")
		return nil, err
	}

	e.logger.Debug("statement prepared",
		zap.String("query", query),
		zap.Duration("duration", duration),
	)
	e.stats.Increment("db.prepare.success")
	e.stats.Timing("db.prepare.duration", duration)

	return &InstrumentedStmt{
		stmt:   stmt,
		query:  query,
		logger: e.logger,
		stats:  e.stats,
	}, nil
}

// Ping tests the database connection with logging and metrics
func (e *engine) Ping(ctx context.Context) error {
	start := time.Now()

	e.logger.Debug("pinging database")

	err := e.db.PingContext(ctx)
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("database ping failed",
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		e.stats.Increment("db.ping.error")
	} else {
		e.logger.Debug("database ping successful",
			zap.Duration("duration", duration),
		)
		e.stats.Increment("db.ping.success")
	}

	e.stats.Timing("db.ping.duration", duration)
	return err
}

// Close closes the database connection with logging
func (e *engine) Close() error {
	e.logger.Info("closing database connection")

	err := e.db.Close()
	if err != nil {
		e.logger.Error("failed to close database connection", zap.Error(err))
		e.stats.Increment("db.close.error")
	} else {
		e.logger.Info("database connection closed successfully")
		e.stats.Increment("db.close.success")
	}

	return err
}

// Stats returns database statistics with logging
func (e *engine) Stats() sql.DBStats {
	stats := e.db.Stats()

	e.logger.Debug("database stats",
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle),
		zap.Int64("wait_count", stats.WaitCount),
		zap.Duration("wait_duration", stats.WaitDuration),
		zap.Int64("max_idle_closed", stats.MaxIdleClosed),
		zap.Int64("max_idle_time_closed", stats.MaxIdleTimeClosed),
		zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
	)

	// Send connection pool metrics
	e.stats.Gauge("db.connections.open", stats.OpenConnections)
	e.stats.Gauge("db.connections.in_use", stats.InUse)
	e.stats.Gauge("db.connections.idle", stats.Idle)
	e.stats.Count("db.connections.wait_count", stats.WaitCount)
	e.stats.Timing("db.connections.wait_duration", stats.WaitDuration)

	return stats
}

// InstrumentedTx wraps sql.Tx with logging and metrics
type InstrumentedTx struct {
	tx     *sql.Tx
	logger *zap.Logger
	stats  metrics.Agent
	start  time.Time
}

// Commit commits the transaction with logging and metrics
func (tx *InstrumentedTx) Commit() error {
	duration := time.Since(tx.start)

	tx.logger.Debug("committing transaction")

	err := tx.tx.Commit()
	if err != nil {
		tx.logger.Error("transaction commit failed",
			zap.Duration("total_duration", duration),
			zap.Error(err),
		)
		tx.stats.Increment("db.transaction.commit.error")
	} else {
		tx.logger.Debug("transaction committed successfully",
			zap.Duration("total_duration", duration),
		)
		tx.stats.Increment("db.transaction.commit.success")
	}

	tx.stats.Timing("db.transaction.total_duration", duration)
	return err
}

// Rollback rolls back the transaction with logging and metrics
func (tx *InstrumentedTx) Rollback() error {
	duration := time.Since(tx.start)

	tx.logger.Debug("rolling back transaction")

	err := tx.tx.Rollback()
	if err != nil {
		tx.logger.Error("transaction rollback failed",
			zap.Duration("total_duration", duration),
			zap.Error(err),
		)
		tx.stats.Increment("db.transaction.rollback.error")
	} else {
		tx.logger.Debug("transaction rolled back successfully",
			zap.Duration("total_duration", duration),
		)
		tx.stats.Increment("db.transaction.rollback.success")
	}

	tx.stats.Timing("db.transaction.total_duration", duration)
	return err
}

// Query executes a query within the transaction
func (tx *InstrumentedTx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	tx.logger.Debug("executing query in transaction",
		zap.String("query", query),
		zap.Any("args", args),
	)

	rows, err := tx.tx.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		tx.logger.Error("transaction query failed",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		tx.stats.Increment("db.transaction.query.error")
	} else {
		tx.logger.Debug("transaction query completed",
			zap.String("query", query),
			zap.Duration("duration", duration),
		)
		tx.stats.Increment("db.transaction.query.success")
	}

	tx.stats.Timing("db.transaction.query.duration", duration)
	return rows, err
}

// Exec executes a statement within the transaction
func (tx *InstrumentedTx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()

	tx.logger.Debug("executing statement in transaction",
		zap.String("query", query),
		zap.Any("args", args),
	)

	result, err := tx.tx.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		tx.logger.Error("transaction statement execution failed",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		tx.stats.Increment("db.transaction.exec.error")
	} else {
		rowsAffected, _ := result.RowsAffected()
		tx.logger.Debug("transaction statement completed",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.Int64("rows_affected", rowsAffected),
		)
		tx.stats.Increment("db.transaction.exec.success")
		tx.stats.Count("db.transaction.rows_affected", rowsAffected)
	}

	tx.stats.Timing("db.transaction.exec.duration", duration)
	return result, err
}

// InstrumentedStmt wraps sql.Stmt with logging and metrics
type InstrumentedStmt struct {
	stmt   *sql.Stmt
	query  string
	logger *zap.Logger
	stats  metrics.Agent
}

// Query executes the prepared statement query
func (s *InstrumentedStmt) Query(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	s.logger.Debug("executing prepared statement query",
		zap.String("query", s.query),
		zap.Any("args", args),
	)

	rows, err := s.stmt.QueryContext(ctx, args...)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("prepared statement query failed",
			zap.String("query", s.query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		s.stats.Increment("db.prepared.query.error")
	} else {
		s.logger.Debug("prepared statement query completed",
			zap.String("query", s.query),
			zap.Duration("duration", duration),
		)
		s.stats.Increment("db.prepared.query.success")
	}

	s.stats.Timing("db.prepared.query.duration", duration)
	return rows, err
}

// Exec executes the prepared statement
func (s *InstrumentedStmt) Exec(ctx context.Context, args ...interface{}) (sql.Result, error) {
	start := time.Now()

	s.logger.Debug("executing prepared statement",
		zap.String("query", s.query),
		zap.Any("args", args),
	)

	result, err := s.stmt.ExecContext(ctx, args...)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("prepared statement execution failed",
			zap.String("query", s.query),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		s.stats.Increment("db.prepared.exec.error")
	} else {
		rowsAffected, _ := result.RowsAffected()
		s.logger.Debug("prepared statement completed",
			zap.String("query", s.query),
			zap.Duration("duration", duration),
			zap.Int64("rows_affected", rowsAffected),
		)
		s.stats.Increment("db.prepared.exec.success")
		s.stats.Count("db.prepared.rows_affected", rowsAffected)
	}

	s.stats.Timing("db.prepared.exec.duration", duration)
	return result, err
}

// Close closes the prepared statement
func (s *InstrumentedStmt) Close() error {
	s.logger.Debug("closing prepared statement", zap.String("query", s.query))

	err := s.stmt.Close()
	if err != nil {
		s.logger.Error("failed to close prepared statement",
			zap.String("query", s.query),
			zap.Error(err),
		)
		s.stats.Increment("db.prepared.close.error")
	} else {
		s.logger.Debug("prepared statement closed successfully",
			zap.String("query", s.query),
		)
		s.stats.Increment("db.prepared.close.success")
	}

	return err
}
