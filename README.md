# gin-pizza-api

A simple REST API built with Go and Gin to retrieve different pizza types - a learning project

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [Development](#development)
- [Testing](#testing)

## Features

- RESTful API for pizza management
- JWT-based authentication and authorization
- Role-based access control (Admin/User)
- Swagger/OpenAPI documentation
- SQLite database with GORM
- Environment-based configuration
- Structured logging

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

1. **Create environment file:**

```bash
cp .env.example .env
```

2. **Update `.env` file with your configuration:**

```env
APP_ENV=development
APP_PORT=8080
APP_HOST=localhost
DATABASE_URL=postgres://user:password@localhost:5432/mydb
DB_NAME=mydb
DB_USER=user
DB_PASSWORD=password
LOG_LEVEL=info
JWT_SECRET="your-super-secret-256-bit-key-here-make-it-long-and-random"
```

3. **Generate a secure JWT secret:**

```bash
openssl rand -base64 32
```

## API Documentation

### Generating Swagger Documentation

The API uses Swagger/OpenAPI for documentation. To regenerate the documentation:

1. **Generate Swagger docs:**

```bash
swag init -g cmd/main.go -o docs
```

2. **Start the server:**

```bash
go run cmd/main.go
```

3. **Access Swagger UI:**

```
http://localhost:8080/swagger/index.html
```

### API Endpoints

#### Public Endpoints

- `GET /health` - Health check
- `GET /api/v1/public/pizzas` - Get all pizzas
- `GET /api/v1/public/pizzas/{id}` - Get pizza by ID

#### Development Endpoints

- `GET /test-token` - Generate test JWT token (remove in production)

#### Protected Endpoints (Requires JWT)

- `POST /api/v1/protected/admin/pizzas` - Create pizza (Admin only)
- `PUT /api/v1/protected/admin/pizzas/{id}` - Update pizza (Admin only)
- `DELETE /api/v1/protected/admin/pizzas/{id}` - Delete pizza (Admin only)

### Authentication

The API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

**Getting a test token for development:**

```bash
curl http://localhost:8080/test-token
```

**Using the token in requests:**

```bash
curl -H "Authorization: Bearer <token>" \
     -X POST http://localhost:8080/api/v1/protected/admin/pizzas \
     -H "Content-Type: application/json" \
     -d '{"name":"Margherita","price":12.99,"ingredients":["tomato","mozzarella","basil"]}'
```

## Development

### Running the Application

1. **Development mode:**

```bash
go run cmd/main.go
```

2. **Build and run:**

```bash
go build -o gin-pizza-api cmd/main.go
./gin-pizza-api
```

### Project Structure

```
gin-pizza-api/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── controllers/         # HTTP handlers
│   ├── middleware/          # Authentication middleware
│   ├── models/              # Data models
│   └── services/            # Business logic
├── docs/                    # Generated Swagger documentation
├── .env                     # Environment variables
├── go.mod                   # Go module file
└── README.md
```

### Adding New Swagger Documentation

When adding new endpoints, include Swagger annotations:

```go
// CreatePizza godoc
// @Summary Create a new pizza
// @Description Create a new pizza with the input payload
// @Tags pizzas
// @Accept json
// @Produce json
// @Param pizza body models.Pizza true "Pizza object"
// @Security BearerAuth
// @Success 201 {object} models.Pizza
// @Failure 400 {object} map[string]string
// @Router /api/v1/protected/admin/pizzas [post]
func (c *controller) CreatePizza(ctx *gin.Context) {
    // Implementation
}
```

**Regenerate documentation after changes:**

```bash
swag init -g cmd/main.go -o docs
```

## Testing

### Manual Testing with curl

1. **Test public endpoints:**

```bash
# Health check
curl http://localhost:8080/health

# Get all pizzas
curl http://localhost:8080/api/v1/public/pizzas
```

2. **Test protected endpoints:**

```bash
# Get test token
TOKEN=$(curl -s http://localhost:8080/test-token | jq -r '.token')

# Create pizza (admin required)
curl -H "Authorization: Bearer $TOKEN" \
     -X POST http://localhost:8080/api/v1/protected/admin/pizzas \
     -H "Content-Type: application/json" \
     -d '{"name":"Test Pizza","price":15.99,"ingredients":["cheese","tomato"]}'
```

### Using Swagger UI

1. Open `http://localhost:8080/swagger/index.html`
2. Click "Authorize" button
3. Enter: `Bearer <your-test-token>`
4. Test endpoints directly from the UI

## Security Notes

- Never commit `.env` files to version control
- Use strong, random JWT secrets in production
- The `/test-token` endpoint should be removed in production
- Consider implementing refresh tokens for better security

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add Swagger documentation for new endpoints
4. Regenerate docs with `swag init -g cmd/main.go -o docs`
5. Test your changes
6. Submit a pull request
