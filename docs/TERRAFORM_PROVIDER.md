# Terraform Provider Developer Guide

## Overview

This guide is for developers building a **Terraform provider** for the Pizza API. It covers authentication flows, resource mapping, error handling strategies, and testing approaches.

**Prerequisites:**
- Familiarity with Terraform provider development (Terraform Plugin Framework or SDK)
- Understanding of OAuth2 client credentials flow
- Go programming language knowledge

**Related Documentation:**
- [API_CONTRACT.md](./API_CONTRACT.md) - Detailed API specifications
- [README.md](../README.md) - General API usage and endpoints
- [KUBERNETES.md](./KUBERNETES.md) - Deployment instructions

---

## Table of Contents

1. [Authentication Flow](#authentication-flow)
2. [Provider Configuration](#provider-configuration)
3. [Resource Mapping](#resource-mapping)
4. [Error Handling](#error-handling)
5. [State Management](#state-management)
6. [Testing Strategy](#testing-strategy)
7. [Example Implementation](#example-implementation)

---

## Authentication Flow

### OAuth2 Client Credentials

The Pizza API uses OAuth2 Client Credentials grant for machine-to-machine authentication. Your provider must:

1. **Obtain credentials** (pre-configured by user)
2. **Request access token** from the API
3. **Use token** for subsequent requests
4. **Refresh token** when expired

### Step-by-Step Flow

#### Step 1: Request Access Token

```bash
curl -X POST https://pizza-api.local/api/v1/oauth/token \
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

#### Step 2: Use Access Token

Include the token in the `Authorization` header for all protected endpoints:

```bash
curl -X GET https://pizza-api.local/api/v1/protected/admin/pizzas \
  -H "Authorization: Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9..."
```

#### Step 3: Handle Token Expiration

- **Token lifetime:** 3600 seconds (1 hour) by default
- **No refresh token:** Request a new access token when expired
- **Detection:** API returns `401 Unauthorized` with `expired_token` error

**Provider Implementation Tip:**
Cache the access token and its expiration time. Request a new token when:
- No cached token exists
- Cached token is within 5 minutes of expiration
- API returns 401 with expired_token error

### Pseudo-Code: Token Management

```go
type Client struct {
    endpoint     string
    clientID     string
    clientSecret string
    token        string
    tokenExpiry  time.Time
}

func (c *Client) ensureToken() error {
    // Check if token is valid and not expiring soon
    if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-5*time.Minute)) {
        return nil
    }

    // Request new token
    resp, err := http.PostForm(c.endpoint+"/api/v1/oauth/token", url.Values{
        "grant_type":    {"client_credentials"},
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
    })
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var tokenResp struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return err
    }

    c.token = tokenResp.AccessToken
    c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
    return nil
}

func (c *Client) makeRequest(method, path string, body interface{}) (*http.Response, error) {
    if err := c.ensureToken(); err != nil {
        return nil, err
    }

    req, _ := http.NewRequest(method, c.endpoint+path, marshalBody(body))
    req.Header.Set("Authorization", "Bearer "+c.token)
    req.Header.Set("Content-Type", "application/json")
    
    return http.DefaultClient.Do(req)
}
```

---

## Provider Configuration

### HCL Configuration Example

```hcl
provider "pizza" {
  endpoint      = "https://pizza-api.local"
  client_id     = "terraform-client"
  client_secret = "your-secret-here"
}
```

### Configuration Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `endpoint` | string | Yes | Base URL of the Pizza API (e.g., `https://pizza-api.local`) |
| `client_id` | string | Yes | OAuth2 client ID (created via API or `create_dev_client.go` script) |
| `client_secret` | string | Yes | OAuth2 client secret (sensitive, shown only once at creation) |

### Environment Variable Support

Allow users to configure via environment variables:

```bash
export PIZZA_API_ENDPOINT="https://pizza-api.local"
export PIZZA_API_CLIENT_ID="terraform-client"
export PIZZA_API_CLIENT_SECRET="your-secret-here"

terraform apply
```

### Provider Schema (Terraform Plugin Framework)

```go
func (p *PizzaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "endpoint": schema.StringAttribute{
                Description: "Base URL of the Pizza API",
                Required:    true,
            },
            "client_id": schema.StringAttribute{
                Description: "OAuth2 client ID",
                Required:    true,
            },
            "client_secret": schema.StringAttribute{
                Description: "OAuth2 client secret",
                Required:    true,
                Sensitive:   true,
            },
        },
    }
}
```

---

## Resource Mapping

### Resource: `pizza_pizza`

Manages a pizza in the Pizza API.

**HCL Example:**
```hcl
resource "pizza_pizza" "margherita" {
  name        = "Margherita"
  description = "Classic Italian pizza with fresh basil"
  ingredients = ["Tomato Sauce", "Mozzarella", "Basil", "Olive Oil"]
  price       = 10.99
}
```

**API Mapping:**

| Terraform Attribute | API Field | Type | Notes |
|---------------------|-----------|------|-------|
| `id` (computed) | `id` | int | Set from API response after create |
| `name` | `name` | string | Required |
| `description` | `description` | string | Optional |
| `ingredients` | `ingredients` | []string | Required (at least one ingredient) |
| `price` | `price` | float64 | Required |
| `created_by` (computed) | `created_by` | int | Set by API based on OAuth client's user ID |
| `created_at` (computed) | `created_at` | string | ISO 8601 timestamp |
| `updated_at` (computed) | `updated_at` | string | ISO 8601 timestamp |

### CRUD Operations

#### Create
- **Endpoint:** `POST /api/v1/protected/admin/pizzas`
- **Auth Required:** Yes (Admin role)
- **Idempotent:** ❌ No (duplicate pizzas will be created)
- **Response:** Returns created pizza with `id` field

**Provider Responsibility:**
- Store returned `id` in Terraform state
- Do not retry on success or client errors (4xx)
- Handle validation errors gracefully

#### Read
- **Endpoint:** `GET /api/v1/public/pizzas/:id`
- **Auth Required:** No (public endpoint)
- **Idempotent:** ✅ Yes

**Provider Responsibility:**
- Handle 404 as "resource not found" (remove from state)
- Use for state refresh operations

#### Update
- **Endpoint:** `PUT /api/v1/protected/admin/pizzas/:id`
- **Auth Required:** Yes (Admin role)
- **Idempotent:** ✅ Yes
- **Behavior:** Full replacement (all fields must be provided)

**Provider Responsibility:**
- Send complete resource representation, not just changed fields
- Safe to retry on transient errors (5xx, network timeout)

#### Delete
- **Endpoint:** `DELETE /api/v1/protected/admin/pizzas/:id`
- **Auth Required:** Yes (Admin role)
- **Idempotent:** ✅ Yes
- **Ownership:** Only the creator (matched by JWT's `uid` claim) can delete

**Provider Responsibility:**
- Treat 404 as success (already deleted)
- Handle 403 Forbidden (not owner) as an error

### Query Capabilities

The List endpoint supports filtering:

**Filter by Creator:**
```bash
GET /api/v1/public/pizzas?created_by=1
```

**Filter by Name (partial match):**
```bash
GET /api/v1/public/pizzas?name=Margherita
```

**Combined Filters:**
```bash
GET /api/v1/public/pizzas?created_by=1&name=Pizza
```

**Use Case:** Pre-flight check before creating a pizza to avoid duplicates.

---

## Error Handling

### Error Response Format

**Standard Errors:**
```json
{
  "error": "Pizza not found",
  "message": "Additional context (optional)"
}
```

**OAuth Errors (RFC 6749):**
```json
{
  "error": "invalid_client",
  "error_description": "Client authentication failed"
}
```

### Retry Strategy

| HTTP Status | Error Type | Action |
|-------------|------------|--------|
| **400** | Bad Request | Do not retry. Fix request payload. |
| **401** | Unauthorized (invalid_client) | Do not retry. Fix credentials. |
| **401** | Unauthorized (expired_token) | Request new access token, then retry. |
| **403** | Forbidden | Do not retry. Insufficient permissions or not resource owner. |
| **404** | Not Found (on GET) | Resource doesn't exist. Remove from state. |
| **404** | Not Found (on DELETE) | Treat as success (already deleted). |
| **500** | Internal Server Error | Retry with exponential backoff (max 3 attempts). |
| **503** | Service Unavailable | Retry with exponential backoff (max 3 attempts). |

### Exponential Backoff Example

```go
func retryWithBackoff(operation func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        // Check if error is retryable (5xx status)
        if !isRetryable(err) {
            return err
        }
        
        // Exponential backoff: 1s, 2s, 4s, ...
        waitTime := time.Duration(1<<uint(i)) * time.Second
        time.Sleep(waitTime)
    }
    return fmt.Errorf("max retries exceeded")
}
```

### Common Error Scenarios

#### Scenario 1: Token Expired During Operation

**API Response:**
```json
{
  "error": "expired_token",
  "error_description": "Token has expired"
}
```

**Provider Action:**
1. Detect 401 with `expired_token` error
2. Clear cached token
3. Request new access token
4. Retry original operation (once)

#### Scenario 2: Insufficient Permissions

**API Response:**
```json
{
  "error": "insufficient_permissions",
  "message": "Admin role required"
}
```

**Provider Action:**
1. Return clear error to user: "OAuth client does not have admin role. Check credentials."
2. Do not retry
3. Suggest creating a new OAuth client with admin user association

#### Scenario 3: Pizza Not Owned by Client

**API Response:**
```json
{
  "error": "Unauthorized to delete this pizza"
}
```

**Provider Action:**
1. Return error: "Cannot delete pizza: created by a different user"
2. Do not retry
3. This indicates state drift (pizza created outside Terraform)

---

## State Management

### State Storage

Store the following in Terraform state for each pizza resource:

```json
{
  "id": "42",
  "name": "Margherita",
  "description": "Classic Italian pizza",
  "ingredients": ["Tomato Sauce", "Mozzarella", "Basil"],
  "price": 10.99,
  "created_by": 1,
  "created_at": "2025-11-10T12:34:56Z",
  "updated_at": "2025-11-10T12:34:56Z"
}
```

### Handling State Drift

**Detecting Drift:**
- Perform Read operation during `terraform plan`
- Compare API response to stored state
- Flag differences to user

**Common Drift Scenarios:**
1. **Pizza deleted externally:** API returns 404 → Remove from state, plan will recreate
2. **Pizza modified externally:** API returns different values → Show diff, plan will update
3. **Pizza ownership changed:** Not possible via API (creator is immutable)

### Import Support

Allow users to import existing pizzas into Terraform state:

```bash
terraform import pizza_pizza.margherita 42
```

**Implementation:**
1. Parse pizza ID from import identifier
2. Call `GET /api/v1/public/pizzas/42`
3. Populate state with API response
4. Verify user has permission to manage (check `created_by` matches client's user)

---

## Testing Strategy

### Unit Tests

Test individual CRUD functions in isolation:

```go
func TestPizzaCreate(t *testing.T) {
    // Mock HTTP client
    mockClient := &MockHTTPClient{
        Response: `{"id": 1, "name": "Test Pizza", "price": 9.99}`,
        StatusCode: 201,
    }
    
    client := NewPizzaClient(mockClient)
    pizza, err := client.CreatePizza(PizzaInput{
        Name: "Test Pizza",
        Price: 9.99,
    })
    
    assert.NoError(t, err)
    assert.Equal(t, 1, pizza.ID)
}
```

### Integration Tests

Test against a real Pizza API instance:

**Setup:**
1. Deploy Pizza API to local microk8s (see [KUBERNETES.md](./KUBERNETES.md))
2. Create OAuth client for testing: `go run scripts/create_dev_client.go`
3. Export credentials: `export PIZZA_API_CLIENT_ID=...`

**Test Example:**
```go
func TestPizzaLifecycle(t *testing.T) {
    client := NewPizzaClientFromEnv()
    
    // Create
    pizza, err := client.CreatePizza(PizzaInput{
        Name: "Test Pizza",
        Ingredients: []string{"Cheese", "Tomato"},
        Price: 12.99,
    })
    require.NoError(t, err)
    
    // Read
    retrieved, err := client.GetPizza(pizza.ID)
    require.NoError(t, err)
    assert.Equal(t, pizza.Name, retrieved.Name)
    
    // Update
    pizza.Price = 14.99
    updated, err := client.UpdatePizza(pizza.ID, pizza)
    require.NoError(t, err)
    assert.Equal(t, 14.99, updated.Price)
    
    // Delete
    err = client.DeletePizza(pizza.ID)
    require.NoError(t, err)
    
    // Verify deletion
    _, err = client.GetPizza(pizza.ID)
    assert.Error(t, err) // Should be 404
}
```

### Acceptance Tests (Terraform)

Use Terraform acceptance testing framework:

```go
func TestAccPizzaPizza_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: testAccPizzaPizzaConfig("Margherita", 10.99),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("pizza_pizza.test", "name", "Margherita"),
                    resource.TestCheckResourceAttr("pizza_pizza.test", "price", "10.99"),
                    resource.TestCheckResourceAttrSet("pizza_pizza.test", "id"),
                ),
            },
            // Update and Read
            {
                Config: testAccPizzaPizzaConfig("Margherita Deluxe", 12.99),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("pizza_pizza.test", "name", "Margherita Deluxe"),
                    resource.TestCheckResourceAttr("pizza_pizza.test", "price", "12.99"),
                ),
            },
            // Delete (implicit - no config)
        },
    })
}
```

### Manual Testing

Use the provided test script to validate API behavior:

```bash
cd gin-pizza-api
./scripts/test-api.sh
```

This script validates:
- OAuth token acquisition
- CRUD operations
- Creator attribution
- Ownership-based deletion

---

## Example Implementation

### Complete Provider Skeleton

```go
package provider

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"

    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/provider/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource"
)

// PizzaProvider defines the provider implementation
type PizzaProvider struct {
    version string
}

func New(version string) func() provider.Provider {
    return func() provider.Provider {
        return &PizzaProvider{
            version: version,
        }
    }
}

func (p *PizzaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "pizza"
    resp.Version = p.version
}

func (p *PizzaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "endpoint": schema.StringAttribute{
                Description: "Pizza API base URL",
                Required:    true,
            },
            "client_id": schema.StringAttribute{
                Description: "OAuth2 client ID",
                Required:    true,
            },
            "client_secret": schema.StringAttribute{
                Description: "OAuth2 client secret",
                Required:    true,
                Sensitive:   true,
            },
        },
    }
}

func (p *PizzaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config struct {
        Endpoint     string `tfsdk:"endpoint"`
        ClientID     string `tfsdk:"client_id"`
        ClientSecret string `tfsdk:"client_secret"`
    }

    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }

    client := &APIClient{
        endpoint:     config.Endpoint,
        clientID:     config.ClientID,
        clientSecret: config.ClientSecret,
        httpClient:   &http.Client{Timeout: 30 * time.Second},
    }

    resp.ResourceData = client
}

func (p *PizzaProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewPizzaResource,
    }
}

// APIClient handles Pizza API communication
type APIClient struct {
    endpoint     string
    clientID     string
    clientSecret string
    httpClient   *http.Client
    token        string
    tokenExpiry  time.Time
}

func (c *APIClient) ensureToken(ctx context.Context) error {
    if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-5*time.Minute)) {
        return nil
    }

    resp, err := c.httpClient.PostForm(c.endpoint+"/api/v1/oauth/token", url.Values{
        "grant_type":    {"client_credentials"},
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
    })
    if err != nil {
        return fmt.Errorf("token request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("token request failed with status %d", resp.StatusCode)
    }

    var tokenResp struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return fmt.Errorf("failed to decode token response: %w", err)
    }

    c.token = tokenResp.AccessToken
    c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
    return nil
}

// Additional methods: CreatePizza, GetPizza, UpdatePizza, DeletePizza...
```

---

## Additional Resources

### Example Terraform Configuration

```hcl
terraform {
  required_providers {
    pizza = {
      source = "example.com/example/pizza"
      version = "~> 1.0"
    }
  }
}

provider "pizza" {
  endpoint      = "https://pizza-api.local"
  client_id     = var.pizza_client_id
  client_secret = var.pizza_client_secret
}

resource "pizza_pizza" "margherita" {
  name        = "Margherita"
  description = "Classic Italian pizza"
  ingredients = [
    "Tomato Sauce",
    "Mozzarella",
    "Fresh Basil",
    "Olive Oil"
  ]
  price = 10.99
}

resource "pizza_pizza" "pepperoni" {
  name        = "Pepperoni"
  description = "American classic"
  ingredients = [
    "Tomato Sauce",
    "Mozzarella",
    "Pepperoni"
  ]
  price = 12.99
}

output "margherita_id" {
  value = pizza_pizza.margherita.id
}
```

### Setting Up Development Environment

```bash
# 1. Deploy Pizza API to microk8s
cd gin-pizza-api
docker build -t pizza-api:latest .
docker save pizza-api:latest | microk8s ctr image import -
kubectl apply -f k8s/

# 2. Create OAuth client for testing
go run scripts/create_dev_client.go

# 3. Test API manually
export API_ENDPOINT="https://pizza-api.local"
export CLIENT_ID="dev-client"
export CLIENT_SECRET="<from-step-2>"

# 4. Develop provider
cd terraform-provider-pizza
go mod init github.com/example/terraform-provider-pizza
# ... implement provider using this guide
```

---

## Conclusion

This guide provides the foundation for building a Terraform provider for the Pizza API. Key takeaways:

- **Authentication:** Use OAuth2 client credentials with token caching
- **Idempotency:** Handle non-idempotent creates carefully, rely on state for duplicate prevention
- **Error Handling:** Implement retry logic for transient errors, fail fast on client errors
- **State Management:** Store full resource representation, handle drift gracefully
- **Testing:** Use unit, integration, and acceptance tests for comprehensive coverage

For detailed API specifications, see [API_CONTRACT.md](./API_CONTRACT.md).

---

**Document Version:** 1.0.0  
**Last Updated:** November 11, 2025
