# Quick Start Guide

Get your Go API up and running in 5 minutes!

## Prerequisites

- Go 1.25+ installed
- PostgreSQL 18+ (recommended) or use Docker

## Step 1: Setup

```bash
# Copy starter to your new project
cp -r starter my-go-api
cd my-go-api

# Initialize Go module (update with your module path)
go mod init github.com/yourusername/my-go-api

# Update all import paths in the codebase
# Replace "github.com/yourusername/go-api-starter" with your module path
```

## Step 2: Install Dependencies

```bash
go mod tidy
```

## Step 3: Configure Environment

```bash
# Copy example env file
cp .env.example .env

# Edit .env and set:
# - DATABASE_URL (your PostgreSQL connection string)
# - JWT_SECRET (generate with: openssl rand -base64 32)
```

## Step 4: Setup Database

```bash
# Create database
createdb go_api_starter

# Install migration tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
make migrate-up
```

## Step 5: Run the API

```bash
make run
```

## Step 6: Test the API

```bash
# Health check
curl http://localhost:8080/api/health

# Register a user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","full_name":"John Doe"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Use the token from login response
curl http://localhost:8080/api/users \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## View API Documentation

Open http://localhost:8080/swagger/index.html in your browser.

## Using Docker Instead

> **Note:** For local development, using PostgreSQL 18 directly is recommended. Docker is available as an alternative.

```bash
# Start everything with Docker
docker compose up -d --build

# Run migrations (first time)
docker compose exec api sh -c "migrate -path migrations -database \$DATABASE_URL up"
```

**Note:** Docker PostgreSQL runs on port 5434 to avoid conflict with local PostgreSQL 18.

That's it! Your API is running. 🚀

## Next Steps

- Read [Architecture](./ARCHITECTURE.md) to understand the codebase
- Read [Setup Guide](./SETUP.md) for detailed setup instructions
- Check [Contributing](../CONTRIBUTING.md) for development guidelines

