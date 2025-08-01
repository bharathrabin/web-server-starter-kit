# Go Web Server Starter Kit

A production-ready Go web server starter kit that eliminates the boilerplate setup time. Clone, configure, and start building your business logic immediately instead of spending hours wiring up logging, metrics, database connections, and other infrastructure components.

## Features

### Core Infrastructure
- ** Structured Logging** - Zap-based logger with configurable formats, outputs, and log rotation
- ** Metrics & Observability** - StatsD integration with comprehensive instrumentation
- ** Database Layer** - Instrumented PostgreSQL engine with connection pooling and transaction support
- ** HTTP Server** - Chi router with CORS, middleware, and TLS support
- ** Graceful Shutdown** - Proper cleanup and shutdown handling
- ** Docker Ready** - Multi-stage Dockerfile and docker-compose setup
- ** gRPC Support** - Protobuf generation and gRPC server capabilities

### Production Features
- Configuration-driven setup via YAML
- Comprehensive error handling and logging
- Connection pool management
- Request timeouts and rate limiting
- TLS/SSL support
- Health checks and monitoring
- Database migrations support

## Project Structure

```
├── src/
│   ├── app/                    # Application lifecycle management
│   ├── config/                 # Configuration management
│   ├── observability/
│   │   ├── logger/            # Zap logger setup
│   │   └── metrics/           # StatsD metrics agent
│   ├── server/                # HTTP server with Chi router
│   └── storage/               # Database engine with instrumentation
├── proto/                     # Protocol buffer definitions
├── scripts/
│   └── migrations/            # SQL migration files
├── gen/                       # Generated protobuf code
├── migrations/                # Database migration files
├── docker-compose.yml         # Development environment
├── Dockerfile                 # Multi-stage production build
├── Makefile                   # Development workflow automation
└── config-development.yaml    # Development configuration
```

##  Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- Protocol Buffers compiler (for gRPC features)

### 1. Clone and Setup
```bash
git clone <repo-url> my-web-service
cd my-web-service

# Install all development tools and dependencies
make setup
```

### 2. Configuration
Edit `config-development.yaml` to match your requirements:

```yaml
server:
  host: "0.0.0.0"
  port: 3000
  read_timeout: 15s
  write_timeout: 15s

database:
  driver: "postgres"
  host: "postgres"
  port: 5432
  name: "myapp_dev"
  user: "postgres"
  password: "devpassword"

logger:
  level: "info"
  format: "json"
  development: true

metrics:
  enabled: true
  type: "statsd"
  address: "statsd:8125"
  prefix: "myapp"
```

### 3. Start Development Environment
```bash
# Start all services (app, postgres, metrics)
make dev-start

# Your app will be available at:
# - API: http://localhost:3000
# - Metrics Dashboard: http://localhost:8080
# - PostgreSQL: localhost:5432
```

### 4. Development Workflow
```bash
# Run with auto-reload during development
make dev

# Run tests
make test

# Generate protobuf code (if using gRPC)
make gen

# View logs
make compose-logs

# Reset everything for fresh start
make dev-reset
```

##  Available Make Commands

### Development
```bash
make setup          # Initial project setup
make dev            # Auto-reload development server
make dev-start      # Start development environment
make test           # Run tests with coverage
make check          # Run all quality checks (fmt, vet, lint, test)
```

### Code Quality
```bash
make fmt            # Format Go code
make lint           # Run golangci-lint
make vet            # Run go vet
make tidy           # Tidy go modules
```

### Protobuf/gRPC
```bash
make install-deps   # Install protobuf tools
make gen            # Generate protobuf code
make proto          # Generate protobuf only
make run-grpc       # Run gRPC server
make run-http       # Run HTTP/REST server
```

### Docker
```bash
make docker-build       # Build Docker image
make docker-run         # Run container
make compose-up         # Start all services
make compose-down       # Stop all services
make compose-logs       # View logs
```

### Database
```bash
make db-migrate         # Run all pending migrations
make db-migrate-down    # Rollback last migration
make db-migrate-status  # Show migration status
make db-reset           # Reset all migrations and reseed
make create-migration NAME="create_users_table"  # Create new migration
```

### Database Migrations

The starter kit includes a built-in migration system using your existing storage abstraction. Migrations are stored in `scripts/migrations/` and follow a simple naming convention.

#### Migration File Format
Migration files use separate `.up.sql` and `.down.sql` files with the format: `001_migration_name.up.sql` and `001_migration_name.down.sql`

**001_create_users_table.up.sql:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

**001_create_users_table.down.sql:**
```sql
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
```

#### Migration Commands
```bash
# Create a new migration
make create-migration NAME="create_users_table"

# Run all pending migrations
make db-migrate

# Check migration status
make db-migrate-status

# Rollback the last migration
make db-migrate-down

# Reset everything (CAREFUL!)
make db-reset
```

