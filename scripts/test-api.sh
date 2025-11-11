#!/bin/bash
# Comprehensive API Testing Script
# Tests the complete Pizza API workflow

set -e

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
CLIENT_ID="${CLIENT_ID:-dev-client}"
CLIENT_SECRET="${CLIENT_SECRET:-dev-secret-123}"
SERVER_STARTUP_WAIT=3

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Separator
SEPARATOR="======================================================================"

echo ""
echo -e "${CYAN}${SEPARATOR}${NC}"
echo -e "${CYAN}Pizza API - Comprehensive Test Suite${NC}"
echo -e "${CYAN}${SEPARATOR}${NC}"
echo ""

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}>> $1${NC}"
    echo -e "${BLUE}${SEPARATOR}${NC}"
}

# Function to print test status
test_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

test_fail() {
    echo -e "${RED}✗${NC} $1"
    echo ""
    echo -e "${CYAN}Server logs:${NC}"
    tail -20 /tmp/pizza-api.log 2>/dev/null || echo "No logs available"
    exit 1
}

test_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

# Function to print JSON responses
print_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq '.'
    else
        echo "$1"
    fi
}

# ============================================================================
# STEP 1: CLEANUP AND START SERVER
# ============================================================================
print_section "Step 1: Environment Cleanup & Server Start"

# Kill any existing pizza-api processes
test_info "Stopping any running pizza-api servers..."
pkill -f pizza-api 2>/dev/null || true
sleep 1

# Check if port is free
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    test_fail "Port 8080 is still in use. Please free the port manually."
fi
test_pass "Port 8080 is available"

# Build the application
test_info "Building pizza-api..."
if go build -o bin/pizza-api cmd/main.go 2>&1 | grep -qi "error"; then
    test_fail "Failed to build application"
fi
test_pass "Application built successfully"

# Start the server in background
test_info "Starting pizza-api server..."
nohup ./bin/pizza-api > /tmp/pizza-api.log 2>&1 &
SERVER_PID=$!
test_info "Server PID: $SERVER_PID"

# Wait for server to start
test_info "Waiting ${SERVER_STARTUP_WAIT} seconds for server to start..."
sleep $SERVER_STARTUP_WAIT

# Verify server is running
if ! ps -p $SERVER_PID > /dev/null 2>&1; then
    test_fail "Server failed to start. Check /tmp/pizza-api.log for errors."
fi

# Test health endpoint
if curl -sf $BASE_URL/health > /dev/null; then
    test_pass "Server is running and healthy"
else
    test_fail "Server health check failed"
fi

# ============================================================================
# STEP 2: GET OAUTH TOKEN
# ============================================================================
print_section "Step 2: OAuth Token Acquisition"

