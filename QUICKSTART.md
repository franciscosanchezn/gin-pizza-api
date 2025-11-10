# Quick Start Guide - Pizza API

Get up and running with the Pizza API in 5 minutes!

> **✨ Latest Update (Nov 10, 2025):** OAuth2 Client Credentials flow has been enhanced with improved field validation. Use the provided `scripts/create_dev_client.go` script to create properly configured OAuth clients.

---

## For Users (Just Want to Use the API)

### 1. Start the Server

```bash
git clone https://github.com/franciscosanchezn/gin-pizza-api.git
cd gin-pizza-api
go run cmd/main.go
```

Server starts on: `http://localhost:8080`

### 2. Test Public Endpoints (No Auth Required)

```bash
# Health check
curl http://localhost:8080/health

# View all pizzas
curl http://localhost:8080/api/v1/public/pizzas | jq

# Get specific pizza
curl http://localhost:8080/api/v1/public/pizzas/1 | jq
```

### 3. Create an OAuth Client

**Option A: Use Development Script (Recommended)**

```bash
# Run the development client creation script
go run scripts/create_dev_client.go
```

This creates a client with:
- **Client ID:** `dev-client`
- **Client Secret:** `dev-secret-123`
- **Domain:** `http://localhost`
- **UserID:** `1` (required for token generation)
- **Scopes:** `read write`
- **Grant Types:** `client_credentials`

**Option B: Create Custom Client via Database**

```bash
# Generate bcrypt hash for your secret
go run -exec 'package main; import ("fmt"; "golang.org/x/crypto/bcrypt"); func main() { hash, _ := bcrypt.GenerateFromPassword([]byte("my-secret-123"), bcrypt.DefaultCost); fmt.Println(string(hash)) }'

# Insert into SQLite database (note: includes required fields)
sqlite3 test.sqlite "INSERT INTO oauth_clients (id, secret, name, domain, user_id, scopes, grant_types, created_at, updated_at) VALUES ('my-client', '\$2a\$10\$...hash...', 'My Client', 'http://localhost', 1, 'read write', 'client_credentials', datetime('now'), datetime('now'));"
```

**Option C: Use Admin API (Production)**

Contact your administrator to create an OAuth client for you.

### 4. Get Access Token

**Using the dev client created above:**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=dev-client" \
  -d "client_secret=dev-secret-123" \
  | jq -r '.access_token')

echo "Token: $TOKEN"
```

**Or with your custom client:**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=my-client" \
  -d "client_secret=my-secret-123" \
  | jq -r '.access_token')

echo "Token: $TOKEN"
```

### 5. Use the API

```bash
# Create a pizza
curl -X POST http://localhost:8080/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Special Pizza",
    "description": "Delicious custom pizza",
    "ingredients": ["mozzarella", "tomatoes", "basil", "olive oil"],
    "price": 14.99
  }' | jq

# Update a pizza
curl -X PUT http://localhost:8080/api/v1/protected/admin/pizzas/4 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Updated Pizza",
    "price": 16.99
  }' | jq

# Delete a pizza
curl -X DELETE http://localhost:8080/api/v1/protected/admin/pizzas/4 \
  -H "Authorization: Bearer $TOKEN"
```

### 6. Explore Interactive Docs

Open in browser: http://localhost:8080/swagger/index.html

Click "Authorize" → Enter: `Bearer YOUR_TOKEN` → Test endpoints!

---

## For Developers (Setting Up Development Environment)

### Prerequisites

- Go 1.21+
- Git
- (Optional) Docker
- (Optional) SQLite CLI

### Setup (5 Steps)

#### 1. Clone & Install

```bash
git clone https://github.com/franciscosanchezn/gin-pizza-api.git
cd gin-pizza-api
go mod download
```

#### 2. Install Dev Tools

```bash
# Swagger CLI (required for doc generation)
go install github.com/swaggo/swag/cmd/swag@latest

# Air (optional - hot reload)
go install github.com/air-verse/air@latest

# golangci-lint (optional - linting)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### 3. Configure Environment

```bash
# Copy example env file
cp .env.example .env

# Generate secure JWT secret
openssl rand -base64 32

# Edit .env file
nano .env
```

Minimal `.env`:
```env
JWT_SECRET=your-generated-secret-from-above
APP_PORT=8080
LOG_LEVEL=debug
```

#### 4. Run the Application

**Option A: Standard**
```bash
go run cmd/main.go
```

**Option B: Hot Reload**
```bash
air
```

**Option C: Docker**
```bash
docker build -t pizza-api .
docker run -p 8080:8080 -e JWT_SECRET=your-secret pizza-api
```

#### 5. Verify Installation

```bash
# Run tests
go test ./...

# Check health
curl http://localhost:8080/health

# View Swagger
open http://localhost:8080/swagger/index.html
```

### Development Workflow

**1. Make changes to code**

**2. Run tests**
```bash
go test ./...
```

**3. Format code**
```bash
gofmt -w .
```

**4. Update Swagger docs (if API changed)**
```bash
swag init -g cmd/main.go
```

**5. Commit & push**
```bash
git add .
git commit -m "feat: your feature"
git push
```

---

## Common Development Tasks

### Create a Test OAuth Client

**Method 1: Use Provided Script (Recommended)**

```bash
# Run the development client creation script
go run scripts/create_dev_client.go
```

This automatically creates a client with:
- **Client ID:** `dev-client`
- **Client Secret:** `dev-secret-123`
- **Domain:** `http://localhost`
- **UserID:** `1` (required for OAuth2 token generation)
- **Scopes:** `read write`
- **Grant Types:** `client_credentials`

