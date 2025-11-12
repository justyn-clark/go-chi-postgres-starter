.PHONY: help swagger swagger-serve build run run-dev dev stop lint lint-fix fmt vet test test-coverage test-utils bench-utils test-setup check migrate-up migrate-down migrate-create migrate-status docker-up docker-down docker-build clean

# Default target
help:
	@echo "Available commands:"
	@echo "  make run              - Run the API server"
	@echo "  make run-dev          - Run the API server with live reload (Air)"
	@echo "  make dev              - Start database and run API with live reload"
	@echo "  make stop             - Stop the running API server"
	@echo "  make build            - Build the binary"
	@echo "  make test             - Run tests"
	@echo "  make test-coverage    - Run tests with coverage report"
	@echo "  make test-setup       - Set up test database (create DB and run migrations)"
	@echo "  make test-utils       - Run utils tests (goroutines)"
	@echo "  make bench-utils      - Run utils benchmarks"
	@echo "  make lint             - Run linters"
	@echo "  make lint-fix         - Run linters and auto-fix issues"
	@echo "  make fmt              - Format code"
	@echo "  make vet              - Run go vet"
	@echo "  make check             - Run all checks (fmt, vet, lint, test)"
	@echo "  make swagger          - Generate Swagger documentation"
	@echo "  make swagger-serve    - Generate Swagger docs and start server"
	@echo "  make migrate-up       - Run database migrations"
	@echo "  make migrate-down     - Rollback last migration"
	@echo "  make migrate-create   - Create a new migration (NAME=description)"
	@echo "  make migrate-status   - Check migration status"
	@echo "  make docker-up        - Start Docker services"
	@echo "  make docker-down      - Stop Docker services"
	@echo "  make docker-build     - Build Docker image"
	@echo "  make redis-up         - Start Redis container for queue testing"
	@echo "  make redis-down       - Stop Redis container"
	@echo "  make redis-test       - Start Redis and run Redis queue tests"
	@echo "  make clean            - Clean build artifacts"

