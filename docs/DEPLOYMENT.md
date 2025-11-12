# Deployment Guide

This guide covers deploying the Go API starter to production platforms.

## Overview

The starter includes a production-ready Dockerfile and can be deployed to:
- **Fly.io** - Global edge deployment with PostgreSQL and Redis
- **Railway** - Simple deployment with managed services
- **Any Docker-compatible platform** - AWS ECS, Google Cloud Run, DigitalOcean, etc.

## Prerequisites

- Docker installed locally (for building images)
- Account on your chosen platform
- PostgreSQL database (managed or self-hosted)
- Redis (optional, for queue system)

## Dockerfile

The included `Dockerfile` uses a multi-stage build:

1. **Build stage**: Compiles the Go application
2. **Final stage**: Minimal Alpine image with just the binary

##### Key features

- CGO disabled for static binary
- Minimal final image size (~20MB)
- Non-root user support (can be added)
- Health check ready

## Fly.io Deployment

Fly.io provides global edge deployment with built-in PostgreSQL and Redis.

### 1. Install Fly CLI

```bash
curl -L https://fly.io/install.sh | sh
```

### 2. Login

```bash
fly auth login
```

### 3. Initialize Fly App

```bash
fly launch
```

This will:
- Create a `fly.toml` configuration file
- Set up your app on Fly.io
- Optionally create a PostgreSQL database

### 4. Configure Environment Variables

```bash
# Set required variables
fly secrets set DATABASE_URL="postgresql://..." 
fly secrets set JWT_SECRET="your-secret-key"

# Optional: Set Redis for queue system
fly secrets set QUEUE_URL="redis://..."

# Optional: Rate limiting
fly secrets set RATE_LIMIT_ENABLED="true"
fly secrets set RATE_LIMIT_REQUESTS_PER_SEC="10.0"
fly secrets set RATE_LIMIT_BURST="20"
```

### 5. Add PostgreSQL (if not added during launch)

```bash
fly postgres create --name your-app-db
fly postgres attach --app your-app-name your-app-db
```

### 6. Add Redis (for queue system)

```bash
fly redis create --name your-app-redis
fly redis attach --app your-app-name your-app-redis
```

This automatically sets `REDIS_URL` which you can use as `QUEUE_URL`.

### 7. Run Migrations

```bash
# SSH into your app
fly ssh console

# Run migrations
export DATABASE_URL="your-database-url"
./migrate -path migrations -database "$DATABASE_URL" up
```

Or use a one-off command:

```bash
fly ssh console -C "./migrate -path migrations -database \"$DATABASE_URL\" up"
```

### 8. Deploy

```bash
fly deploy
```

### 9. Check Status

```bash
fly status
fly logs
```

### Example `fly.toml`

```toml
app = "your-app-name"
primary_region = "iad"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"
  ENVIRONMENT = "production"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]
    force_https = true

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [services.concurrency]
    type = "connections"
    hard_limit = 1000
    soft_limit = 500

  [[services.http_checks]]
    interval = "10s"
    timeout = "2s"
    grace_period = "5s"
    method = "GET"
    path = "/api/health"
```

## Railway Deployment

Railway provides simple deployment with automatic PostgreSQL and Redis.

### 1. Install Railway CLI

```bash
npm i -g @railway/cli
```

### 2. Login

```bash
railway login
```

### 3. Initialize Project

```bash
railway init
```

### 4. Add PostgreSQL

```bash
railway add postgresql
```

This automatically sets `DATABASE_URL`.

### 5. Add Redis (for queue system)

```bash
railway add redis
```

This automatically sets `REDIS_URL` (use as `QUEUE_URL`).

### 6. Set Environment Variables

In Railway dashboard or via CLI:

```bash
railway variables set JWT_SECRET="your-secret-key"
railway variables set QUEUE_URL="$REDIS_URL"
railway variables set RATE_LIMIT_ENABLED="true"
```

### 7. Run Migrations

Railway can run migrations automatically. Create a `railway.json`:

```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "DOCKERFILE",
    "dockerfilePath": "Dockerfile"
  },
  "deploy": {
    "startCommand": "./api",
    "healthcheckPath": "/api/health",
    "healthcheckTimeout": 100,
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 10
  }
}
```

Or run migrations manually:

```bash
railway run migrate -path migrations -database "$DATABASE_URL" up
```

### 8. Deploy

```bash
railway up
```

Railway automatically detects your Dockerfile and deploys.

## Environment Variables

### Required

- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for JWT tokens

### Optional

- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - `production` or `development`
- `LOG_LEVEL` - `debug`, `info`, `warn`, `error`
- `QUEUE_URL` - Redis URL for queue system (empty = in-memory)
- `RATE_LIMIT_ENABLED` - Enable rate limiting (default: true)
- `RATE_LIMIT_REQUESTS_PER_SEC` - Requests per second (default: 10.0)
- `RATE_LIMIT_BURST` - Burst size (default: 20)
- `API_ACCESS_TOKEN` - Service-to-service auth token

## Redis Configuration

**Important:** Redis is typically a **separate service**, not mounted in the container.

### Why Separate?

- **Scalability**: Redis can scale independently
- **Persistence**: Managed Redis services handle persistence
- **High Availability**: Managed services provide replication
- **Resource Management**: Separate resource limits

### Options:

1. **Managed Redis** (Recommended):
   - Fly.io: `fly redis create`
   - Railway: `railway add redis`
   - AWS ElastiCache
   - Google Cloud Memorystore
   - DigitalOcean Managed Redis

2. **Docker Compose** (Development):
   ```yaml
   redis:
     image: redis:7-alpine
     ports:
       - "6379:6379"
   ```

3. **In-Memory Queue** (Development):
   - Set `QUEUE_URL=""` to use in-memory queue
   - Jobs are lost on restart (fine for development)

## Health Checks

Both platforms support health checks:

- **Endpoint**: `/api/health`
- **Expected**: `200 OK` with JSON response
- **Timeout**: 2-5 seconds

## Monitoring

### Fly.io

```bash
# View logs
fly logs

# View metrics
fly metrics

# SSH into container
fly ssh console
```

### Railway

- View logs in Railway dashboard
- Metrics available in dashboard
- Can set up alerts

## Troubleshooting

### Database Connection Issues

- Verify `DATABASE_URL` is set correctly
- Check database is accessible from your app region
- Ensure SSL mode matches your database config

### Redis Connection Issues

- Verify `QUEUE_URL` is set (if using Redis)
- Check Redis is accessible from your app
- For development, use in-memory queue (`QUEUE_URL=""`)

### Migration Issues

- Run migrations before first deploy
- Check database user has CREATE/ALTER permissions
- Verify migration files are included in Docker image

### Build Issues

- Ensure `go.mod` and `go.sum` are committed
- Check Dockerfile paths are correct
- Verify Go version matches (1.25+)

## Production Checklist

- [ ] Set `ENVIRONMENT=production`
- [ ] Set strong `JWT_SECRET` (generate with `openssl rand -base64 32`)
- [ ] Enable rate limiting
- [ ] Set up managed PostgreSQL
- [ ] Set up managed Redis (if using queues)
- [ ] Run database migrations
- [ ] Configure health checks
- [ ] Set up monitoring/logging
- [ ] Configure custom domain (if needed)
- [ ] Set up SSL/TLS (usually automatic)
- [ ] Review security headers
- [ ] Test all endpoints
- [ ] Set up backup strategy for database

## Additional Resources

- [Fly.io Documentation](https://fly.io/docs/)
- [Railway Documentation](https://docs.railway.app/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)