The script checks if the client already exists and provides you with the credentials to use.

**Method 2: Direct SQL (Custom Client)**

```bash
sqlite3 test.sqlite
```

```sql
-- Generate a client with known credentials (includes all required fields)
INSERT INTO oauth_clients (id, secret, name, domain, user_id, scopes, grant_types, created_at, updated_at)
VALUES (
  'dev-client',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- password: "secret"
  'Development Client',
  'http://localhost',
  1,  -- Required for token generation
  'read write',
  'client_credentials',
  datetime('now'),
  datetime('now')
);
```

**Important:** The OAuth2 library requires the following fields for proper token generation:
- `user_id`: Used as the subject in JWT tokens (falls back to client ID if NULL)
- `domain`: Client's authorized domain
- `grant_types`: Space-separated list of allowed grant types (e.g., "client_credentials")

### Reset Database

```bash
rm test.sqlite
go run cmd/main.go
```

### Debug OAuth Tokens

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=dev-client" \
  -d "client_secret=secret" \
  | jq -r '.access_token')

# Decode JWT (base64)
echo $TOKEN | cut -d'.' -f2 | base64 -d | jq
```

### View Database

```bash
sqlite3 test.sqlite

# Show tables
.tables

# View pizzas
SELECT * FROM pizzas;

# View OAuth clients
SELECT id, name, scopes FROM oauth_clients;

# Exit
.quit
```

---

## Quick Testing Script

Save as `test_api.sh`:

```bash
#!/bin/bash
set -e

BASE_URL="http://localhost:8080"

echo "🍕 Pizza API Quick Test"
echo "======================="

# Health check
echo -n "✓ Health check... "
curl -sf $BASE_URL/health > /dev/null && echo "OK" || echo "FAIL"

# List pizzas
echo -n "✓ List pizzas... "
PIZZAS=$(curl -sf $BASE_URL/api/v1/public/pizzas)
echo "Found $(echo $PIZZAS | jq length) pizzas"

# Get OAuth token (update with your credentials)
echo -n "✓ Get OAuth token... "
TOKEN=$(curl -sf -X POST $BASE_URL/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=dev-client" \
  -d "client_secret=secret" \
  | jq -r '.access_token')

if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
    echo "OK"
else
    echo "FAIL - Check your OAuth credentials"
    exit 1
fi

# Create pizza
echo -n "✓ Create pizza... "
PIZZA=$(curl -sf -X POST $BASE_URL/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Pizza","price":19.99,"ingredients":["test"]}')
PIZZA_ID=$(echo $PIZZA | jq -r '.id')
echo "Created pizza #$PIZZA_ID"

# Update pizza
echo -n "✓ Update pizza... "
curl -sf -X PUT $BASE_URL/api/v1/protected/admin/pizzas/$PIZZA_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Pizza","price":21.99}' > /dev/null && echo "OK"

# Delete pizza
echo -n "✓ Delete pizza... "
curl -sf -X DELETE $BASE_URL/api/v1/protected/admin/pizzas/$PIZZA_ID \
  -H "Authorization: Bearer $TOKEN" > /dev/null && echo "OK"

echo ""
echo "✅ All tests passed!"
```

Run:
```bash
chmod +x test_api.sh
./test_api.sh
```

---

## Troubleshooting

### Port 8080 already in use

```bash
# Find and kill process
lsof -ti:8080 | xargs kill -9

# Or use different port
APP_PORT=8081 go run cmd/main.go
```

### Cannot get OAuth token

```bash
# Verify client exists and has required fields
sqlite3 test.sqlite "SELECT id, name, domain, user_id, grant_types FROM oauth_clients;"

# Create development client if missing
go run scripts/create_dev_client.go

# Check logs for errors
LOG_LEVEL=debug go run cmd/main.go
```

**Common OAuth Token Issues:**
- **Missing user_id field:** The OAuth2 library requires a valid user_id for token generation. Run `go run scripts/create_dev_client.go` to create a properly configured client.
- **Missing grant_types field:** Ensure the client has `grant_types` set to `client_credentials`.
- **Wrong credentials:** Double-check your client_id and client_secret match the database values.

### Swagger not working

```bash
# Regenerate Swagger docs
swag init -g cmd/main.go

# Restart server
go run cmd/main.go
```

### Database errors

```bash
# Reset database
rm test.sqlite
go run cmd/main.go
```

---

## Next Steps

- 📖 Read full documentation: [README.md](README.md)
- 🏗️ See architecture analysis: [.tasks/architect-analysis-20251110.md](.tasks/architect-analysis-20251110.md)
- 🧹 Review cleanup summary: [.tasks/cleanup-summary-20251110.md](.tasks/cleanup-summary-20251110.md)
- 🐳 Deploy with Docker: See Dockerfile
- ☸️ Deploy to Kubernetes: See [docs/KUBERNETES.md](docs/KUBERNETES.md) for complete deployment guide with HTTPS

---

**Questions?** Open an issue on GitHub!

**Last Updated:** November 10, 2025
