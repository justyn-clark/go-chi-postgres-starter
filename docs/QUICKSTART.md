# Quick Start Guide

Get the starter running locally with the current route and auth model.

## Prerequisites

- Go 1.25+
- PostgreSQL installed locally, or Docker
- git

> Local docs in this repo recommend PostgreSQL 18. The checked-in Docker Compose and CI workflows currently use PostgreSQL 16.

## 1. Clone the repo

```bash
git clone https://github.com/justyn-clark/go-chi-postgres-starter.git
cd go-chi-postgres-starter
```

## 2. Install dependencies

```bash
go mod tidy
```

## 3. Configure environment

```bash
cp .env.example .env
```

Minimum local settings:

```env
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable
JWT_SECRET=dev-secret-change-in-production
PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info
```

Generate a real JWT secret if needed:

```bash
openssl rand -base64 32
```

## 4. Create the database

```bash
createdb go_api_starter
```

Or:

```bash
psql -d postgres -c 'CREATE DATABASE go_api_starter;'
```

## 5. Install migrate and run migrations

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
make migrate-up
```

## 6. Start the API

```bash
make run
```

## 7. Verify the server

```bash
curl http://localhost:8080/api/health
```

## 8. Register and log in

Register:

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","full_name":"John Doe"}'
```

Login:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

## 9. Call an authenticated endpoint

Use the token from login against your own profile endpoint:

```bash
curl http://localhost:8080/api/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

> `GET /api/users` is admin-only in the current codebase, so a freshly registered normal user should use `/api/users/me` or `/api/users/{their-id}`.

## Swagger and metrics

- Swagger UI: <http://localhost:8080/swagger/index.html>
- Metrics: <http://localhost:8080/metrics>

## Docker option

```bash
docker compose up -d --build
```

When using the Docker Postgres service from your host shell, use:

```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:5434/go_api_starter?sslmode=disable"
```

If you need migrations after the stack is up, run them from a host shell with that `DATABASE_URL`, or install `migrate` in the API image yourself before using in-container commands.

## Next steps

- [Setup Guide](./SETUP.md)
- [Authentication Guide](./AUTHENTICATION.md)
- [Testing Guide](./TESTING.md)
