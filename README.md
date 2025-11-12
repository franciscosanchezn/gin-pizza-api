# gin-pizza-api

A REST API built with Go and Gin for pizza management with OAuth2 Client Credentials authentication - designed for Terraform provider integration.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
  - [For Users](#for-users)
  - [For Developers](#for-developers)
- [Authentication](#authentication)
- [API Endpoints](#api-endpoints)
- [Testing](#testing)
- [Documentation](#documentation)
- [Support](#support)
- [License](#license)

---

## Features

- **RESTful API** for pizza management (CRUD operations)
- **OAuth2 Client Credentials** authentication (machine-to-machine)
- **Role-based access control** (admin required for mutations)
- **Swagger/OpenAPI documentation** (interactive UI)
- **Dual-database support**: SQLite for local development, PostgreSQL for production (zero-code switching via environment variables)
- **JWT-based stateless tokens** (1 hour expiration)
- **Creator tracking** for resource ownership
- **Terraform provider ready** (idempotent operations, stable API contract)

---

## Quick Start

### For Users

If you just want to use the API:

**1. Start the server:**
```bash
go run cmd/main.go
```

Server starts on `http://localhost:8080`

**2. View available pizzas (no auth required):**
```bash
curl http://localhost:8080/api/v1/public/pizzas
```

**3. Create a dev OAuth client:**
```bash
go run scripts/create_dev_client.go
```

This creates:
- **Client ID:** `dev-client`
- **Client Secret:** `dev-secret-123`

**4. Get an access token:**
```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=dev-client" \
  -d "client_secret=dev-secret-123"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```

**5. Create a pizza:**
```bash
TOKEN="your_access_token"
curl -X POST http://localhost:8080/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Margherita",
    "price": 12.99,
    "ingredients": ["tomato sauce", "mozzarella", "basil"],
    "description": "Classic Italian pizza"
  }'
```

**6. Explore interactive API docs:**

Open http://localhost:8080/swagger/index.html

---

### Using PostgreSQL Locally

By default, the API uses SQLite for development. To use PostgreSQL instead:

**1. Start PostgreSQL with Docker:**
```bash
docker run -d --name pizza-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=pizza_api \
  -p 5432:5432 \
  postgres:16-alpine
```

**2. Set environment variables:**
```bash
export DB_DRIVER=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=pizza_api
export DB_SSLMODE=disable
```

**3. Start the server:**
```bash
go run cmd/main.go
```

The application automatically creates tables and seeds data on startup. Use the same API endpoints - database choice is transparent.

**4. Stop PostgreSQL when done:**
```bash
docker stop pizza-postgres && docker rm pizza-postgres
```

> **Note:** See [DATABASE_ARCHITECTURE.md](docs/internal/DATABASE_ARCHITECTURE.md) for detailed documentation on dual-database support, connection pooling, Kubernetes deployment, and migration strategies.

---

### For Developers

If you're setting up the project for development:

**1. Clone and setup:**
```bash
git clone https://github.com/franciscosanchezn/gin-pizza-api.git
cd gin-pizza-api
go mod download
```

**2. Create environment file:**
```bash
cp .env.example .env
# Edit .env with your settings (JWT_SECRET is required)
```

**Generate a secure JWT secret:**
```bash
openssl rand -base64 32
```

**3. Install development tools:**
```bash
# Swagger CLI for documentation generation
go install github.com/swaggo/swag/cmd/swag@latest

# Optional: Air for hot reload
go install github.com/air-verse/air@latest
```

**4. Run the application:**
```bash
# Standard development mode
go run cmd/main.go

# With hot reload
air

# Build and run binary
go build -o bin/pizza-api cmd/main.go
./bin/pizza-api
```

**5. Run tests:**
```bash
# Unit tests
go test ./...

# Integration tests
./scripts/test-api.sh
```

**6. Create OAuth client for testing:**
```bash
go run scripts/create_dev_client.go
```

**7. Regenerate Swagger docs (after API changes):**
```bash
swag init -g cmd/main.go
```

---

## Authentication

This API uses **OAuth2 Client Credentials** flow for machine-to-machine authentication.

### Quick Overview

1. Create OAuth client credentials (ID + secret)
2. Exchange credentials for JWT access token: `POST /oauth/token`
3. Include token in requests: `Authorization: Bearer <token>`
4. Token lifetime: 3600 seconds (1 hour)

### Token Acquisition

**Endpoint:** `POST /api/v1/oauth/token`

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET"
```

**Response:**
```json
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```

### Using the Token

Include in `Authorization` header:
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/protected/admin/pizzas
```

### JWT Token Details

Tokens contain these claims:
- **`uid`**: User ID (for creator attribution)
- **`role`**: User role (`admin` or `user`)
- **`aud`**: OAuth client ID
- **`scope`**: Granted scopes (`read write`)
- **`exp`**: Expiration timestamp
- **`iat`**: Issued at timestamp

**For detailed authentication architecture, see:**
- [JWT Internals Documentation](docs/internal/JWT_INTERNALS.md) - Deep dive into token structure, service account model, and security considerations

---

## API Endpoints

### Public Endpoints (No Authentication)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/public/pizzas` | List all pizzas |
| `GET` | `/api/v1/public/pizzas/:id` | Get specific pizza |

### Protected Endpoints (Requires Admin Role)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/oauth/token` | Get OAuth access token |
| `POST` | `/api/v1/protected/admin/pizzas` | Create pizza |
| `PUT` | `/api/v1/protected/admin/pizzas/:id` | Update pizza |
| `DELETE` | `/api/v1/protected/admin/pizzas/:id` | Delete pizza |

### Query Parameters (Future Phase 3)

```bash
# Filter by name
GET /api/v1/public/pizzas?name=Margherita

# Filter by creator
GET /api/v1/public/pizzas?created_by=<user_id>
```

### Interactive API Documentation

**Swagger UI:** http://localhost:8080/swagger/index.html

**Using Swagger with authentication:**
1. Get OAuth token (see [Authentication](#authentication))
2. Click "Authorize" button in Swagger UI
3. Enter: `Bearer YOUR_ACCESS_TOKEN`
4. Test endpoints directly from browser

---

## Testing

### Unit Tests

Run all Go unit tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -v -cover ./...
```

Run with race detector:
```bash
go test -v -race ./...
```

### Integration Tests

The project includes a comprehensive integration test suite that validates the complete CRUD lifecycle with OAuth2 authentication:

```bash
./scripts/test-api.sh
```

**What it tests:**
- OAuth2 token acquisition
- Public endpoint access (no authentication)
- Protected endpoint authorization
- Full CRUD operations (Create, Read, Update, Delete)
- Resource ownership validation

**Requirements:**
- Port 8080 must be available
- Dev OAuth client must exist (auto-created by script)

**CI Integration:**
This script runs automatically on all pull requests via GitHub Actions.

**For detailed testing information, see:**
- [Development Guide](docs/internal/DEVELOPMENT.md) - Test coverage details and development workflow

---

## Documentation

### User-Facing Documentation

- **[API Contract](docs/API_CONTRACT.md)** - Formal API specifications, versioning, error codes, idempotency guarantees
- **[Terraform Provider Guide](docs/TERRAFORM_PROVIDER.md)** - Complete guide for building a Terraform provider against this API
- **[Kubernetes Deployment](docs/KUBERNETES.md)** - Deploy to microk8s with HTTPS and ingress configuration

### Internal/Contributor Documentation

- **[Development Guide](docs/internal/DEVELOPMENT.md)** - Project structure, adding endpoints, coding standards, environment variables
- **[Operations Guide](docs/internal/OPERATIONS.md)** - Deployment strategies, monitoring, troubleshooting, production considerations
- **[Contributing Guide](docs/internal/CONTRIBUTING.md)** - How to contribute, development process, commit guidelines, code review
- **[JWT Internals](docs/internal/JWT_INTERNALS.md)** - Deep dive into token structure, claims, OAuth service account model

### API Documentation

**Swagger/OpenAPI documentation is auto-generated:**

**View:** http://localhost:8080/swagger/index.html

**Regenerate after API changes:**
```bash
swag init -g cmd/main.go
```

---

## Configuration

The API uses environment variables for configuration. Create a `.env` file:

```bash
cp .env.example .env
```

**Key environment variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | `8080` | Server port |
| `JWT_SECRET` | *(required)* | JWT signing secret (minimum 32 chars) |
| `DATABASE_URL` | `sqlite://test.sqlite` | Database connection string |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `GIN_MODE` | `debug` | Gin mode (`debug` or `release`) |

**Generate secure JWT secret:**
```bash
openssl rand -base64 32
```

**For complete configuration details, see:**
- [Development Guide](docs/internal/DEVELOPMENT.md#environment-variables) - All environment variables and configuration loading

---

## Deployment

### Docker

```bash
# Build image
docker build -t pizza-api:latest .

# Run container
docker run -d \
  --name pizza-api \
  -p 8080:8080 \
  -e JWT_SECRET="your-production-secret" \
  -e APP_ENV="production" \
  pizza-api:latest
```

### Kubernetes

See [Kubernetes Deployment Guide](docs/KUBERNETES.md) for complete setup including:
- Deployment manifests
- Service configuration
- ConfigMaps and Secrets
- Ingress with TLS

### Production Considerations

**For production deployment best practices, see:**
- [Operations Guide](docs/internal/OPERATIONS.md) - Security, database, monitoring, disaster recovery

---

## Support

### Getting Help

- **Issues:** https://github.com/franciscosanchezn/gin-pizza-api/issues
- **Discussions:** https://github.com/franciscosanchezn/gin-pizza-api/discussions
- **Documentation:** All guides available in [`docs/`](docs/) directory

### Contributing

We welcome contributions! Please read:
- [Contributing Guide](docs/internal/CONTRIBUTING.md) - Complete contribution workflow and standards

### Troubleshooting

Common issues and solutions:
- [Operations Guide - Troubleshooting](docs/internal/OPERATIONS.md#troubleshooting) - Port conflicts, database issues, OAuth errors

---

## License

[MIT License](LICENSE)

---

## Acknowledgments

- Built with [Gin Web Framework](https://github.com/gin-gonic/gin)
- Authentication via [go-oauth2/oauth2](https://github.com/go-oauth2/oauth2)
- Documentation with [Swag](https://github.com/swaggo/swag)
- Database with [GORM](https://gorm.io/)

---

**Project Status:** Active development | **Version:** 1.0.0 | **Last Updated:** November 11, 2025
