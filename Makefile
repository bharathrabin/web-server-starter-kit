# Variables
APP_NAME := myapp
DOCKER_IMAGE := $(APP_NAME):latest
DOCKER_COMPOSE_FILE := docker-compose.yml
DOCKER_COMPOSE_DEV_FILE := docker-compose.dev.yml
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
CONFIG_FILE := config-development.yaml
MIGRATIONS_DIR := scripts/migrations

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help build run test clean docker-build docker-run docker-stop docker-clean compose-up compose-down compose-logs compose-restart lint fmt vet deps migrate seed db-reset dev hot-reload proto gen third-party run-grpc run-http install-deps

# Default target
.DEFAULT_GOAL := help

## Help command
help: ## Show this help message
	@echo "$(BLUE)Available commands:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Go Development
build: ## Build the Go application
	@echo "$(YELLOW)Building Go application...$(NC)"
	@go build -ldflags="-w -s" -o bin/$(APP_NAME) .
	@echo "$(GREEN)Build completed: bin/$(APP_NAME)$(NC)"

run: ## Run the Go application locally
	@echo "$(YELLOW)Running Go application...$(NC)"
	@go run .

dev: ## Run the application in development mode with auto-reload
	@echo "$(YELLOW)Starting development server with auto-reload...$(NC)"
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/cosmtrek/air@latest; }
	@air

dev-proto: clean gen build ## Clean, generate protobuf code, and build (your original dev command)
	@echo "$(GREEN)Development build with protobuf generation completed$(NC)"

test: ## Run tests
	@echo "$(YELLOW)Running tests...$(NC)"
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

bench: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@go test -bench=. ./...

## Code Quality
lint: ## Run golangci-lint
	@echo "$(YELLOW)Running linter...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@golangci-lint run

fmt: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	@go vet ./...

tidy: ## Tidy go modules
	@echo "$(YELLOW)Tidying go modules...$(NC)"
	@go mod tidy
	@echo "$(GREEN)Modules tidied$(NC)"

deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@go mod download
	@echo "$(GREEN)Dependencies downloaded$(NC)"

install-deps: ## Install required protobuf and development tools
	@echo "$(YELLOW)Installing protobuf and development tools...$(NC)"
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "$(GREEN)All development tools installed$(NC)"

## Protobuf/gRPC Commands
third-party: ## Download third party protobuf files (googleapis)
	@echo "$(YELLOW)Downloading third party protobuf files...$(NC)"
	@mkdir -p third_party/google/api
	@curl -o third_party/google/api/annotations.proto \
		https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
	@curl -o third_party/google/api/http.proto \
		https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto
	@echo "$(GREEN)Third party protobuf files downloaded$(NC)"

proto: ## Generate protobuf code
	@echo "$(YELLOW)Generating protobuf code...$(NC)"
	@mkdir -p gen/api/v1
	@protoc --proto_path=proto \
		--proto_path=third_party \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=gen --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=gen --openapiv2_opt=logtostderr=true \
		proto/models/v1/models.proto
	@echo "$(GREEN)Protobuf code generated$(NC)"

gen: third-party proto ## Download third-party protos and generate code

run-grpc: ## Run the gRPC server
	@echo "$(YELLOW)Starting gRPC server...$(NC)"
	@go run cmd/grpc/main.go

run-http: ## Run the HTTP/REST server
	@echo "$(YELLOW)Starting HTTP/REST server...$(NC)"
	@go run cmd/http/main.go

## Docker Commands
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE)$(NC)"

docker-run: ## Run Docker container
	@echo "$(YELLOW)Running Docker container...$(NC)"
	@docker run --rm -p 3000:3000 -v $(PWD)/$(CONFIG_FILE):/app/config.yaml:ro $(DOCKER_IMAGE)

docker-run-bg: ## Run Docker container in background
	@echo "$(YELLOW)Running Docker container in background...$(NC)"
	@docker run -d --name $(APP_NAME) -p 3000:3000 -v $(PWD)/$(CONFIG_FILE):/app/config.yaml:ro $(DOCKER_IMAGE)

