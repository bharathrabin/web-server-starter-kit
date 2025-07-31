package migrations

import (
	"coffee-and-running/src/storage"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
)

type Migration struct {
	Version   int
	Name      string
	UpSQL     string
	DownSQL   string
	Timestamp time.Time
}

type Migrator struct {
	engine        storage.Engine
	logger        *zap.Logger
	migrationsDir string
}

// NewMigrator creates a new migration runner
func NewMigrator(engine storage.Engine, logger *zap.Logger, migrationsDir string) *Migrator {
	return &Migrator{
		engine:        engine,
		logger:        logger,
		migrationsDir: migrationsDir,
	}
}

// ensureMigrationsTable creates the migrations tracking table if it doesn't exist
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`

	_, err := m.engine.Exec(ctx, query)
	if err != nil {
		m.logger.Error("failed to create migrations table", zap.Error(err))
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	m.logger.Debug("migrations table ensured")
	return nil
}

// loadMigrations reads all migration files from the migrations directory
func (m *Migrator) loadMigrations() ([]Migration, error) {
	var migrations []Migration

	err := filepath.WalkDir(m.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		filename := d.Name()
		migration, err := m.parseMigrationFile(path, filename)
		if err != nil {
			m.logger.Warn("skipping invalid migration file",
				zap.String("file", filename),
				zap.Error(err))
			return nil // Continue processing other files
		}

		migrations = append(migrations, migration)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	m.logger.Info("loaded migrations", zap.Int("count", len(migrations)))
	return migrations, nil
}

// parseMigrationFile parses migration files and extracts up/down SQL
// Expected format: 001_create_users_table.up.sql and 001_create_users_table.down.sql
func (m *Migrator) parseMigrationFile(path, filename string) (Migration, error) {
	// Check if this is an up or down file
	var isUpFile, isDownFile bool
	var baseName string

	if strings.HasSuffix(filename, ".up.sql") {
		isUpFile = true
		baseName = strings.TrimSuffix(filename, ".up.sql")
	} else if strings.HasSuffix(filename, ".down.sql") {
		isDownFile = true
		baseName = strings.TrimSuffix(filename, ".down.sql")
	} else {
		return Migration{}, fmt.Errorf("migration file must end with .up.sql or .down.sql: %s", filename)
	}

	// Parse version from filename (e.g., "001_create_users_table" -> version 1)
	parts := strings.Split(baseName, "_")
	if len(parts) < 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	versionStr := parts[0]
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return Migration{}, fmt.Errorf("invalid version in filename %s: %w", filename, err)
	}

	// Extract name (remove version)
	name := strings.Join(parts[1:], "_")

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file %s: %w", path, err)
	}

	// Get file modification time
	info, err := os.Stat(path)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to get file info for %s: %w", path, err)
	}

	migration := Migration{
		Version:   version,
		Name:      name,
		Timestamp: info.ModTime(),
	}

	// Set the appropriate SQL content
	if isUpFile {
		migration.UpSQL = strings.TrimSpace(string(content))
	} else if isDownFile {
		migration.DownSQL = strings.TrimSpace(string(content))
	}

	return migration, nil
}

// parseMigrationContent splits migration content into up and down parts
func (m *Migrator) parseMigrationContent(content string) (upSQL, downSQL string) {
	lines := strings.Split(content, "\n")
	var currentSection strings.Builder
	var inUpSection, inDownSection bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "-- +migrate Up") {
			inUpSection = true
			inDownSection = false
			currentSection.Reset()
			continue
		}

		if strings.Contains(trimmed, "-- +migrate Down") {
			if inUpSection {
				upSQL = strings.TrimSpace(currentSection.String())
			}
			inUpSection = false
			inDownSection = true
			currentSection.Reset()
			continue
		}

		if inUpSection || inDownSection {
			currentSection.WriteString(line + "\n")
		}
	}

	// Handle case where we're still in a section at end of file
	if inUpSection {
		upSQL = strings.TrimSpace(currentSection.String())
	} else if inDownSection {
		downSQL = strings.TrimSpace(currentSection.String())
	}

	// If no sections found, treat entire content as up migration
	if upSQL == "" && downSQL == "" {
		upSQL = strings.TrimSpace(content)
	}

	return upSQL, downSQL
}

// getAppliedMigrations returns list of applied migration versions
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	query := "SELECT version FROM schema_migrations ORDER BY version"
	rows, err := m.engine.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	return applied, nil
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	m.logger.Info("starting migration up")

	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	var pending []Migration
	for _, migration := range migrations {
		if !applied[migration.Version] {
			pending = append(pending, migration)
		}
	}

	if len(pending) == 0 {
		m.logger.Info("no pending migrations")
		return nil
	}

	m.logger.Info("applying migrations", zap.Int("count", len(pending)))

	for _, migration := range pending {
		if err := m.applyMigration(ctx, migration, Up); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w",
				migration.Version, migration.Name, err)
		}
	}

	m.logger.Info("all migrations applied successfully")
	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down(ctx context.Context) error {
	m.logger.Info("starting migration down")

	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	// Get the last applied migration
	query := "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1"
	row := m.engine.QueryRow(ctx, query)

	var lastVersion int
	if err := row.Scan(&lastVersion); err != nil {
		if err.Error() == "sql: no rows in result set" {
			m.logger.Info("no migrations to roll back")
			return nil
		}
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	// Load migration files to get the down SQL
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	var targetMigration *Migration
	for _, migration := range migrations {
		if migration.Version == lastVersion {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration file not found for version %d", lastVersion)
	}

	if err := m.applyMigration(ctx, *targetMigration, Down); err != nil {
		return fmt.Errorf("failed to roll back migration %d (%s): %w",
			targetMigration.Version, targetMigration.Name, err)
	}

	m.logger.Info("migration rolled back successfully",
		zap.Int("version", targetMigration.Version),
		zap.String("name", targetMigration.Name))

	return nil
}

// applyMigration applies a single migration in the specified direction
func (m *Migrator) applyMigration(ctx context.Context, migration Migration, direction Direction) error {
	var sql string
	switch direction {
	case Up:
		sql = migration.UpSQL
	case Down:
		sql = migration.DownSQL
	}

	if sql == "" {
		return fmt.Errorf("no %s SQL found for migration %d", direction, migration.Version)
	}

	m.logger.Info("applying migration",
		zap.Int("version", migration.Version),
		zap.String("name", migration.Name),
		zap.String("direction", string(direction)))

	// Start transaction
	tx, err := m.engine.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Update migrations table
	switch direction {
	case Up:
		_, err = tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			migration.Version, migration.Name)
	case Down:
		_, err = tx.Exec(ctx,
			"DELETE FROM schema_migrations WHERE version = $1",
			migration.Version)
	}

	if err != nil {
		return fmt.Errorf("failed to update migrations table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	m.logger.Info("migration applied successfully",
		zap.Int("version", migration.Version),
		zap.String("name", migration.Name),
		zap.String("direction", string(direction)))

	return nil
}

// Status shows the current migration status
func (m *Migrator) Status(ctx context.Context) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	m.logger.Info("migration status")
	for _, migration := range migrations {
		status := "pending"
		if applied[migration.Version] {
			status = "applied"
		}

		m.logger.Info("migration",
			zap.Int("version", migration.Version),
			zap.String("name", migration.Name),
			zap.String("status", status))
	}

	return nil
}

// Reset rolls back all migrations (BE CAREFUL!)
func (m *Migrator) Reset(ctx context.Context) error {
	m.logger.Warn("resetting all migrations - this will drop all data!")

	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	// Get all applied migrations in reverse order
	query := "SELECT version FROM schema_migrations ORDER BY version DESC"
	rows, err := m.engine.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		versions = append(versions, version)
	}

	if len(versions) == 0 {
		m.logger.Info("no migrations to reset")
		return nil
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	// Create a map for quick lookup
	migrationMap := make(map[int]Migration)
	for _, migration := range migrations {
		migrationMap[migration.Version] = migration
	}

	// Roll back each migration
	for _, version := range versions {
		migration, exists := migrationMap[version]
		if !exists {
			m.logger.Warn("migration file not found for applied version", zap.Int("version", version))
			continue
		}

		if err := m.applyMigration(ctx, migration, Down); err != nil {
			return fmt.Errorf("failed to roll back migration %d: %w", version, err)
		}
	}

	m.logger.Info("all migrations reset successfully")
	return nil
}
