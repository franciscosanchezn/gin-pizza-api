# Development Guide

This guide covers the internal development workflow, project structure, coding standards, and best practices for contributing to the Pizza API codebase.

---

## Table of Contents

- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Database Management](#database-management)
- [Adding New Endpoints](#adding-new-endpoints)
- [Swagger Annotations](#swagger-annotations)
- [Code Style and Formatting](#code-style-and-formatting)
- [Environment Variables](#environment-variables)

---

## Project Structure

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
└── README.md                # Project overview
```

### Directory Responsibilities

**`cmd/`**: Application entry points and main initialization logic.

**`internal/auth/`**: OAuth2 server implementation, JWT token generation, and credential storage.

**`internal/config/`**: Environment variable parsing and application configuration.

**`internal/controllers/`**: HTTP request handlers (Gin handlers). Thin layer that delegates to services.

**`internal/middleware/`**: HTTP middleware for authentication, authorization, and logging.

**`internal/models/`**: Data models (structs) and GORM schema definitions.

**`internal/services/`**: Business logic layer. Controllers call services, services interact with database.

**`internal/logging/`**: Centralized logging configuration.

**`docs/`**: Auto-generated Swagger documentation files (do not edit manually).

---

## Development Workflow

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

---

## Database Management

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

### Switching to PostgreSQL/MySQL

For production or more advanced development scenarios:

1. Update `DATABASE_URL` in `.env`:
   ```env
   DATABASE_URL=postgres://user:pass@localhost/dbname
   ```

2. Install the appropriate driver (GORM handles the rest):
   ```bash
   go get gorm.io/driver/postgres
   ```

3. No code changes required - GORM auto-migrates schemas.

---

## Adding New Endpoints

Follow this workflow to add a new endpoint:

### 1. Define the Model

In `internal/models/`, create or update a model:

```go
type Pizza struct {
    ID          int            `json:"id" gorm:"primaryKey"`
    Name        string         `json:"name" gorm:"not null"`
    Price       float64        `json:"price"`
    CreatedBy   uint           `json:"created_by"`
    CreatedAt   time.Time      `json:"created_at"`
}
```

### 2. Create the Service Interface

In `internal/services/`, define the business logic interface:

```go
type PizzaService interface {
    GetAllPizzas() ([]models.Pizza, error)
    GetPizzaByID(id int) (*models.Pizza, error)
    CreatePizza(pizza *models.Pizza) error
    UpdatePizza(pizza *models.Pizza) error
    DeletePizza(id int) error
}
```

### 3. Implement the Controller

In `internal/controllers/`, add the HTTP handler:

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

### 4. Register the Route

In `cmd/main.go`, wire up the route:

```go
publicApi.GET("/pizzas", pizzaController.GetAllPizzas)
```

### 5. Regenerate Swagger Documentation

```bash
swag init -g cmd/main.go
```

This updates `docs/swagger.json` and `docs/swagger.yaml`.

---

## Swagger Annotations

Use these annotations in controller functions to document endpoints:

### Common Annotations

```go
// @Summary      Short description of the endpoint
// @Description  Long description with details
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

### Parameter Types

- **`path`**: URL parameter (e.g., `/pizzas/:id`)
- **`query`**: Query string parameter (e.g., `?name=Margherita`)
- **`body`**: Request body (JSON)
- **`header`**: HTTP header

### Example: Protected Endpoint

```go
// CreatePizza godoc
// @Summary Create a new pizza
// @Description Create a new pizza (requires admin role)
// @Tags pizzas
// @Accept json
// @Produce json
// @Param body body models.Pizza true "Pizza object"
// @Success 201 {object} models.Pizza
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/pizzas [post]
func (c *controller) CreatePizza(ctx *gin.Context) {
    // Implementation
}
```

### Regenerating Documentation

Always regenerate after adding/modifying endpoints:

```bash
swag init -g cmd/main.go
```

---

## Code Style and Formatting

### Formatting

**Format code before committing:**
```bash
gofmt -w .
```

**Recommended:** Configure your editor to run `gofmt` on save.

### Linting

**Install golangci-lint:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Run linter:**
```bash
golangci-lint run
```

### Code Conventions

- **Package naming**: Short, lowercase, no underscores (e.g., `auth`, `models`)
- **Struct naming**: PascalCase (e.g., `PizzaService`)
- **Function naming**: PascalCase for exported, camelCase for unexported
- **Error handling**: Always check errors, return early
- **Comments**: Use complete sentences, start with function/type name

**Example:**
```go
// GetAllPizzas retrieves all pizzas from the database.
// It returns an error if the database query fails.
func (s *service) GetAllPizzas() ([]models.Pizza, error) {
    var pizzas []models.Pizza
    if err := s.db.Find(&pizzas).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch pizzas: %w", err)
    }
    return pizzas, nil
}
```

---

## Environment Variables

During development, the app loads from `.env` file (see `.env.example`).

### Available Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment name (`development`, `staging`, `production`) |
| `APP_PORT` | `8080` | Server port |
| `APP_HOST` | `localhost` | Server host (use `0.0.0.0` for Docker) |
| `DATABASE_URL` | `sqlite://test.sqlite` | Database connection string |
| `DB_NAME` | `test.sqlite` | Database name |
| `DB_USER` | `admin` | Database user (PostgreSQL/MySQL only) |
| `DB_PASSWORD` | `secret` | Database password (PostgreSQL/MySQL only) |
| `JWT_SECRET` | *(required)* | JWT signing secret (minimum 32 characters) |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `GIN_MODE` | `debug` | Gin framework mode (`debug`, `release`) |

### Configuration Loading

The app uses `godotenv` to load `.env` files:

```go
import "github.com/joho/godotenv"

func init() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using system environment")
    }
}
```

### Best Practices

- ✅ Never commit `.env` files
- ✅ Use `.env.example` as a template
- ✅ Generate strong JWT secrets: `openssl rand -base64 32`
- ✅ Use environment variables in production (not `.env` files)
- ✅ Validate required variables on startup

---

## Testing

### Unit Tests

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -v -cover ./...
```

Run tests with race detector:
```bash
go test -v -race ./...
```

### Integration Tests

The project includes a comprehensive integration test suite:

```bash
./scripts/test-api.sh
```

This script validates:
- OAuth2 token acquisition
- Public endpoint access
- Protected endpoint authorization
- Full CRUD lifecycle
- Resource ownership rules

See [README.md](../../README.md#testing) for details.

---

## Additional Resources

- [Operations Guide](OPERATIONS.md) - Deployment, monitoring, troubleshooting
- [Contributing Guide](CONTRIBUTING.md) - Contribution process and code review
- [JWT Internals](JWT_INTERNALS.md) - Deep dive into authentication architecture
- [API Contract](../API_CONTRACT.md) - Formal API specifications
- [Terraform Provider Guide](../TERRAFORM_PROVIDER.md) - Building providers against this API
