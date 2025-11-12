# Go Chi Router + PostgreSQL Starter

<!-- markdownlint-disable MD033 -->
<p align="center">
  <img src="https://go.dev/images/gophers/ladder.svg" alt="Go Gopher" width="200"/>
</p>
<!-- markdownlint-enable MD033 -->

A modern Go REST API starter with Chi Router and PostgreSQL 18

A clean, minimal Go REST API starter template with:

* **Chi Router** - Lightweight, standard library compatible router
* **pgx/v5** - High-performance PostgreSQL driver
* **golang-migrate** - Database migrations
* **JWT Authentication** - Secure token-based auth system
* **Zerolog** - Structured logging

## Features

![CI](https://github.com/justyn-clark/go-chi-postgres-starter/workflows/CI/badge.svg)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen)
![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

> **Note for template users:** After forking this repository, update the CI badge URL above (line 22) to point to your repository: `https://github.com/YOUR_USERNAME/YOUR_REPO_NAME/workflows/CI/badge.svg`

* ✅ **Production Ready** - Complete CI/CD pipeline with GitHub Actions
* ✅ **Type Safe** - Full Go type safety and validation
* ✅ **Clean Architecture** - Handler-Service-Repository pattern
* ✅ **PostgreSQL 18** - Optimized for latest PostgreSQL features
* ✅ **Docker Support** - Optional Docker configuration for containerized development
* ✅ **Database Migrations** - golang-migrate integration for schema management
* ✅ **JWT Authentication** - Secure token-based auth system
* ✅ **OpenAPI Documentation** - Auto-generated Swagger UI
* ✅ **Testing Setup** - Comprehensive test configuration
* ✅ **Built-in Utilities** - Pagination, validation, context helpers, custom errors, goroutines patterns
* ✅ **Rate Limiting** - IP-based rate limiting with token bucket algorithm (enabled by default)
* ✅ **Request Body Limits** - 1MB body size limit (DoS protection)
* ✅ **Prometheus Metrics** - `/metrics` endpoint for observability
* ✅ **Database Pool Config** - Production-ready connection pool settings
* ✅ **Queue System** - Pluggable background job processing (Redis/in-memory)
* ✅ **Deployment Ready** - Dockerfile + Fly.io & Railway configs
* ✅ **Optional Middleware** - CORS (ready to enable)
* ✅ **Enhanced Health Checks** - Database status monitoring
* ✅ **Extensible** - Easy to adapt for GraphQL, gRPC, microservices, CLI tools

## Why Go? Versatility & Use Cases

This starter template demonstrates Go's strength: **one language, many solutions**. The same clean architecture patterns can adapt to:

* **REST APIs** (current setup) - Traditional HTTP/JSON APIs
* **Microservices** - Distributed systems with service discovery
* **GraphQL APIs** - Replace handlers with GraphQL resolvers, keep services/repositories
* **gRPC Services** - Replace HTTP handlers with gRPC handlers, same business logic
* **Event-Driven Systems** - Add message queues, keep the service layer
* **CLI Tools** - Add command structure alongside the API
* **Background Workers** - Process jobs using the same service layer
* **WebSocket Servers** - Real-time connections with goroutines

**Go's Philosophy:** Simple, fast, and versatile. This starter gives you a solid foundation that evolves with your needs while keeping code clean, maintainable, and performant.

## API Documentation

Interactive API documentation is available at `/swagger/index.html` after starting the server.

## Prerequisites

Before you begin, ensure you have:

* **Go 1.25+** - [Download and install Go](https://golang.org/doc/install)
* **PostgreSQL 18+** - [Install PostgreSQL](https://www.postgresql.org/download/) (recommended) or use Docker
* **Git** - For cloning the repository

Verify your installation:

```bash
go version    # Should show go1.25.x or higher
psql --version  # Should show PostgreSQL 18.x or higher
```

## Quick start

### 1) After Forking/Cloning (First Time Setup)

If you've forked or cloned this template, update these placeholders:

1. **Update module name in `go.mod`**: Change `github.com/yourusername/go-chi-postgres-starter` to your module path
2. **Update CI badge in `README.md`**: Replace `justyn-clark/go-chi-postgres-starter` in the CI badge URL (line 22) with your GitHub username and repo name
3. **Search and replace**: Use your IDE's find/replace to update `yourusername` → `your-github-username` throughout the codebase

### 2) Clone and setup

```bash
git clone https://github.com/justyn-clark/go-chi-postgres-starter.git
cd go-chi-postgres-starter
go mod tidy
```

**Note about `go.mod` and `go.sum`:**

* `go.mod` - Defines your module and its direct dependencies (you can edit this)
* `go.sum` - Auto-generated checksum file for security and reproducibility (never edit manually)
  * Created/updated automatically by `go mod tidy`, `go build`, `go test`, etc.
  * Contains cryptographic hashes (checksums) to verify dependency integrity
  * **Always commit `go.sum` to version control** - it ensures everyone gets the same dependency versions

### 3) Create database

```bash
# Create the database using createdb (comes with PostgreSQL)
createdb go_api_starter

# Alternative: Use psql if createdb is not available
# psql -d postgres -c 'CREATE DATABASE go_api_starter;'

# If needed, create postgres user (for compatibility with docker-compose setup)
psql -d postgres -c "CREATE USER postgres WITH SUPERUSER PASSWORD 'postgres';" 2>/dev/null || echo "User postgres already exists"
```

**Note:** `createdb` is a PostgreSQL utility command that comes bundled with PostgreSQL installations. If you get a "command not found" error, make sure PostgreSQL 18 is in your PATH (see Prerequisites) or use the `psql` alternative shown above.

### 4) Run migrations

```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
make migrate-up
```

### 5) Start the API

```bash
make run
```

### Notes

* App uses **pgx/v5** for PostgreSQL connection pooling
* Migrations use **golang-migrate** for schema management
* Default entity: `User` with endpoints:  
  * `POST /api/auth/register` - Register new user  
  * `POST /api/auth/login` - Login and get JWT token  
  * `POST /api/users` - Create user (admin)  
  * `GET /api/users` - List all users  
  * `GET /api/users/{id}` - Get user by ID

## Environment Setup

Create a `.env` file in the project root (optional - the app will load it automatically if present):

```bash
# Copy the example file
cp .env.example .env

# Edit .env with your configuration
```

Or create `.env` manually with:

```env
# Database connection (PostgreSQL 18 on localhost:5432)
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable

# JWT Secret (generate with command below)
JWT_SECRET=your-very-secure-random-secret-key-here

# Server configuration
PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info

# Rate Limiting (enabled by default)
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20
```

**Note:** You can also set these as environment variables instead of using a `.env` file. See `.env.example` for all available configuration options.

Generate JWT secret:

```bash
openssl rand -base64 32
```

## Go Module Management

### Understanding `go.mod` and `go.sum`

**`go.mod`** - Your module definition file:

* Lists direct dependencies and their versions
* Can be manually edited (though `go mod tidy` is recommended)
* Module path format: `github.com/username/repo` (matches the repository URL)

**`go.sum`** - Auto-generated checksum file:

* **Never edit manually** - automatically maintained by Go
* Created/updated by: `go mod tidy`, `go build`, `go test`, `go get`, etc.
* Contains cryptographic checksums (hashes) for all dependencies
* **Purpose:**
  * **Security**: Verifies downloaded code hasn't been tampered with
  * **Reproducibility**: Ensures everyone gets identical dependency versions
  * **Integrity**: Detects if dependencies have been modified

**What is a checksum?**
A checksum is a cryptographic hash (like a fingerprint) of a file. It's a fixed-size string generated from the file's contents. If the file changes even slightly, the checksum changes completely. Go uses checksums to verify that the exact same code you downloaded before is what you're getting now.

**Why do dependencies use `github.com`?**

* Go modules use the repository URL as the module path
* Most Go packages are hosted on GitHub, so module paths match GitHub URLs
* Examples: `github.com/go-chi/chi/v5`, `github.com/jackc/pgx/v5`
* Other hosts work too: `gitlab.com`, `gitea.com`, custom domains, etc.
* The module path tells Go where to download the package from

**Best Practice:** Always commit both `go.mod` and `go.sum` to version control.

## Migration commands

```bash
# Generate a new migration (after editing models)
make migrate-create NAME=add_users_table

# Apply migrations
make migrate-up

# Rollback one
make migrate-down

# Check status
make migrate-status
```

## Testing the API

```bash
# Register a new user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","full_name":"Test User","password":"password123"}'

# Login to get JWT token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# List users (requires authentication)
curl -X GET http://localhost:8080/api/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Docker Development

> **Note:** For local development, we recommend using PostgreSQL 18 directly (see Quick Start above). Docker is available as an alternative for containerized environments.

```bash
# Start database and API
docker compose up -d --build

# View logs
docker compose logs -f api

# Stop everything
docker compose down -v
```

**Connection Details:**

* API: <http://localhost:8080>
* Database: `localhost:5434` (postgres/postgres)
  * **Note:** Port 5434 is used to avoid conflict with local PostgreSQL 18 on port 5432
  * Update `DATABASE_URL` to `postgresql://postgres:postgres@localhost:5434/go_api_starter?sslmode=disable` when using Docker

## Built-in Utilities & Helpers

This starter includes production-ready utilities ready to use:

### Pagination (`cmd/api/utils/pagination.go`)

```go
import "github.com/yourusername/go-chi-postgres-starter/cmd/api/utils"

// Parse pagination from query params
params := utils.ParsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("limit"))

// Create pagination response
pagination := utils.NewPaginationResponse(params.Page, params.Limit, total)
```

### Validation (`cmd/api/utils/validation.go`)

```go
// Email validation
if !utils.ValidateEmail(email) {
    respondError(w, http.StatusBadRequest, "invalid email format")
    return
}

// Password validation
if !utils.ValidatePassword(password, 8) {
    respondError(w, http.StatusBadRequest, "password too weak")
    return
}
```

### Context Helpers (`cmd/api/utils/context.go`)

```go
// Get user ID from context (set by auth middleware)
userID, err := utils.GetUserID(r.Context())
if err != nil {
    respondError(w, http.StatusUnauthorized, "user not authenticated")
    return
}
```

### Custom Error Types (`cmd/api/errors/errors.go`)

```go
import "github.com/yourusername/go-chi-postgres-starter/cmd/api/errors"

// Use typed API errors
return errors.WrapAPIError(http.StatusNotFound, "user not found", err)

// Check error type
if errors.IsAPIError(err) {
    apiErr := errors.GetAPIError(err)
    respondError(w, apiErr.Code, apiErr.Message)
}
```

### Goroutines Utilities (`cmd/api/utils/goroutines.go`)

Production-ready patterns for concurrent operations:

```go
import "github.com/yourusername/go-chi-postgres-starter/cmd/api/utils"

// Fire-and-forget: Send email in background
utils.BackgroundTask(r.Context(), func() error {
    return emailService.SendWelcomeEmail(user.Email)
})

// With retry: Process payment with automatic retries
utils.BackgroundTaskWithRetry(r.Context(), func() error {
    return paymentService.ProcessPayment(orderID, amount)
}, 3, 5*time.Second)

// Batch processing: Process multiple items concurrently
err := utils.ProcessConcurrently(r.Context(), users, 5, func(user User) error {
    return processUser(user)
})

// Scheduled tasks: Cleanup expired tokens every hour
utils.PeriodicTask(ctx, func() error {
    return tokenService.CleanupExpiredTokens()
}, 1*time.Hour)
```

**Common Use Cases:**

* **BackgroundTask**: Welcome emails, audit logging, webhooks
* **BackgroundTaskWithRetry**: Payments, critical notifications, external API sync
* **ProcessConcurrently**: Bulk imports, mass emails, report generation
* **PeriodicTask**: Token cleanup, daily reports, health checks
* **WaitGroupWithTimeout**: Coordinating multiple API calls
* **FanOutFanIn**: Parallel processing with result collection

See [Goroutines Guide](./docs/GOROUTINES.md) for complete documentation and examples.

### Queue System (`cmd/api/queue/`)

Pluggable queue system for background job processing. Supports Redis (production) or in-memory (development).

**Default:** Simple Redis Lists (fast, lightweight, no extra dependencies)

**Optional:** [Asynq](https://pkg.go.dev/github.com/hibiken/asynq) implementation available for advanced features (priorities, scheduling, status tracking, dead letter queue, exponential backoff retries, job deduplication, rate limiting). Uses build tags - install with `go get github.com/hibiken/asynq` and build with `-tags asynq`.

**Features:**

* Interface-based design - plug in any queue (Redis, RabbitMQ, SQS, etc.)
* Redis implementation included (default)
* In-memory queue for development/testing
* Worker pool pattern with configurable concurrency
* Automatic retry logic
* Job acknowledgment and rejection
* **Optional Asynq** - Advanced features when needed (install: `go get github.com/hibiken/asynq`)

**Cost:** **FREE** - Uses Redis (free if self-hosted). No additional costs for the queue system itself.

**Quick Start:**

```go
import "github.com/yourusername/go-chi-postgres-starter/cmd/api/queue"

// Initialize queue (Redis for production, in-memory for dev)
var q queue.Queue
if cfg.QueueURL != "" {
    q, _ = queue.NewRedisQueue(cfg.QueueURL)
} else {
    q = queue.NewMemoryQueue()
}
defer q.Close()

// Enqueue a job
q.Enqueue(ctx, "emails", "send_welcome", EmailJob{
    To: user.Email,
    Subject: "Welcome!",
})

// Start worker
worker := queue.NewWorker(q, "emails", handleEmailJob, 5)
worker.Start(ctx)
```

**Implement Your Own Queue:**

```go
type MyQueue struct {
    // Your implementation
}

func (m *MyQueue) Enqueue(ctx context.Context, queueName string, jobType string, payload any) error {
    // Your implementation
}
// ... implement other Queue interface methods
```

See `cmd/api/queue/example.go` for complete examples.

### Optional Middleware (Ready to Enable)

**CORS** (`cmd/api/middleware/cors.go`) - Uncomment in `routes.go` to enable

## Deployment

The starter includes production-ready deployment configurations for popular platforms.

### Quick Deploy

**Fly.io:**

```bash
fly launch
fly postgres create
fly redis create  # Optional: For queue system (free tier available)
fly deploy
```

**Railway:**

```bash
railway init
railway add postgresql
railway add redis  # Optional: For queue system (free tier available)
railway up
```

**Note:** Queue system is optional. If `QUEUE_URL` is not set, the app uses in-memory queue (development only). Redis is free if self-hosted.

### Detailed Guides

* **[Deployment Guide](./docs/DEPLOYMENT.md)** - Complete deployment instructions for Fly.io, Railway, and Docker platforms
* Includes Redis configuration, environment variables, health checks, and troubleshooting

**Key Points:**

* Dockerfile included (multi-stage build, ~20MB final image)
* Redis is a **separate service** (not mounted in container)
* Health check endpoint: `/api/health`
* Automatic SSL/TLS on Fly.io and Railway

## Development Commands

```bash
# Format code
make fmt

# Lint code
make lint

# Run tests
make test

# Generate Swagger docs
make swagger
```

## Authentication

The API includes JWT-based authentication:

1. **Register**: `POST /api/auth/register` with email, password, and optional full_name
2. **Login**: `POST /api/auth/login` with email and password to get JWT token
3. **Protected routes**: Include `Authorization: Bearer <token>` header

All user management endpoints require authentication.

## Documentation

### User Guides (GitHub)

* [Quick Start Guide](./docs/QUICKSTART.md) - Get up and running in 5 minutes
* [Setup Guide](./docs/SETUP.md) - Detailed setup instructions
* [Testing Guide](./docs/TESTING.md) - How to run and write tests
* [Goroutines Guide](./docs/GOROUTINES.md) - Concurrent operations and goroutine patterns
* [Postman Setup Guide](./docs/POSTMAN_SETUP.md) - How to import and use the API in Postman
* [Postman Get Users Guide](./docs/POSTMAN_GET_USERS.md) - Quick guide for GET /api/users endpoint
* [Deployment Guide](./docs/DEPLOYMENT.md) - Deploy to Fly.io, Railway, or Docker platforms
* [Contributing](./CONTRIBUTING.md) - Development guidelines

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

**Justyn Clark**

* GitHub: [@justyn-clark](https://github.com/justyn-clark)
* Email: <justyn-clark@users.noreply.github.com>

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

If you found this starter template helpful, please give it a ⭐ star!

---

Made with ❤️ for the Go community