##  Core Components

### Logger
Structured logging with Zap, supporting:
- Multiple output formats (JSON, Console)
- Log rotation with Lumberjack
- Configurable log levels
- Development vs Production modes

```go
// Usage in your code
logger.Info("User created", 
    zap.String("user_id", userID),
    zap.String("email", email))
```

### Metrics Agent
StatsD-based metrics collection:
- Counters, Gauges, Timings
- Automatic database metrics
- Custom business metrics
- Grafana dashboard ready

```go
// Usage in your code
stats.Increment("user.created")
stats.Timing("api.response_time", duration)
stats.Gauge("active_connections", connCount)
```

### Database Engine
Instrumented PostgreSQL wrapper with built-in migration system:
- Connection pooling and health monitoring
- Query logging and performance metrics
- Transaction support with proper cleanup
- Built-in migration runner using SQL files
- Prepared statements with automatic instrumentation

```go
// Usage in your code
rows, err := engine.Query(ctx, "SELECT * FROM users WHERE active = $1", true)
tx, err := engine.Begin(ctx)

// Migrations are handled automatically
// Just create SQL files in scripts/migrations/
```

### HTTP Server
Production-ready Chi router:
- CORS support
- Middleware stack
- Request timeouts
- TLS configuration
- Graceful shutdown

```go
// Add your routes
router.Get("/health", healthHandler)
router.Route("/api/v1", func(r chi.Router) {
    r.Get("/users", getUsersHandler)
    r.Post("/users", createUserHandler)
})
```

##  Docker & Deployment

### Development with Docker Compose
```bash
# Start everything
make compose-up

# View logs
make compose-logs

# Restart just the app
make compose-restart-app
```

### Production Docker Build
```bash
# Build optimized image
make docker-build

# The Dockerfile uses multi-stage builds for minimal image size
# Final image is ~10MB using scratch base
```

### Environment Variables
```bash
export CONFIG_FILE=/path/to/config.yaml
```

##  Adding Your Business Logic

### 1. Add Routes
Edit `src/server/routes.go` (create if needed):
```go
func SetupRoutes(r *chi.Mux, deps *Dependencies) {
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/users", deps.UserHandler.GetUsers)
        r.Post("/users", deps.UserHandler.CreateUser)
    })
}
```

### 2. Create Handlers
```go
type UserHandler struct {
    logger *zap.Logger
    db     storage.Engine
    stats  metrics.Agent
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    defer h.stats.Timing("api.get_users.duration", time.Since(start))
    
    // Your business logic here
    h.logger.Info("Getting users", zap.String("request_id", middleware.GetReqID(r.Context())))
}
```

### 3. Add Database Models
```go
type User struct {
    ID        int       `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (u *UserService) GetUsers(ctx context.Context) ([]User, error) {
    rows, err := u.db.Query(ctx, "SELECT id, email, created_at FROM users")
    // Handle results...
}
```

##  Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run benchmarks
make bench
```

Example test structure:
```go
func TestUserHandler_GetUsers(t *testing.T) {
    // Setup test dependencies
    logger := zaptest.NewLogger(t)
    mockDB := &MockEngine{}
    mockStats := &MockAgent{}
    
    handler := &UserHandler{
        logger: logger,
        db:     mockDB,
        stats:  mockStats,
    }
    
    // Test your handler
}
```

##  Monitoring & Observability

### Logs
- Structured JSON logs in production
- Colorized console logs in development
- Automatic request/response logging
- Database query logging with performance metrics

### Metrics
The starter kit automatically tracks:
- HTTP request/response metrics
- Database connection pool stats
- Query execution times
- Error rates and counts
- Custom business metrics

### Health Checks
```bash
curl http://localhost:3000/health
```

##  Security Features

- TLS/SSL support with modern cipher suites
- CORS configuration
- Request timeouts to prevent abuse
- SQL injection protection with prepared statements
- Structured logging (no sensitive data leakage)

##  Deployment

### Using Docker
```bash
# Build production image
docker build -t myapp:latest .

# Run in production
docker run -d \
  -p 3000:3000 \
  -v /path/to/config.yaml:/app/config.yaml:ro \
  -e CONFIG_FILE=/app/config.yaml \
  myapp:latest
```

### Using Binary
```bash
# Build optimized binary
make build

# Run with config
CONFIG_FILE=config-production.yaml ./bin/myapp
```

##  Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run quality checks: `make check`
5. Submit a pull request

<!-- ## 📄 License

MIT License - see LICENSE file for details. -->

##  Support

- Create an issue for bugs or feature requests
- Check the Makefile for all available commands: `make help`
- Review the configuration examples in `config-development.yaml`

---

**Happy coding! 🎉** Clone this starter kit and focus on what matters - your business logic.