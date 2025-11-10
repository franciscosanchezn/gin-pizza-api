# gin-pizza-api

A REST API built with Go and Gin for pizza management with OAuth2 Client Credentials authentication - designed for Terraform provider integration.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
  - [For Users](#for-users)
  - [For Developers](#for-developers)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [API Documentation](#api-documentation)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)

## Features

- RESTful API for pizza management (CRUD operations)
- **OAuth2 Client Credentials** authentication (machine-to-machine)
- Role-based access control (Admin only for mutations)
- Swagger/OpenAPI documentation
- SQLite database with GORM
- Environment-based configuration
- Structured logging with logrus
- JWT-based stateless tokens
- Creator tracking for pizzas

## Quick Start

### For Users

If you just want to use the API:

**1. Start the server:**
```bash
go run cmd/main.go
```

**2. View available pizzas:**
```bash
curl http://localhost:8080/api/v1/public/pizzas
```

**3. Get an admin OAuth client:**

Since client management is admin-only, you'll need to create an OAuth client directly in the database or ask an administrator. For development, you can use the seeded admin user.

**4. Get an access token:**
```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET"
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

**6. Explore API docs:**
Open http://localhost:8080/swagger/index.html

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
# Edit .env with your settings
```

**3. Install development tools:**
```bash
# Swagger CLI for documentation
go install github.com/swaggo/swag/cmd/swag@latest

# Optional: Air for hot reload
go install github.com/air-verse/air@latest
```

**4. Run the application:**
```bash
# Development mode
go run cmd/main.go

# Or with hot reload
air

# Or build and run
go build -o bin/pizza-api cmd/main.go
./bin/pizza-api
```

**5. Run tests:**
```bash
go test ./...
```

**6. Regenerate Swagger docs (after API changes):**
```bash
swag init -g cmd/main.go
```

**7. Create an OAuth client for testing:**

The database is automatically seeded with:
- System user (ID: 1, email: system@pizza.com, role: admin)
- Sample pizzas (Margherita, Pepperoni, Vegetarian)

**Use the provided script (Recommended):**
```bash
go run scripts/create_dev_client.go
```

This creates a development OAuth client with:
- Client ID: `dev-client`
- Client Secret: `dev-secret-123`
- Domain: `http://localhost`
- Grant Types: `client_credentials`
- Scopes: `read write`

**Or create a custom client programmatically:**

```go
// Create a client with all required fields
client := models.OAuthClient{
    ID:          "test-client-id",
    Secret:      "$2a$10$...", // bcrypt hash of your secret
    Name:        "Test Client",
    Domain:      "http://localhost",
    UserID:      1, // Required for token generation
    Scopes:      "read write",
    GrantTypes:  "client_credentials",
    RedirectURI: "",
}
```

**Or use SQL:**
```sql
INSERT INTO oauth_clients (id, secret, name, domain, user_id, scopes, grant_types, created_at, updated_at) 
VALUES (
  'test-client', 
  '$2a$10$encrypted_secret_here',
  'Test Client',
  'http://localhost',
  1,  -- user_id is required for OAuth2 token generation
  'read write',
  'client_credentials',
  datetime('now'),
  datetime('now')
);
```

**Generate bcrypt hash in Go:**
```bash
go run -e 'package main; import ("fmt"; "golang.org/x/crypto/bcrypt"); func main() { hash, _ := bcrypt.GenerateFromPassword([]byte("your-secret"), bcrypt.DefaultCost); fmt.Println(string(hash)) }'
```

---

## Prerequisites

- Go 1.21 or higher
- Git

## Installation

1. **Clone the repository:**

```bash
git clone https://github.com/franciscosanchezn/gin-pizza-api.git
cd gin-pizza-api
```

2. **Install dependencies:**

```bash
go mod download
```

3. **Install Swagger CLI (for documentation generation):**

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

## Configuration

The API uses environment variables for configuration. Create a `.env` file in the project root:

**1. Create environment file:**

```bash
cp .env.example .env
```

**2. Configuration options:**

```env
# Application settings
APP_ENV=development          # Environment: development, staging, production
APP_PORT=8080               # Server port
APP_HOST=localhost          # Server host (use 0.0.0.0 for Docker)

# Database settings
DATABASE_URL=sqlite://test.sqlite  # Database connection string
DB_NAME=test.sqlite               # Database name
DB_USER=admin                     # Database user (for PostgreSQL/MySQL)
DB_PASSWORD=secret                # Database password

# Security
JWT_SECRET=your-super-secret-256-bit-key-here-make-it-long-and-random

# Logging
LOG_LEVEL=info              # Log level: debug, info, warn, error
```

**3. Generate a secure JWT secret:**

```bash
openssl rand -base64 32
```

**Important Notes:**
- Never commit `.env` files to version control
- Use strong, random JWT secrets (minimum 32 characters)
- In production, use environment variables instead of `.env` file
- For Kubernetes deployments, use ConfigMaps and Secrets

---

## Authentication

The API uses **OAuth2 Client Credentials** flow for authentication. This is designed for machine-to-machine communication (e.g., Terraform providers, CI/CD systems).

### OAuth2 Flow

1. **Register an OAuth client** (admin only)
2. **Request an access token** using client credentials
3. **Use the token** to access protected endpoints

### Getting an Access Token

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
  "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```

### Using the Access Token

Include the token in the `Authorization` header:

```bash
curl -X GET http://localhost:8080/api/v1/protected/admin/clients \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Token Characteristics

- **Type:** JWT (JSON Web Token)
- **Signing Algorithm:** HS512
- **Stateless:** No server-side storage
- **Expiration:** Configurable (default: 1 hour)
- **Contains:** Client ID, scopes, expiration time

---

## API Documentation

### Interactive Documentation

The API uses Swagger/OpenAPI for interactive documentation:

**1. Start the server:**
```bash
go run cmd/main.go
```

**2. Open Swagger UI:**
```
http://localhost:8080/swagger/index.html
```

**3. Authenticate in Swagger:**
- Click the "Authorize" button (🔒)
- Enter: `Bearer YOUR_ACCESS_TOKEN`
- Click "Authorize"
- Test endpoints directly from the UI

### Regenerating Swagger Documentation

After adding or modifying endpoints:

```bash
swag init -g cmd/main.go
```

### API Endpoints Reference

#### Public Endpoints (No Authentication Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check endpoint |
| `GET` | `/api/v1/public/pizzas` | List all pizzas |
| `GET` | `/api/v1/public/pizzas/:id` | Get pizza by ID |
| `GET` | `/swagger/*any` | Swagger documentation |

#### OAuth2 Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/oauth/token` | Get access token (client credentials) |

**Token Request Example:**
```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET"
```

#### Protected Endpoints (Requires OAuth2 Token + Admin Role)

**Pizza Management:**

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| `POST` | `/api/v1/protected/admin/pizzas` | Create new pizza | Admin |
| `PUT` | `/api/v1/protected/admin/pizzas/:id` | Update pizza | Admin |
| `DELETE` | `/api/v1/protected/admin/pizzas/:id` | Delete pizza | Admin |

**OAuth Client Management:**

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| `POST` | `/api/v1/protected/admin/clients` | Create OAuth client | Admin |
| `GET` | `/api/v1/protected/admin/clients` | List OAuth clients | Admin |
| `DELETE` | `/api/v1/protected/admin/clients/:id` | Delete OAuth client | Admin |

### Example Requests

#### 1. List All Pizzas (Public)
```bash
curl http://localhost:8080/api/v1/public/pizzas
```

**Response:**
```json
[
  {
    "id": 1,
    "name": "Margherita",
    "description": "Classic Italian pizza",
    "ingredients": ["Tomato Sauce", "Mozzarella", "Basil"],
    "price": 10.99,
    "created_by": 1,
    "created_at": "2025-11-10T12:00:00Z",
    "updated_at": "2025-11-10T12:00:00Z"
  }
]
```

#### 2. Get Pizza by ID (Public)
```bash
curl http://localhost:8080/api/v1/public/pizzas/1
```

#### 3. Create Pizza (Protected - Admin Only)
```bash
TOKEN="your_access_token"

curl -X POST http://localhost:8080/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Hawaiian",
    "description": "Tropical delight",
    "ingredients": ["Ham", "Pineapple", "Mozzarella"],
    "price": 13.99
  }'
```

**Response:**
```json
{
  "id": 4,
  "name": "Hawaiian",
  "description": "Tropical delight",
  "ingredients": ["Ham", "Pineapple", "Mozzarella"],
  "price": 13.99,
  "created_by": 1,
  "created_at": "2025-11-10T12:30:00Z",
  "updated_at": "2025-11-10T12:30:00Z"
}
```

#### 4. Update Pizza (Protected - Admin Only)
```bash
curl -X PUT http://localhost:8080/api/v1/protected/admin/pizzas/4 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Hawaiian Deluxe",
    "price": 14.99
  }'
```

#### 5. Delete Pizza (Protected - Admin Only)
```bash
curl -X DELETE http://localhost:8080/api/v1/protected/admin/pizzas/4 \
  -H "Authorization: Bearer $TOKEN"
```

#### 6. Create OAuth Client (Protected - Admin Only)
```bash
curl -X POST http://localhost:8080/api/v1/protected/admin/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Terraform Provider",
    "scopes": "read write"
  }'
```

**Response:**
```json
{
  "client_id": "generated-client-id",
  "client_secret": "generated-secret-shown-only-once",
  "name": "Terraform Provider",
  "scopes": "read write",
  "created_at": "2025-11-10T12:45:00Z"
}
```

**⚠️ Important:** The client secret is only shown once during creation. Store it securely!

#### 7. List OAuth Clients (Protected - Admin Only)
```bash
curl -X GET http://localhost:8080/api/v1/protected/admin/clients \
  -H "Authorization: Bearer $TOKEN"
```

### Error Responses

**401 Unauthorized:**
```json
{
  "error": "invalid_client",
  "error_description": "Client authentication failed"
}
```

**403 Forbidden:**
```json
{
  "error": "insufficient_permissions",
  "message": "Admin role required"
}
```

**404 Not Found:**
```json
{
  "error": "Pizza not found"
}
```

---

## Development

### Project Structure

```
gin-pizza-api/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── auth/                # OAuth2 authentication
│   │   ├── client_credentials.go
│   │   ├── oauth_server.go
│   │   └── gorm_store.go
│   ├── config/              # Configuration management
│   │   └── config.go
│   ├── controllers/         # HTTP handlers
│   │   ├── pizza-controller.go
│   │   └── client_controller.go
│   ├── middleware/          # Middleware (auth, RBAC)
│   │   ├── middleware.go
│   │   └── role.go
│   ├── models/              # Data models
│   │   ├── pizza.go
│   │   ├── user.go
│   │   └── oauth_client.go
│   ├── services/            # Business logic
│   │   ├── pizza_service.go
│   │   └── client_service.go
│   └── logging/             # Logging setup
│       └── logging.go
├── docs/                    # Generated Swagger documentation
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── .tasks/                  # Project documentation
│   ├── architect-analysis-20251110.md
│   └── cleanup-summary-20251110.md
├── .env                     # Environment variables (git ignored)
├── .env.example             # Environment template
├── go.mod                   # Go module file
├── go.sum                   # Go module checksums
├── Dockerfile               # Docker configuration
└── README.md                # This file
```

### Running the Application

**Development mode (with hot reload):**
```bash
# Install Air
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

**Standard development mode:**
```bash
go run cmd/main.go
```

**Build and run:**
```bash
# Build binary
go build -o bin/pizza-api cmd/main.go

# Run binary
./bin/pizza-api
```

**Docker:**
```bash
# Build image
docker build -t pizza-api .

# Run container
docker run -p 8080:8080 \
  -e JWT_SECRET=your-secret \
  pizza-api
```

### Database Management

The application uses **SQLite** by default for simplicity. On startup:

1. **Auto-migration** runs for all models
2. **Seeding** occurs if database is empty:
   - System admin user (email: system@pizza.com)
   - Sample pizzas (Margherita, Pepperoni, Vegetarian)

**Database location:** `test.sqlite` (in project root)

**Reset database:**
```bash
rm test.sqlite
go run cmd/main.go  # Will recreate and seed
```

### Adding New Endpoints

**1. Define the model** (`internal/models/`):
```go
type Pizza struct {
    ID          int            `json:"id" gorm:"primaryKey"`
    Name        string         `json:"name" gorm:"not null"`
    Price       float64        `json:"price"`
    CreatedBy   uint           `json:"created_by"`
    CreatedAt   time.Time      `json:"created_at"`
}
```

**2. Create the service** (`internal/services/`):
```go
type PizzaService interface {
    GetAllPizzas() ([]models.Pizza, error)
    // ... other methods
}
```

**3. Implement the controller** (`internal/controllers/`):
```go
// GetAllPizzas godoc
// @Summary List all pizzas
// @Description Get all pizzas from database
// @Tags pizzas
// @Produce json
// @Success 200 {array} models.Pizza
// @Router /api/v1/public/pizzas [get]
func (c *controller) GetAllPizzas(ctx *gin.Context) {
    pizzas, err := c.service.GetAllPizzas()
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, pizzas)
}
```

**4. Register the route** (`cmd/main.go`):
```go
publicApi.GET("/pizzas", pizzaController.GetAllPizzas)
```

**5. Regenerate Swagger docs:**
```bash
swag init -g cmd/main.go
```

### Swagger Annotations Reference

**Common annotations:**
```go
// @Summary      Short description
// @Description  Long description
// @Tags         category-name
// @Accept       json
// @Produce      json
// @Param        name path string true "Description"
// @Param        body body models.Pizza true "Pizza object"
// @Success      200 {object} models.Pizza
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/path [method]
```

### Code Style and Formatting

**Format code:**
```bash
gofmt -w .
```

**Lint code:**
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Environment Variables

During development, the app loads from `.env` file. Available variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment name |
| `APP_PORT` | `8080` | Server port |
| `APP_HOST` | `localhost` | Server host |
| `DATABASE_URL` | `sqlite://test.sqlite` | Database connection |
| `DB_NAME` | `test.sqlite` | Database name |
| `JWT_SECRET` | (required) | JWT signing secret |
| `LOG_LEVEL` | `info` | Logging level |

---

## Testing

### Running Tests

**Run all tests:**
```bash
go test ./...
```

**Run tests with coverage:**
```bash
go test -cover ./...
```

**Run tests with verbose output:**
```bash
go test -v ./...
```

**Run tests for specific package:**
```bash
go test ./internal/auth/
go test ./internal/config/
```

**Generate coverage report:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Manual API Testing

**Complete testing workflow:**

```bash
# 1. Start the server
go run cmd/main.go

# 2. Test health endpoint
curl http://localhost:8080/health

# 3. Get all pizzas (public)
curl http://localhost:8080/api/v1/public/pizzas

# 4. Get OAuth token (use dev client created by scripts/create_dev_client.go)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=dev-client" \
  -d "client_secret=dev-secret-123" \
  | jq -r '.access_token')

echo "Token: $TOKEN"

# 5. Create a pizza (admin only)
curl -X POST http://localhost:8080/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Pizza",
    "description": "Testing creation",
    "ingredients": ["cheese", "tomato"],
    "price": 15.99
  }'

# 6. List all pizzas again
curl http://localhost:8080/api/v1/public/pizzas

# 7. Update pizza
curl -X PUT http://localhost:8080/api/v1/protected/admin/pizzas/4 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Pizza",
    "price": 16.99
  }'

# 8. Delete pizza
curl -X DELETE http://localhost:8080/api/v1/protected/admin/pizzas/4 \
  -H "Authorization: Bearer $TOKEN"
```

### Using Swagger UI for Testing

1. **Start server:** `go run cmd/main.go`
2. **Open Swagger:** http://localhost:8080/swagger/index.html
3. **Get OAuth token** using curl or Postman
4. **Click "Authorize"** button in Swagger UI
5. **Enter:** `Bearer YOUR_ACCESS_TOKEN`
6. **Test endpoints** directly from Swagger

### Testing with Postman

**1. Import collection:**

Create a Postman collection with these requests:

```json
{
  "info": {
    "name": "Pizza API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "auth": {
    "type": "bearer",
    "bearer": [{"key": "token", "value": "{{access_token}}"}]
  },
  "item": [
    {
      "name": "Get Token",
      "request": {
        "method": "POST",
        "url": "{{base_url}}/api/v1/oauth/token",
        "body": {
          "mode": "urlencoded",
          "urlencoded": [
            {"key": "grant_type", "value": "client_credentials"},
            {"key": "client_id", "value": "{{client_id}}"},
            {"key": "client_secret", "value": "{{client_secret}}"}
          ]
        }
      }
    }
  ]
}
```

**2. Set environment variables:**
- `base_url`: `http://localhost:8080`
- `client_id`: Your OAuth client ID
- `client_secret`: Your OAuth client secret
- `access_token`: (auto-filled from token response)

### Test Checklist

Before submitting changes, verify:

- [ ] All tests pass: `go test ./...`
- [ ] Code is formatted: `gofmt -w .`
- [ ] Swagger docs updated: `swag init -g cmd/main.go`
- [ ] API endpoints work manually
- [ ] Authentication/authorization works correctly
- [ ] Error cases handled properly
- [ ] No sensitive data in logs
- [ ] Database migrations work

---

## Deployment

### Docker Deployment

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

### Production Considerations

**Security:**
- ✅ Use strong JWT secrets (minimum 32 characters)
- ✅ Enable HTTPS (TLS/SSL certificates)
- ✅ Set `GIN_MODE=release`
- ✅ Use environment variables, not `.env` files
- ✅ Implement rate limiting
- ✅ Enable CORS properly
- ✅ Keep dependencies updated

**Database:**
- Consider migrating from SQLite to PostgreSQL/MySQL for:
  - Better concurrent access
  - Production-grade reliability
  - Horizontal scaling support
- Use database connection pooling
- Implement backup strategy

**Monitoring:**
- Add health check endpoint monitoring
- Implement structured logging
- Set up error tracking (e.g., Sentry)
- Monitor API response times
- Track OAuth token usage

**Kubernetes Deployment** (Recommended for production):

See `docs/KUBERNETES.md` for complete Kubernetes deployment guide including:
- Deployment manifests
- Service configuration
- ConfigMaps and Secrets
- Ingress with TLS
- HorizontalPodAutoscaler
- PersistentVolumeClaims

### Environment-Specific Configuration

**Development:**
```env
APP_ENV=development
LOG_LEVEL=debug
GIN_MODE=debug
```

**Staging:**
```env
APP_ENV=staging
LOG_LEVEL=info
GIN_MODE=release
```

**Production:**
```env
APP_ENV=production
LOG_LEVEL=warn
GIN_MODE=release
APP_HOST=0.0.0.0
```

---

## Troubleshooting

### Common Issues

**1. Port already in use:**
```bash
# Find process using port 8080
lsof -ti:8080

# Kill the process
lsof -ti:8080 | xargs kill -9
```

**2. Database locked:**
```bash
# SQLite database is locked
rm test.sqlite
go run cmd/main.go  # Recreate database
```

**3. OAuth token invalid:**
```
Error: invalid_token
```
- Check if token has expired (default: 7200 seconds / 2 hours)
- Verify JWT_SECRET matches between token creation and validation
- Ensure Bearer prefix in Authorization header

**4. Token generation failed:**
```
Error: token_generation_failed or server_error
```
- Verify OAuth client has all required fields populated:
  - `user_id`: Required for token generation (falls back to client ID if NULL)
  - `domain`: Client's authorized domain (e.g., "http://localhost")
  - `grant_types`: Must include "client_credentials"
- Use `go run scripts/create_dev_client.go` to create a properly configured client
- Check server logs for detailed error messages

**4. Permission denied (403):**
```
Error: insufficient_permissions
```
- Verify your OAuth client has admin role
- Check token contains correct role claim

**5. Swagger docs out of date:**
```bash
# Regenerate Swagger documentation
swag init -g cmd/main.go
```

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug go run cmd/main.go
```

Check JWT token contents:
```bash
# Decode JWT (header.payload.signature)
echo "YOUR_TOKEN" | cut -d'.' -f2 | base64 -d | jq
```

---

## Contributing

We welcome contributions! Please follow these guidelines:

### Process

1. **Fork the repository**
2. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes**
4. **Add tests** for new functionality
5. **Update documentation:**
   - Add Swagger annotations to new endpoints
   - Update README if needed
   - Regenerate Swagger: `swag init -g cmd/main.go`
6. **Ensure tests pass:**
   ```bash
   go test ./...
   ```
7. **Format code:**
   ```bash
   gofmt -w .
   ```
8. **Commit changes:**
   ```bash
   git commit -m "feat: add new feature"
   ```
9. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```
10. **Create Pull Request**

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding tests
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

**Examples:**
```
feat: add pizza search endpoint
fix: correct OAuth token expiration
docs: update API documentation
test: add unit tests for pizza service
```

### Code Review Checklist

- [ ] Code follows Go conventions
- [ ] Tests added and passing
- [ ] Swagger documentation updated
- [ ] No breaking changes (or documented)
- [ ] Security considerations addressed
- [ ] Performance impact considered

---

## License

[MIT License](LICENSE)

---

## Support

- **Issues:** https://github.com/franciscosanchezn/gin-pizza-api/issues
- **Documentation:** http://localhost:8080/swagger/index.html
- **Email:** francisco@example.com

---

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

---

## Acknowledgments

- Built with [Gin Web Framework](https://github.com/gin-gonic/gin)
- Authentication via [go-oauth2/oauth2](https://github.com/go-oauth2/oauth2)
- Documentation with [Swag](https://github.com/swaggo/swag)
- Database with [GORM](https://gorm.io/)

---

**Last Updated:** November 10, 2025
