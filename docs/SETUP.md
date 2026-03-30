# Setup Guide

## Prerequisites

- **Go 1.25+**
- **PostgreSQL** for local development
- **Docker** (optional)

> The repo docs recommend PostgreSQL 18 for local development, but the checked-in Docker Compose file and GitHub Actions workflow currently use PostgreSQL 16.

## Option 1: Local development

### 1. Clone the repository

```bash
git clone https://github.com/justyn-clark/go-chi-postgres-starter.git
cd go-chi-postgres-starter
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Set up environment variables

```bash
cp .env.example .env
```

Update `.env` as needed.

### 4. Create the database

```bash
createdb go_api_starter
```

Or:

```bash
psql -d postgres -c 'CREATE DATABASE go_api_starter;'
```

### 5. Install migrate

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 6. Run migrations

```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
make migrate-up
```

### 7. Start the server

```bash
make run
```

### 8. Access the app

- API: <http://localhost:8080>
- Health: <http://localhost:8080/api/health>
- Swagger UI: <http://localhost:8080/swagger/index.html>
- Metrics: <http://localhost:8080/metrics>

## Option 2: Docker development

### Start services

```bash
docker compose up -d --build
```

This starts:

- API on `localhost:8080`
- Postgres on `localhost:5434`
- Redis on `localhost:6379`

### Database URL from your host shell

```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:5434/go_api_starter?sslmode=disable"
```

### Migrations with Docker

The Compose setup does **not** automatically run migrations for you.

Recommended approach:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
export DATABASE_URL="postgresql://postgres:postgres@localhost:5434/go_api_starter?sslmode=disable"
make migrate-up
```

## JWT secret

Generate one for non-default usage:

```bash
openssl rand -base64 32
```

Add it to `.env`:

```text
JWT_SECRET=your-generated-secret-here
```

## Authorization model

Current route protection in `cmd/api/routes.go`:

### Public

- `GET /api/health`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/request-password-reset`
- `POST /api/auth/reset-password`

### Authenticated

- `POST /api/auth/change-password`
- `GET /api/users/me`

### Owner or admin

- `GET /api/users/{id}`
- `PUT /api/users/{id}`

### Admin only

- `GET /api/users`
- `POST /api/users`
- `DELETE /api/users/{id}`
- `PUT /api/users/{id}/role`

## Swagger generation

```bash
make swagger
```

Then visit <http://localhost:8080/swagger/index.html>.

## Testing

```bash
make test
make test-coverage
```

## Helpful dev commands

```bash
make run
make run-dev
make dev
make stop
make fmt
make vet
make lint
make migrate-status
```

## Troubleshooting

### Database connection issues

- Ensure PostgreSQL is running
- Check `DATABASE_URL`
- Verify the target database exists

### Port 8080 already in use

- Stop the conflicting process
- Or run `make stop`

### Docker DB vs local DB mismatch

- Local Postgres examples use port `5432`
- Docker Postgres is exposed on host port `5434`
