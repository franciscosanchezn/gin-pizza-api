#!/bin/bash
# Developer Setup Script for Pizza API
# This script automates the initial setup for developers

set -e

echo "🍕 Pizza API Developer Setup"
echo "=============================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo "📋 Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go is not installed${NC}"
    echo "  Please install Go 1.21+ from https://golang.org/dl/"
    exit 1
fi
echo -e "${GREEN}✓ Go $(go version | awk '{print $3}')${NC}"

# Check Git
if ! command -v git &> /dev/null; then
    echo -e "${RED}✗ Git is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Git $(git --version | awk '{print $3}')${NC}"

# Check jq (optional but helpful)
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠ jq not found (optional but recommended)${NC}"
    echo "  Install: sudo apt-get install jq"
else
    echo -e "${GREEN}✓ jq${NC}"
fi

echo ""
echo "📦 Installing dependencies..."

# Install Go dependencies
go mod download
echo -e "${GREEN}✓ Go dependencies installed${NC}"

# Install Swagger CLI
echo "Installing Swagger CLI..."
go install github.com/swaggo/swag/cmd/swag@latest
echo -e "${GREEN}✓ Swagger CLI installed${NC}"

# Install Air (optional)
read -p "Install Air for hot reload? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    go install github.com/air-verse/air@latest
    echo -e "${GREEN}✓ Air installed${NC}"
fi

# Install golangci-lint (optional)
read -p "Install golangci-lint for code linting? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    echo -e "${GREEN}✓ golangci-lint installed${NC}"
fi

echo ""
echo "⚙️  Setting up configuration..."

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}✓ .env file created from .env.example${NC}"
    else
        # Create basic .env
        cat > .env << EOF
APP_ENV=development
APP_PORT=8080
APP_HOST=localhost
DATABASE_URL=sqlite://test.sqlite
DB_NAME=test.sqlite
DB_USER=admin
DB_PASSWORD=admin
LOG_LEVEL=debug
JWT_SECRET=$(openssl rand -base64 32)
EOF
        echo -e "${GREEN}✓ .env file created with generated JWT secret${NC}"
    fi
else
    echo -e "${YELLOW}⚠ .env file already exists, skipping${NC}"
fi

# Generate JWT secret if not present
if ! grep -q "JWT_SECRET" .env || grep -q "JWT_SECRET=$" .env || grep -q 'JWT_SECRET=""' .env; then
    JWT_SECRET=$(openssl rand -base64 32)
    echo "JWT_SECRET=$JWT_SECRET" >> .env
    echo -e "${GREEN}✓ Generated JWT secret${NC}"
fi

echo ""
echo "🗄️  Setting up database..."

# Remove old database if exists
if [ -f test.sqlite ]; then
    read -p "Remove existing database? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm test.sqlite
        echo -e "${GREEN}✓ Old database removed${NC}"
    fi
fi

echo ""
echo "🔨 Building application..."
go build -o bin/pizza-api cmd/main.go
echo -e "${GREEN}✓ Application built successfully${NC}"

echo ""
echo "🧪 Running tests..."
if go test ./...; then
    echo -e "${GREEN}✓ All tests passed${NC}"
else
    echo -e "${RED}✗ Some tests failed${NC}"
fi

echo ""
echo "📚 Generating Swagger documentation..."
swag init -g cmd/main.go
echo -e "${GREEN}✓ Swagger docs generated${NC}"

echo ""
echo "🎯 Creating development OAuth client..."

# Create a helper script to create OAuth client
cat > scripts/create_dev_client.go << 'EOF'
package main

import (
	"fmt"
	"log"
	"time"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type OAuthClient struct {
	ID        string    `gorm:"primaryKey"`
	Secret    string    `gorm:"not null"`
	Name      string    `gorm:"not null"`
	Scopes    string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	clientID := "dev-client"
	clientSecret := "dev-secret-123"

	// Check if client already exists
	var existing OAuthClient
	if err := db.Where("id = ?", clientID).First(&existing).Error; err == nil {
		fmt.Println("Development client already exists!")
		fmt.Printf("Client ID: %s\n", clientID)
		fmt.Printf("Client Secret: %s\n", clientSecret)
		return
	}

	// Create new client
	hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash secret:", err)
	}

	client := OAuthClient{
		ID:        clientID,
		Secret:    string(hash),
		Name:      "Development Client",
		Scopes:    "read write",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.Create(&client).Error; err != nil {
		log.Fatal("Failed to create client:", err)
	}

	fmt.Println("✓ Development OAuth client created!")
	fmt.Printf("Client ID: %s\n", clientID)
	fmt.Printf("Client Secret: %s\n", clientSecret)
	fmt.Println("\nUse these credentials for testing:")
	fmt.Printf("curl -X POST http://localhost:8080/api/v1/oauth/token \\\n")
	fmt.Printf("  -d 'grant_type=client_credentials' \\\n")
	fmt.Printf("  -d 'client_id=%s' \\\n", clientID)
	fmt.Printf("  -d 'client_secret=%s'\n", clientSecret)
}
EOF

mkdir -p scripts

# Start the server in background to initialize database
echo "Starting server to initialize database..."
go run cmd/main.go > /dev/null 2>&1 &
SERVER_PID=$!
sleep 3

# Create dev client
if [ -f test.sqlite ]; then
    go run scripts/create_dev_client.go
else
    echo -e "${YELLOW}⚠ Database not found, skipping OAuth client creation${NC}"
fi

# Stop the server
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "✅ Setup complete!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Quick Start Commands:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  Start server:"
echo "    go run cmd/main.go"
echo ""
echo "  Start with hot reload:"
echo "    air"
echo ""
echo "  Run tests:"
echo "    go test ./..."
echo ""
echo "  View API docs:"
echo "    http://localhost:8080/swagger/index.html"
echo ""
echo "  Test OAuth flow:"
echo "    curl -X POST http://localhost:8080/api/v1/oauth/token \\"
echo "      -d 'grant_type=client_credentials' \\"
echo "      -d 'client_id=dev-client' \\"
echo "      -d 'client_secret=dev-secret-123'"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📖 Documentation:"
echo "   - README.md - Project overview and quick start"
echo "   - docs/internal/ - Development, operations, and contributing guides"
echo ""
echo "Happy coding! 🎉"
