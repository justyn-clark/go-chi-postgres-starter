# Prometheus Metrics - Complete Guide

## What You're Seeing

The `/metrics` endpoint exposes two types of metrics:

1. **Go Runtime Metrics** (always present):
   - `go_goroutines` - Number of goroutines
   - `go_memstats_*` - Memory usage
   - `go_gc_*` - Garbage collection stats
   - `process_*` - Process-level stats (CPU, memory, file descriptors)

2. **Custom HTTP Metrics** (appear after requests):
   - `http_requests_total` - Total requests per endpoint
   - `http_request_duration_seconds` - Request latency
   - `http_request_size_bytes` - Request payload size
   - `http_response_size_bytes` - Response payload size

## What to Do With Metrics

### Option 1: Manual Monitoring (Quick Check)

Just curl the endpoint to see current state:

```bash
# View all metrics
curl http://localhost:8080/metrics

# Filter for specific metrics
curl http://localhost:8080/metrics | grep http_requests_total
curl http://localhost:8080/metrics | grep http_request_duration_seconds
```

**Use case:** Quick health checks, debugging, manual monitoring

### Option 2: Prometheus Server (Production Monitoring)

Set up Prometheus to scrape and store metrics:

#### 1. Install Prometheus

```bash
# macOS
brew install prometheus

# Or download from https://prometheus.io/download/
```

#### 2. Create `prometheus.yml`

```yaml
global:
  scrape_interval: 15s  # How often to scrape
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'go-api'
    static_configs:
      - targets: ['localhost:8080']  # Your API endpoint
```

#### 3. Start Prometheus

```bash
prometheus --config.file=prometheus.yml
```

Prometheus will:

- Scrape `/metrics` every 15 seconds
- Store historical data
- Provide a query interface at `http://localhost:9090`

#### 4. Query Metrics in Prometheus UI

Open `http://localhost:9090` and try these queries:

##### Total requests

```text
http_requests_total
```

##### Requests per second

```text
rate(http_requests_total[5m])
```

##### Average response time

```text
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

##### Error rate (5xx)

```text
rate(http_requests_total{status=~"5.."}[5m])
```

##### Requests by endpoint

```text
sum by (path) (rate(http_requests_total[5m]))
```

##### 95th percentile latency

```text
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Option 3: Grafana Dashboards (Visualization)

Grafana provides beautiful dashboards for metrics:

#### 1. Install Grafana

```bash
# macOS
brew install grafana

# Or Docker
docker run -d -p 3000:3000 grafana/grafana
```

#### 2. Add Prometheus as Data Source

1. Open `http://localhost:3000`
2. Login (admin/admin)
3. Add data source → Prometheus
4. URL: `http://localhost:9090`

#### 3. Create Dashboard

##### Panel 1: Request Rate

- Query: `rate(http_requests_total[5m])`
- Visualization: Graph
- Title: "Requests per Second"

##### Panel 2: Response Time

- Query: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
- Visualization: Graph
- Title: "95th Percentile Latency"

##### Panel 3: Error Rate

- Query: `rate(http_requests_total{status=~"5.."}[5m])`
- Visualization: Graph
- Title: "Error Rate"

##### Panel 4: Top Endpoints

- Query: `topk(10, sum by (path) (rate(http_requests_total[5m])))`
- Visualization: Table
- Title: "Top 10 Endpoints"

### Option 4: Alerting (Production)

Set up alerts based on metrics:

```yaml
# prometheus.yml
rule_files:
  - alerts.yml
```

```yaml
# alerts.yml
groups:
  - name: api_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High error rate detected"
      
      - alert: SlowResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        annotations:
          summary: "95th percentile latency > 1s"
```

## Common Use Cases

### 1. Performance Monitoring

- Track slow endpoints
- Identify bottlenecks
- Monitor response times

### 2. Capacity Planning

- Understand traffic patterns
- Plan for scaling
- Identify peak usage

### 3. Error Tracking

- Monitor error rates
- Get alerts on failures
- Track error trends

### 4. Debugging

- See which endpoints are called
- Check request/response sizes
- Monitor goroutine count

## Quick Start (Docker Compose)

Add to `docker-compose.yml`:

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

Then:

```bash
docker-compose up -d prometheus grafana
```

## Example Queries

### Total requests in last hour

```text
sum(increase(http_requests_total[1h]))
```

### Average request size

```text
rate(http_request_size_bytes_sum[5m]) / rate(http_request_size_bytes_count[5m])
```

### Success rate

```text
sum(rate(http_requests_total{status=~"2.."}[5m])) / sum(rate(http_requests_total[5m]))
```

### Requests by method

```text
sum by (method) (rate(http_requests_total[5m]))
```

## Next Steps

1. **Development**: Just curl `/metrics` for quick checks
2. **Staging**: Set up Prometheus to scrape metrics
3. **Production**: Add Grafana dashboards and alerts

## Resources

- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