test_info "Requesting OAuth token with client credentials..."
TOKEN_RESPONSE=$(curl -sf -X POST $BASE_URL/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=$CLIENT_ID" \
  -d "client_secret=$CLIENT_SECRET")

if [ $? -ne 0 ]; then
    test_fail "OAuth token request failed. Make sure dev client exists (run: go run scripts/create_dev_client.go)"
fi

TOKEN=$(echo $TOKEN_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
EXPIRES_IN=$(echo $TOKEN_RESPONSE | grep -o '"expires_in":[0-9]*' | cut -d':' -f2)

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    test_fail "Failed to extract access token from response"
fi

test_pass "OAuth token acquired successfully"
test_info "Token: ${TOKEN:0:30}..."
test_info "Expires in: ${EXPIRES_IN}s"

# ============================================================================
# STEP 3: LIST PUBLIC PIZZAS
# ============================================================================
print_section "Step 3: List Public Pizzas (No Auth Required)"

test_info "Fetching all pizzas from public endpoint..."
PUBLIC_PIZZAS=$(curl -sf $BASE_URL/api/v1/public/pizzas)

if [ $? -ne 0 ]; then
    test_fail "Failed to fetch public pizzas"
fi

PIZZA_COUNT=$(echo $PUBLIC_PIZZAS | grep -o '"id":' | wc -l)
test_pass "Found $PIZZA_COUNT pizzas in the database"

if command -v jq &> /dev/null; then
    echo "$PUBLIC_PIZZAS" | jq -r '.[] | "  • \(.name) - $\(.price)"'
fi

# ============================================================================
# STEP 4: CREATE A NEW PIZZA
# ============================================================================
print_section "Step 4: Create New Pizza (Admin Auth Required)"

PIZZA_NAME="Test Pizza $(date +%s)"
test_info "Creating pizza: $PIZZA_NAME"

CREATE_RESPONSE=$(curl -sf -X POST $BASE_URL/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"$PIZZA_NAME\",
    \"description\": \"Automated test pizza with all the best toppings\",
    \"ingredients\": [\"mozzarella\", \"tomato sauce\", \"basil\", \"olive oil\"],
    \"price\": 19.99
  }")

if [ $? -ne 0 ]; then
    test_fail "Failed to create pizza"
fi

PIZZA_ID=$(echo $CREATE_RESPONSE | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -z "$PIZZA_ID" ]; then
    test_fail "Failed to extract pizza ID from response"
fi

test_pass "Pizza created successfully with ID: $PIZZA_ID"
echo ""
test_info "Pizza details:"
print_json "$CREATE_RESPONSE"

# ============================================================================
# STEP 5: LIST PUBLIC PIZZAS AGAIN
# ============================================================================
print_section "Step 5: List Public Pizzas Again (Verify Creation)"

test_info "Fetching updated pizza list..."
PIZZAS_AFTER=$(curl -sf $BASE_URL/api/v1/public/pizzas)

if [ $? -ne 0 ]; then
    test_fail "Failed to fetch pizzas after creation"
fi

NEW_PIZZA_COUNT=$(echo $PIZZAS_AFTER | grep -o '"id":' | wc -l)
test_pass "Now found $NEW_PIZZA_COUNT pizzas (was $PIZZA_COUNT)"

if [ $NEW_PIZZA_COUNT -le $PIZZA_COUNT ]; then
    test_fail "Pizza count did not increase after creation"
fi

# ============================================================================
# STEP 6: GET SPECIFIC PIZZA
# ============================================================================
print_section "Step 6: Get Specific Pizza by ID"

test_info "Fetching pizza #$PIZZA_ID..."
PIZZA_DETAILS=$(curl -sf $BASE_URL/api/v1/public/pizzas/$PIZZA_ID)

if [ $? -ne 0 ]; then
    test_fail "Failed to fetch pizza #$PIZZA_ID"
fi

FETCHED_NAME=$(echo $PIZZA_DETAILS | grep -o '"name":"[^"]*"' | cut -d'"' -f4)

if [ "$FETCHED_NAME" != "$PIZZA_NAME" ]; then
    test_fail "Pizza name mismatch. Expected: $PIZZA_NAME, Got: $FETCHED_NAME"
fi

test_pass "Successfully fetched pizza: $FETCHED_NAME"
echo ""
test_info "Pizza details:"
print_json "$PIZZA_DETAILS"

# ============================================================================
# STEP 7: UPDATE THE PIZZA
# ============================================================================
print_section "Step 7: Update Pizza"

UPDATED_NAME="Updated $PIZZA_NAME"
UPDATED_PRICE="24.99"

test_info "Updating pizza #$PIZZA_ID..."
test_info "New name: $UPDATED_NAME"
test_info "New price: \$$UPDATED_PRICE"

UPDATE_RESPONSE=$(curl -sf -X PUT $BASE_URL/api/v1/protected/admin/pizzas/$PIZZA_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"$UPDATED_NAME\",
    \"description\": \"This pizza has been updated via API test\",
    \"ingredients\": [\"mozzarella\", \"tomato sauce\", \"basil\", \"olive oil\", \"parmesan\"],
    \"price\": $UPDATED_PRICE
  }")

if [ $? -ne 0 ]; then
    test_fail "Failed to update pizza"
fi

test_pass "Pizza updated successfully"
echo ""
test_info "Updated pizza details:"
print_json "$UPDATE_RESPONSE"

# ============================================================================
# STEP 8: GET UPDATED PIZZA
# ============================================================================
print_section "Step 8: Verify Pizza Update"

test_info "Fetching updated pizza #$PIZZA_ID..."
UPDATED_PIZZA=$(curl -sf $BASE_URL/api/v1/public/pizzas/$PIZZA_ID)

if [ $? -ne 0 ]; then
    test_fail "Failed to fetch updated pizza"
fi

VERIFIED_NAME=$(echo $UPDATED_PIZZA | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
VERIFIED_PRICE=$(echo $UPDATED_PIZZA | grep -o '"price":[0-9.]*' | cut -d':' -f2)

if [ "$VERIFIED_NAME" != "$UPDATED_NAME" ]; then
    test_fail "Pizza name was not updated. Expected: $UPDATED_NAME, Got: $VERIFIED_NAME"
fi

if [ "$VERIFIED_PRICE" != "$UPDATED_PRICE" ]; then
    test_fail "Pizza price was not updated. Expected: $UPDATED_PRICE, Got: $VERIFIED_PRICE"
fi

test_pass "Pizza update verified successfully"
test_info "Name: $VERIFIED_NAME"
test_info "Price: \$$VERIFIED_PRICE"

# ============================================================================
# STEP 9: DELETE THE PIZZA
# ============================================================================
print_section "Step 9: Delete Pizza"

test_info "Deleting pizza #$PIZZA_ID..."
DELETE_RESPONSE=$(curl -sf -X DELETE $BASE_URL/api/v1/protected/admin/pizzas/$PIZZA_ID \
  -H "Authorization: Bearer $TOKEN")

if [ $? -ne 0 ]; then
    test_fail "Failed to delete pizza"
fi

test_pass "Pizza deleted successfully"

# Verify deletion
test_info "Verifying pizza was deleted..."
if curl -sf $BASE_URL/api/v1/public/pizzas/$PIZZA_ID > /dev/null 2>&1; then
    test_fail "Pizza still exists after deletion"
fi

test_pass "Confirmed pizza #$PIZZA_ID no longer exists"

# ============================================================================
# CLEANUP
# ============================================================================
print_section "Cleanup"

test_info "Stopping test server (PID: $SERVER_PID)..."
kill $SERVER_PID 2>/dev/null || true
sleep 1

if ps -p $SERVER_PID > /dev/null 2>&1; then
    test_info "Force killing server..."
    kill -9 $SERVER_PID 2>/dev/null || true
fi

test_pass "Server stopped"

# ============================================================================
# SUMMARY
# ============================================================================
echo ""
echo -e "${CYAN}${SEPARATOR}${NC}"
echo -e "${GREEN}✅ All Tests Passed Successfully!${NC}"
echo -e "${CYAN}${SEPARATOR}${NC}"
echo ""
echo -e "${CYAN}Test Summary:${NC}"
echo -e "  ${GREEN}✓${NC} Environment cleanup and server startup"
echo -e "  ${GREEN}✓${NC} OAuth token acquisition"
echo -e "  ${GREEN}✓${NC} List public pizzas (initial)"
echo -e "  ${GREEN}✓${NC} Create new pizza"
echo -e "  ${GREEN}✓${NC} List public pizzas (after creation)"
echo -e "  ${GREEN}✓${NC} Get specific pizza by ID"
echo -e "  ${GREEN}✓${NC} Update pizza"
echo -e "  ${GREEN}✓${NC} Verify pizza update"
echo -e "  ${GREEN}✓${NC} Delete pizza"
echo ""
echo -e "${CYAN}Server logs available at: /tmp/pizza-api.log${NC}"
echo ""
