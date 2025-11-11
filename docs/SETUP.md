# Setup Guide

## Prerequisites

- **Go 1.25+** - [Install Go](https://golang.org/doc/install)
- **PostgreSQL 16+** - [Install PostgreSQL](https://www.postgresql.org/download/)
- **Docker & Docker Compose** (optional, for containerized development)

## Quick Start

### Option 1: Local Development

1. **Clone the repository**

   ```bash
   git clone https://github.com/yourusername/go-api-starter.git
   cd go-api-starter
   ```

2. **Install Go dependencies**

   ```bash
   go mod tidy
   ```

3. **Set up environment variables**

   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Create database**

   ```bash
   createdb go_api_starter
   # Or: psql -c 'CREATE DATABASE go_api_starter;'
   ```

5. **Install migration tool**

   ```bash
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

6. **Run migrations**

   ```bash
   export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
   make migrate-up
   ```

7. **Start the server**

   ```bash
   make run
   ```

8. **Access the API**
   - API: <http://localhost:8080>
   - Swagger UI: <http://localhost:8080/swagger/index.html>

### Option 2: Docker Development

1. **Start services**

   ```bash
   docker compose up -d --build
   ```

2. **Run migrations** (first time only)

   ```bash
   docker compose exec api sh -c "migrate -path migrations -database \$DATABASE_URL up"
   ```

3. **Access the API**
   - API: <http://localhost:8080>
   - Swagger UI: <http://localhost:8080/swagger/index.html>

## Generate JWT Secret

For production, generate a secure JWT secret:

```bash
openssl rand -base64 32
```

Add it to your `.env` file:

```text
JWT_SECRET=your-generated-secret-here
```

## Database Migrations

### Create a new migration

```bash
make migrate-create NAME=add_user_table
```

This creates two files:

- `migrations/XXXXXX_add_user_table.up.sql` - Migration to apply
- `migrations/XXXXXX_add_user_table.down.sql` - Migration to rollback

### Run migrations

```bash
make migrate-up
```

### Rollback last migration

```bash
make migrate-down
```

### Check migration status

```bash
make migrate-status
```

## Generate Swagger Documentation

```bash
make swagger
```

This generates OpenAPI documentation from code annotations. Visit <http://localhost:8080/swagger/index.html> after starting the server.

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

## Production Deployment

### Railway

1. Connect your GitHub repository
2. Add PostgreSQL service
3. Set environment variables:
   - `DATABASE_URL` (auto-provided by Railway)
   - `JWT_SECRET` (generate with `openssl rand -base64 32`)
   - `ENVIRONMENT=production`
4. Deploy!

### Docker

```bash
# Build image
docker build -t go-api-starter .

# Run container
docker run -p 8080:8080 \
  -e DATABASE_URL=postgresql://... \
  -e JWT_SECRET=... \
  go-api-starter
```

## Troubleshooting

### Database connection issues

- Ensure PostgreSQL is running
- Check `DATABASE_URL` format: `postgresql://user:password@host:port/dbname?sslmode=disable`
- Verify database exists: `psql -l`

### Migration errors

- Ensure migrations are in correct order
- Check database connection
- Verify migration files are valid SQL

### Port already in use

- Change `PORT` in `.env` file
- Or stop the process using port 8080