# Generate Swagger docs from code annotations
swagger:
	@echo "Generating Swagger documentation..."
	@go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o ./docs --parseDependency --parseInternal
	@echo "Fixing model names in swagger.json, swagger.yaml, and docs.go..."
	@sed -i.bak 's/github_com_yourusername_go_api_starter_cmd_api_models\./models./g' docs/swagger.json docs/swagger.yaml docs/docs.go 2>/dev/null && rm -f docs/*.bak || true
	@echo "✅ Swagger docs generated in ./docs"
	@echo "Visit http://localhost:8080/swagger/index.html after starting the server"

# Serve Swagger UI (requires docs to be generated first)
swagger-serve: swagger
	@echo "Starting Swagger UI server..."
	@echo "Open http://localhost:$${PORT:-8080}/swagger/index.html"
	@DATABASE_URL="$${DATABASE_URL:-postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable}" \
	JWT_SECRET="$${JWT_SECRET:-dev-secret-change-in-production}" \
	PORT="$${PORT:-8080}" \
	ENVIRONMENT="$${ENVIRONMENT:-development}" \
	LOG_LEVEL="$${LOG_LEVEL:-info}" \
	go run ./cmd/api

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/api ./cmd/api
	@echo "✅ Built: bin/api"

# Run the application
run:
	@echo "Starting API server..."
	@go run ./cmd/api

# Run the application with live reload (Air)
run-dev:
	@if lsof -ti:8080 > /dev/null 2>&1; then \
		echo "⚠️  Port 8080 is already in use. Attempting to stop existing processes..."; \
		$(MAKE) stop > /dev/null 2>&1 || true; \
		sleep 1; \
		if lsof -ti:8080 > /dev/null 2>&1; then \
			echo "❌ Error: Port 8080 is still in use. Please stop the process manually or use a different port."; \
			echo "   You can check what's using it with: lsof -i:8080"; \
			exit 1; \
		fi; \
	fi
	@if ! docker info > /dev/null 2>&1; then \
		echo ""; \
		echo "⚠️  Warning: Docker is not running. Database might not be available."; \
		echo "   Start Docker Desktop and run 'make docker-up' to start the database."; \
		echo ""; \
	fi
	@echo "Starting API server with live reload (Air)..."
	@AIR_CMD=$$(command -v air 2>/dev/null || echo ""); \
	if [ -z "$$AIR_CMD" ]; then \
		AIR_CMD=$$HOME/go/bin/air; \
		if [ ! -f "$$AIR_CMD" ]; then \
			echo "⚠️  Air not found. Installing..."; \
			go install github.com/air-verse/air@latest; \
			AIR_CMD=$$HOME/go/bin/air; \
		fi; \
	fi; \
	if [ ! -f "$$AIR_CMD" ]; then \
		echo "❌ Error: Air installation failed. Please install manually: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi; \
	echo "Using Air: $$AIR_CMD"; \
	DATABASE_URL="$${DATABASE_URL:-postgresql://postgres:postgres@localhost:5434/go_api_starter?sslmode=disable}" \
	JWT_SECRET="$${JWT_SECRET:-dev-secret-change-in-production}" \
	PORT="$${PORT:-8080}" \
	ENVIRONMENT="$${ENVIRONMENT:-development}" \
	LOG_LEVEL="$${LOG_LEVEL:-info}" \
	$$AIR_CMD

# Start database and run API with live reload (convenience command)
dev:
	@if ! docker info > /dev/null 2>&1; then \
		echo ""; \
		echo "❌ Error: Docker is not running."; \
		echo ""; \
		echo "Please:"; \
		echo "  1. Start Docker Desktop"; \
		echo "  2. Wait for it to fully start (check the menu bar icon)"; \
		echo "  3. Run 'make dev' again"; \
		echo ""; \
		exit 1; \
	fi
	@echo "Stopping Docker API container (if running) to free port 8080..."
	@docker compose stop api 2>/dev/null || true
	@echo "Starting database only..."
	@docker compose up -d postgres
	@echo "Waiting for database to be ready..."
	@echo "Checking database connection..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then \
			echo "✅ Database container is ready"; \
			break; \
		fi; \
		if [ $$i -eq 10 ]; then \
			echo "❌ Database container failed to start"; \
			exit 1; \
		else \
			echo "   Waiting for database container... (attempt $$i/10)"; \
			sleep 1; \
		fi; \
	done
	@echo "Testing database connection from host..."
	@if command -v psql > /dev/null 2>&1; then \
		for i in 1 2 3 4 5; do \
			if PGPASSWORD=postgres psql -h localhost -p 5434 -U postgres -d go_api_starter -c "SELECT 1;" > /dev/null 2>&1; then \
				echo "✅ Database connection successful!"; \
				break; \
			fi; \
			if [ $$i -eq 5 ]; then \
				echo "⚠️  Could not verify database connection (psql not found or connection failed)"; \
				echo "   Continuing anyway..."; \
			else \
				echo "   Testing connection... (attempt $$i/5)"; \
				sleep 1; \
			fi; \
		done; \
	else \
		echo "⚠️  psql not found, skipping connection test"; \
		echo "   Make sure database is accessible on localhost:5434"; \
	fi
	@echo "Starting API server with live reload..."
	@$(MAKE) run-dev

# Stop the running API server
stop:
	@echo "Stopping API server..."
	@if lsof -ti:8080 > /dev/null 2>&1; then \
		echo "Killing process on port 8080..."; \
		lsof -ti:8080 | xargs kill -9 2>/dev/null || true; \
		echo "✅ Stopped process on port 8080"; \
	else \
		echo "No process found on port 8080"; \
	fi
	@if pgrep -f "air" > /dev/null 2>&1; then \
		echo "Killing Air processes..."; \
		pkill -f "air" 2>/dev/null || true; \
		echo "✅ Stopped Air processes"; \
	else \
		echo "No Air processes found"; \
	fi
	@if pgrep -f "go run.*cmd/api" > /dev/null 2>&1; then \
		echo "Killing go run processes..."; \
		pkill -f "go run.*cmd/api" 2>/dev/null || true; \
		echo "✅ Stopped go run processes"; \
	else \
		echo "No go run processes found"; \
	fi
	@if pgrep -f "./tmp/main" > /dev/null 2>&1; then \
		echo "Killing tmp/main processes..."; \
		pkill -f "./tmp/main" 2>/dev/null || true; \
		echo "✅ Stopped tmp/main processes"; \
	else \
		echo "No tmp/main processes found"; \
	fi
	@if pgrep -f "bin/api" > /dev/null 2>&1; then \
		echo "Killing bin/api processes..."; \
		pkill -f "bin/api" 2>/dev/null || true; \
		echo "✅ Stopped bin/api processes"; \
	else \
		echo "No bin/api processes found"; \
	fi
	@echo "✅ Server stopped"

# Run linters (golangci-lint)
lint:
	@echo "Running linters..."
	@golangci-lint run ./cmd/... ./tests/... || (echo "⚠️  Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@echo "✅ Linting complete"

# Run linters and auto-fix issues where possible
lint-fix:
	@echo "Running linters with auto-fix..."
	@golangci-lint run --fix ./cmd/... ./tests/... || (echo "⚠️  Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@echo "✅ Linting with auto-fix complete"

# Format code (gofmt)
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✅ go vet complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "✅ Tests complete"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Run utils tests (goroutines utilities)
test-utils:
	@echo "Running utils tests..."
	@go test -v -cover ./cmd/api/utils
	@echo "✅ Utils tests complete"

# Run utils benchmarks
bench-utils:
	@echo "Running utils benchmarks..."
	@go test -bench=. -benchmem ./cmd/api/utils
	@echo "✅ Utils benchmarks complete"

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "✅ All checks passed"

# Database migrations
migrate-up:
	@echo "Running migrations..."
	@migrate -path migrations -database "$$DATABASE_URL" up || (echo "⚠️  Install migrate: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)

migrate-down:
	@echo "Rolling back migration..."
	@migrate -path migrations -database "$$DATABASE_URL" down || (echo "⚠️  Install migrate: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "❌ Error: NAME is required. Usage: make migrate-create NAME=add_users_table"; exit 1; fi
	@migrate create -ext sql -dir migrations -seq $(NAME) || (echo "⚠️  Install migrate: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)

migrate-status:
	@echo "Migration status:"
	@migrate -path migrations -database "$$DATABASE_URL" version || (echo "⚠️  Install migrate: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)

# Test database setup
test-setup:
	@echo "Setting up test database..."
	@if ! docker info > /dev/null 2>&1; then \
		echo "❌ Error: Docker is not running. Please start Docker Desktop."; \
		exit 1; \
	fi
	@if ! docker compose ps postgres | grep -q "Up"; then \
		echo "❌ Error: PostgreSQL container is not running. Run 'make docker-up' first."; \
		exit 1; \
	fi
	@echo "Creating test database (if it doesn't exist)..."
	@docker compose exec -T postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'go_api_starter_test'" | grep -q 1 || \
		docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE go_api_starter_test;" > /dev/null 2>&1 && \
		echo "✅ Test database created" || \
		echo "ℹ️  Test database already exists"
	@echo "Running migrations on test database..."
	@MIGRATE_BIN=$$(which migrate 2>/dev/null || echo "$$HOME/go/bin/migrate"); \
	if [ ! -f "$$MIGRATE_BIN" ]; then \
		echo "⚠️  migrate not found. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		MIGRATE_BIN="$$HOME/go/bin/migrate"; \
	fi; \
	$$MIGRATE_BIN -path migrations -database "postgresql://postgres:postgres@localhost:5434/go_api_starter_test?sslmode=disable" up || \
		(echo "⚠️  Failed to run migrations. Install migrate: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	@echo "✅ Test database setup complete"

# Docker commands
docker-up:
	@if ! docker info > /dev/null 2>&1; then \
		echo "❌ Error: Docker is not running. Please start Docker Desktop."; \
		exit 1; \
	fi
	@echo "Starting Docker services..."
	@docker compose up -d postgres
	@echo "✅ Database service started"
	@echo "PostgreSQL: localhost:5434"
	@echo "Note: API will run locally (not in Docker) when using make run-dev"
	@echo "Note: Docker API container is not started (use 'docker compose up -d' to start it)"

docker-down:
	@echo "Stopping Docker services..."
	@docker compose down
	@echo "✅ Services stopped"

docker-build:
	@echo "Building Docker image..."
	@docker compose build
	@echo "✅ Docker image built"

# Redis commands for queue testing
redis-up: ## Start Redis container for queue testing
	@echo "Starting Redis container..."
	@docker run -d -p 6379:6379 --name go-chi-postgres-starter-redis redis:7-alpine 2>/dev/null || \
		docker start go-chi-postgres-starter-redis 2>/dev/null || \
		echo "Redis container already running or failed to start"
	@echo "✅ Redis available at redis://localhost:6379"
	@echo "Test with: go test ./cmd/api/queue -v -run TestRedisQueue"

redis-down: ## Stop Redis container
	@echo "Stopping Redis container..."
	@docker stop go-chi-postgres-starter-redis 2>/dev/null || echo "Redis container not running"
	@docker rm go-chi-postgres-starter-redis 2>/dev/null || echo "Redis container not found"

redis-test: redis-up ## Start Redis and run Redis queue tests
	@echo "Waiting for Redis to be ready..."
	@sleep 2
	@go test ./cmd/api/queue -v -run TestRedisQueue
	@$(MAKE) redis-down

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ coverage.out coverage.html
	@echo "✅ Cleaned"