docker-stop: ## Stop running Docker container
	@echo "$(YELLOW)Stopping Docker container...$(NC)"
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true
	@echo "$(GREEN)Container stopped$(NC)"

docker-clean: ## Clean Docker images and containers
	@echo "$(YELLOW)Cleaning Docker resources...$(NC)"
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true
	@docker rmi $(DOCKER_IMAGE) || true
	@docker system prune -f
	@echo "$(GREEN)Docker resources cleaned$(NC)"

docker-logs: ## Show Docker container logs
	@docker logs -f $(APP_NAME)

## Docker Compose Commands
compose-up: ## Start all services with docker-compose
	@echo "$(YELLOW)Starting services with docker-compose...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "$(GREEN)Services started$(NC)"

compose-up-build: ## Start services and rebuild images
	@echo "$(YELLOW)Starting services and rebuilding...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d --build
	@echo "$(GREEN)Services started with fresh build$(NC)"

compose-down: ## Stop all services
	@echo "$(YELLOW)Stopping services...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)Services stopped$(NC)"

compose-down-volumes: ## Stop services and remove volumes
	@echo "$(YELLOW)Stopping services and removing volumes...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down -v
	@echo "$(GREEN)Services stopped and volumes removed$(NC)"

compose-logs: ## Show logs from all services
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

compose-logs-app: ## Show logs from app service only
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f app

compose-restart: ## Restart all services
	@echo "$(YELLOW)Restarting services...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) restart
	@echo "$(GREEN)Services restarted$(NC)"

compose-restart-app: ## Restart only the app service
	@echo "$(YELLOW)Restarting app service...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) restart app
	@echo "$(GREEN)App service restarted$(NC)"

compose-ps: ## Show running services
	@docker-compose -f $(DOCKER_COMPOSE_FILE) ps

compose-exec-app: ## Execute command in app container (usage: make compose-exec-app CMD="bash")
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec app $(CMD)

compose-exec-db: ## Execute command in postgres container
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgres psql -U postgres -d myapp_dev

## Database Commands
build-migrate: ## Build migration CLI tool
	@echo "$(YELLOW)Building migration tool...$(NC)"
	@go build -o bin/migrate cmd/migrator/main.go
	@echo "$(GREEN)Migration tool built: bin/migrate$(NC)"

db-migrate: ## Run database migrations
	@echo "$(YELLOW)Running database migrations...$(NC)"
	@make build-migrate
	@CONFIG_FILE=$(CONFIG_FILE) ./bin/migrate -command=up
	@echo "$(GREEN)Database migrations completed$(NC)"

db-migrate-down: ## Rollback last database migration
	@echo "$(YELLOW)Rolling back last database migration...$(NC)"
	@make build-migrate
	@CONFIG_FILE=$(CONFIG_FILE) ./bin/migrate -command=down
	@echo "$(GREEN)Database migration rolled back$(NC)"

db-migrate-status: ## Show migration status
	@echo "$(YELLOW)Checking migration status...$(NC)"
	@make build-migrate
	@CONFIG_FILE=$(CONFIG_FILE) ./bin/migrate -command=status

db-migrate-reset: ## Reset all migrations (DANGEROUS - drops all data)
	@echo "$(RED)WARNING: This will reset all migrations and drop all data!$(NC)"
	@make build-migrate
	@CONFIG_FILE=$(CONFIG_FILE) ./bin/migrate -command=reset

db-migrate-docker: ## Run migrations inside docker container
	@echo "$(YELLOW)Running database migrations in docker...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec app sh -c "go run cmd/migrate/main.go -command=up"

db-seed: ## Seed database with test data
	@echo "$(YELLOW)Seeding database...$(NC)"
	@go run cmd/seed/main.go || echo "$(RED)Seed command not found - create cmd/seed/main.go$(NC)"

db-reset: ## Reset database (reset migrations, migrate up, seed)
	@echo "$(YELLOW)Resetting database completely...$(NC)"
	@make db-migrate-reset
	@make db-migrate
	@make db-seed
	@echo "$(GREEN)Database reset completed$(NC)"

