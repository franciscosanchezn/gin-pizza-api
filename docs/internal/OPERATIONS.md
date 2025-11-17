# Operations Guide

This guide covers deployment strategies, production considerations, monitoring, and troubleshooting for the Pizza API.

---

## Table of Contents

- [Docker Deployment](#docker-deployment)
- [Production Considerations](#production-considerations)
- [Environment-Specific Configuration](#environment-specific-configuration)
- [Troubleshooting](#troubleshooting)
- [Monitoring and Observability](#monitoring-and-observability)

---

## Docker Deployment

### Building and Running

**1. Build Docker image:**
```bash
docker build -t pizza-api:latest .
```

**2. Run container:**
```bash
docker run -d \
  --name pizza-api \
  -p 8080:8080 \
  -e JWT_SECRET="your-production-secret" \
  -e APP_ENV="production" \
  -e GIN_MODE="release" \
  -v $(pwd)/data:/app/data \
  pizza-api:latest
```

**3. View logs:**
```bash
docker logs -f pizza-api
```

**4. Stop container:**
```bash
docker stop pizza-api
docker rm pizza-api
```

### Docker Compose

For local development with dependencies:

```yaml
version: '3.8'
services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - JWT_SECRET=dev-secret-123
      - APP_ENV=development
      - DATABASE_URL=sqlite://test.sqlite
    volumes:
      - ./data:/app/data
```

Run with:
```bash
docker-compose up -d
```

---

## Production Considerations

### Security

**Required Security Measures:**

- ✅ **Use strong JWT secrets** (minimum 32 characters, cryptographically random)
  ```bash
  openssl rand -base64 32
  ```

- ✅ **Enable HTTPS** (TLS/SSL certificates)
  - Terminate TLS at load balancer or ingress controller
  - Never expose HTTP endpoints in production

- ✅ **Set `GIN_MODE=release`**
  - Disables debug logging
  - Improves performance

- ✅ **Use environment variables, not `.env` files**
  - Kubernetes Secrets for sensitive values
  - ConfigMaps for non-sensitive configuration

- ✅ **Implement rate limiting**
  - Protect OAuth token endpoint from brute force
  - Limit API request rates per client

- ✅ **Enable CORS properly**
  - Restrict allowed origins in production
  - Don't use wildcard (`*`) for credentials

- ✅ **Keep dependencies updated**
  ```bash
  go get -u ./...
  go mod tidy
  ```

### Database

**SQLite Limitations:**
- Not suitable for high concurrency
- No horizontal scaling
- Single file can be a bottleneck

**Production Database Recommendations:**

**PostgreSQL** (Recommended):
```env
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=require
```

**MySQL/MariaDB**:
```env
DATABASE_URL=mysql://user:password@host:3306/dbname?parseTime=true
```

**Connection Pooling:**
```go
// In config/database.go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

**Backup Strategy:**
- Automated daily backups
- Point-in-time recovery capability
- Test restoration procedures regularly

### Performance

**Optimization Checklist:**

- [ ] Enable database query caching
- [ ] Add indexes to frequently queried columns
- [ ] Implement connection pooling
- [ ] Use CDN for static assets (if applicable)
- [ ] Enable GZIP compression
- [ ] Set appropriate cache headers
- [ ] Profile critical endpoints

**Horizontal Scaling:**
- Deploy multiple API instances behind a load balancer
- Use stateless authentication (JWT) - no session storage required
- Ensure database can handle concurrent connections

### Kubernetes Deployment

**Recommended for Production.**

See `docs/KUBERNETES.md` for complete deployment guide including:
- Deployment manifests
- Service configuration
- ConfigMaps and Secrets
- Ingress with TLS
- HorizontalPodAutoscaler
- PersistentVolumeClaims

**Quick Reference:**
```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -l app=pizza-api

# View logs
kubectl logs -f deployment/pizza-api

# Scale deployment
kubectl scale deployment pizza-api --replicas=3
```

---

## Environment-Specific Configuration

### Development

```env
APP_ENV=development
LOG_LEVEL=debug
GIN_MODE=debug
APP_HOST=localhost
DATABASE_URL=sqlite://test.sqlite
```

**Characteristics:**
- Verbose logging
- Hot reload enabled (with Air)
- SQLite for simplicity
- Debug endpoints enabled

### Staging

```env
APP_ENV=staging
LOG_LEVEL=info
GIN_MODE=release
APP_HOST=0.0.0.0
DATABASE_URL=postgres://user:pass@staging-db:5432/pizza_api
```

**Characteristics:**
- Production-like environment
- PostgreSQL database
- HTTPS enforced
- Similar resource limits to production

### Production

```env
APP_ENV=production
LOG_LEVEL=warn
GIN_MODE=release
APP_HOST=0.0.0.0
DATABASE_URL=postgres://user:pass@prod-db:5432/pizza_api
```

**Characteristics:**
- Minimal logging (warn/error only)
- High availability (multiple replicas)
- Database with backups
- Monitoring and alerting enabled

---

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Symptom:**
```
Error: listen tcp :8080: bind: address already in use
```

**Solution:**
```bash
# Find process using port 8080
lsof -ti:8080

# Kill the process
lsof -ti:8080 | xargs kill -9

# Or use a different port
APP_PORT=8081 go run cmd/main.go
```

---

#### 2. Database Locked

**Symptom:**
```
Error: database is locked
```

**Cause:** SQLite doesn't handle concurrent writes well.

**Solution (Development):**
```bash
# Reset database
rm test.sqlite
go run cmd/main.go
```

**Solution (Production):**
Migrate to PostgreSQL or MySQL.

---

#### 3. OAuth Token Invalid

**Symptom:**
```json
{
  "error": "invalid_token",
  "error_description": "Token signature is invalid"
}
```

**Possible Causes:**
- Token has expired (default: 3600 seconds / 1 hour)
- `JWT_SECRET` mismatch between token creation and validation
- Malformed `Authorization` header (missing `Bearer` prefix)

**Solutions:**
```bash
# Check token expiration
echo "YOUR_TOKEN" | cut -d'.' -f2 | base64 -d | jq '.exp'

# Verify JWT_SECRET matches
echo $JWT_SECRET

# Correct Authorization header format
Authorization: Bearer <token>
```

---

#### 4. Token Generation Failed

**Symptom:**
```json
{
  "error": "server_error",
  "error_description": "Token generation failed"
}
```

**Possible Causes:**
- OAuth client missing required fields (`user_id`, `domain`, `grant_types`)
- Database constraint violation
- `JWT_SECRET` not set

**Solutions:**
```bash
# Create a properly configured dev client
go run scripts/create_dev_client.go

# Check server logs for detailed error
LOG_LEVEL=debug go run cmd/main.go
```

**Verify OAuth client fields:**
- `user_id`: Must reference a valid User record
- `domain`: Must be set (e.g., "http://localhost")
- `grant_types`: Must include "client_credentials"

---

#### 5. Permission Denied (403)

**Symptom:**
```json
{
  "error": "insufficient_permissions",
  "error_description": "Admin role required"
}
```

**Cause:** OAuth client's associated User doesn't have admin role.

**Solution:**
```bash
# Update user role in database
sqlite3 test.sqlite
sqlite> UPDATE users SET role = 'admin' WHERE id = 1;

# Or create new admin client
go run scripts/create_dev_client.go
```

---

#### 6. Swagger Docs Out of Date

**Symptom:** Swagger UI doesn't reflect recent endpoint changes.

**Solution:**
```bash
# Regenerate Swagger documentation
swag init -g cmd/main.go

# Restart server
go run cmd/main.go
```

---

### Debug Mode

**Enable verbose logging:**
```bash
LOG_LEVEL=debug go run cmd/main.go
```

**Inspect JWT token contents:**
```bash
# Decode JWT payload (base64)
echo "YOUR_TOKEN" | cut -d'.' -f2 | base64 -d | jq
```

**Example decoded token:**
```json
{
  "uid": "1",
  "role": "admin",
  "aud": "dev-client",
  "scope": "read write",
  "exp": 1699632000,
  "iat": 1699628400
}
```

---

## Monitoring and Observability

### Health Checks

**Kubernetes liveness probe:**
```yaml
livenessProbe:
  httpGet:
    path: /api/v1/public/pizzas
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
```

**Readiness probe:**
```yaml
readinessProbe:
  httpGet:
    path: /api/v1/public/pizzas
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Logging

**Structured logging** (current implementation uses Gin's default logger).

**Recommended improvements:**
- Use `logrus` or `zap` for structured JSON logs
- Include correlation IDs for request tracing
- Log severity levels: DEBUG, INFO, WARN, ERROR

**Example log entry:**
```json
{
  "timestamp": "2025-11-11T10:30:00Z",
  "level": "info",
  "method": "POST",
  "path": "/api/v1/pizzas",
  "status": 201,
  "latency_ms": 15,
  "client_id": "terraform-client"
}
```

### Metrics

**Key metrics to track:**

**Application Metrics:**
- Request rate (requests/second)
- Response times (p50, p95, p99)
- Error rate (4xx, 5xx responses)
- Active OAuth tokens

**Infrastructure Metrics:**
- CPU usage
- Memory usage
- Database connections (active/idle)
- Disk I/O (for SQLite)

**Business Metrics:**
- Pizzas created/updated/deleted per day
- Unique OAuth clients
- API endpoint usage distribution

**Tools:**
- Prometheus + Grafana
- DataDog
- New Relic
- AWS CloudWatch (if on AWS)

### Error Tracking

**Recommended tools:**
- Sentry (recommended for Go)
- Rollbar
- Bugsnag

**Integration example (Sentry):**
```go
import "github.com/getsentry/sentry-go"

func init() {
    sentry.Init(sentry.ClientOptions{
        Dsn: os.Getenv("SENTRY_DSN"),
        Environment: os.Getenv("APP_ENV"),
    })
}
```

---

## Disaster Recovery

### Backup Procedures

**Database backups:**
```bash
# PostgreSQL
pg_dump -h localhost -U user -d pizza_api > backup_$(date +%Y%m%d).sql

# Restore
psql -h localhost -U user -d pizza_api < backup_20251111.sql
```

**Kubernetes persistent volumes:**
```bash
# Create snapshot
kubectl create snapshot pvc-snapshot --pvc=data-pvc

# Restore from snapshot
kubectl create pvc restored-data --snapshot=pvc-snapshot
```

### Rollback Strategy

**Kubernetes rollback:**
```bash
# View deployment history
kubectl rollout history deployment/pizza-api

# Rollback to previous version
kubectl rollout undo deployment/pizza-api

# Rollback to specific revision
kubectl rollout undo deployment/pizza-api --to-revision=2
```

---

## Additional Resources

- [Development Guide](DEVELOPMENT.md) - Project structure, coding standards
- [Kubernetes Deployment Guide](../KUBERNETES.md) - Complete K8s setup
- [Contributing Guide](CONTRIBUTING.md) - Contribution process
- [JWT Internals](JWT_INTERNALS.md) - Authentication architecture