db-backup: ## Backup database
	@echo "$(YELLOW)Creating database backup...$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgres pg_dump -U postgres myapp_dev > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)Database backup created$(NC)"

db-local-migrate: ## Run migrations against local database (not docker)
	@echo "$(YELLOW)Running migrations against local database...$(NC)"
	@make build-migrate
	@CONFIG_FILE=config-development.yaml ./bin/migrate -command=up

## Development Workflow
setup: ## Initial project setup
	@echo "$(YELLOW)Setting up project...$(NC)"
	@make install-deps
	@make deps
	@make gen
	@mkdir -p $(MIGRATIONS_DIR)
	@make compose-up
	@sleep 5
	@make db-migrate
	@make db-seed
	@echo "$(GREEN)Project setup completed$(NC)"

dev-start: ## Start development environment
	@echo "$(YELLOW)Starting development environment...$(NC)"
	@make compose-up
	@echo "$(GREEN)Development environment started$(NC)"
	@echo "$(BLUE)App: http://localhost:3000$(NC)"
	@echo "$(BLUE)Graphite: http://localhost:8080$(NC)"
	@echo "$(BLUE)PostgreSQL: localhost:5432$(NC)"

dev-stop: ## Stop development environment
	@make compose-down

dev-restart: ## Restart development environment
	@make compose-down
	@make compose-up

dev-fresh: ## Fresh development environment (reset DB and restart)
	@echo "$(YELLOW)Starting fresh development environment...$(NC)"
	@make compose-down-volumes
	@make compose-up-build
	@sleep 5
	@make db-migrate
	@make db-seed
	@echo "$(GREEN)Fresh development environment ready$(NC)"

create-migration: ## Create a new migration file (usage: make create-migration NAME="create_users_table")
	@if [ -z "$(NAME)" ]; then echo "$(RED)Usage: make create-migration NAME=\"migration_name\"$(NC)"; exit 1; fi
	@mkdir -p $(MIGRATIONS_DIR)
	@NEXT_NUM=$(ls $(MIGRATIONS_DIR)/*.sql 2>/dev/null | wc -l | awk '{printf "%03d", $1+1}'); \
	FILENAME="$(MIGRATIONS_DIR)/${NEXT_NUM}_$(NAME).sql"; \
	echo "-- +migrate Up" > $FILENAME; \
	echo "" >> $FILENAME; \
	echo "-- Your UP migration SQL here" >> $FILENAME; \
	echo "" >> $FILENAME; \
	echo "-- +migrate Down" >> $FILENAME; \
	echo "" >> $FILENAME; \
	echo "-- Your DOWN migration SQL here" >> $FILENAME; \
	echo "$(GREEN)Created migration: $FILENAME$(NC)"

## Utility Commands
clean: ## Clean build artifacts and generated files
	@echo "$(YELLOW)Cleaning build artifacts and generated files...$(NC)"
	@rm -rf bin/
	@rm -rf gen/
	@rm -rf third_party/
	@rm -f coverage.out coverage.html
	@go clean -cache -modcache -testcache
	@echo "$(GREEN)Cleaned$(NC)"

check: ## Run all checks (fmt, vet, lint, test)
	@echo "$(YELLOW)Running all checks...$(NC)"
	@make fmt
	@make vet
	@make lint
	@make test
	@echo "$(GREEN)All checks passed$(NC)"

install-tools: ## Install all development tools (deprecated - use install-deps)
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@make install-deps
	@echo "$(GREEN)Development tools installed$(NC)"

status: ## Show project status
	@echo "$(BLUE)Project Status:$(NC)"
	@echo "$(YELLOW)Docker Images:$(NC)"
	@docker images | grep $(APP_NAME) || echo "No images found"
	@echo "$(YELLOW)Running Containers:$(NC)"
	@docker ps --filter "name=$(APP_NAME)"
	@echo "$(YELLOW)Docker Compose Services:$(NC)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) ps