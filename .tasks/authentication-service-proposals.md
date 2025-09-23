# Authentication Service Implementation Proposals

## Overview

Based on the analysis of the `gin-pizza-api` repository, which currently implements basic JWT authentication with role-based authorization, here are three comprehensive proposals for implementing an authentication service that supports both user-level authentication (e.g., end-users interacting with the API) and machine-to-machine authentication (e.g., Terraform provider ↔ API communication).

## Current Repository Analysis

The repository uses:
- Gin framework with JWT middleware
- GORM with SQLite for data persistence
- Basic role-based access control
- Swagger documentation
- Environment-based configuration

## Proposal 1: OAuth 2.0 with Authorization Code (Users) and Client Credentials (M2M)

### High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   End Users     │    │  Terraform       │    │   API Server    │
│   (Browser/SPA) │    │  Provider        │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │ Authorization Code     │ Client Credentials   │
         │ Grant Flow             │ Grant Flow           │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ OAuth 2.0       │    │ API Key/Secret   │    │ JWT Tokens      │
│ Authorization   │    │ Management       │    │ + Refresh       │
│ Server          │    │                  │    │ Tokens          │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Integration Points

#### Gin Integration
- **OAuth 2.0 Server**: Implement OAuth 2.0 endpoints (`/oauth/authorize`, `/oauth/token`)
- **Middleware Chain**: JWT validation → OAuth scope checking → Role-based authorization
- **Token Storage**: Database tables for access tokens, refresh tokens, and client credentials

#### Terraform Provider Integration
```hcl
provider "ginpizza" {
  host         = "https://api.example.com"
  client_id    = var.client_id
  client_secret = var.client_secret
}
```

### Suggested Technologies
- **OAuth 2.0 Library**: `golang.org/x/oauth2` or `github.com/go-oauth2/oauth2`
- **JWT**: `github.com/golang-jwt/jwt/v5` (already in use)
- **Database**: Extend existing GORM models for OAuth entities

### Pros
- Industry standard for both user and M2M authentication
- Well-documented and widely supported
- Flexible grant types for different use cases
- Built-in token refresh mechanisms
- Strong security with PKCE for public clients

### Cons
- More complex implementation than simpler approaches
- Requires additional database tables and endpoints
- Higher maintenance overhead
- May be overkill for simple APIs

### Example Implementation Details

#### Gin Middleware Snippet
```go
func OAuth2Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        
        // Validate OAuth 2.0 access token
        claims, err := validateOAuthToken(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
            c.Abort()
            return
        }
        
        // Set user context
        c.Set("user_id", claims.Subject)
        c.Set("scopes", claims.Scopes)
        c.Next()
    }
}
```

#### Terraform Provider Schema
```go
func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "host": schema.StringAttribute{
                Required: true,
                Description: "API server host URL",
            },
            "client_id": schema.StringAttribute{
                Required: true,
                Description: "OAuth 2.0 client ID",
            },
            "client_secret": schema.StringAttribute{
                Required:    true,
                Sensitive:   true,
                Description: "OAuth 2.0 client secret",
            },
        },
    }
}
```

## Proposal 2: JWT for Users + API Keys for M2M

### High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   End Users     │    │  Terraform       │    │   API Server    │
│   (Username/    │    │  Provider        │    │                 │
│    Password)    │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │ JWT Tokens            │ API Key + HMAC       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ User Database   │    │ API Key Store    │    │ JWT Validation  │
│ + Password Hash │    │ + HMAC Secret    │    │ + API Key Auth  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Integration Points

#### Gin Integration
- **Dual Middleware**: Separate middleware for JWT and API key authentication
- **User Registration/Login**: Extend existing user model with password hashing
- **API Key Management**: Admin endpoints for creating/managing API keys
- **Unified Context**: Both auth methods set similar context variables

#### Terraform Provider Integration
```hcl
provider "ginpizza" {
  host     = "https://api.example.com"
  api_key  = var.api_key
  api_secret = var.api_secret
}
```

### Suggested Technologies
- **Password Hashing**: `golang.org/x/crypto/bcrypt` (already available)
- **HMAC**: `crypto/hmac` with SHA-256
- **JWT**: Existing `github.com/golang-jwt/jwt/v5`
- **API Key Storage**: Extend existing database schema

### Pros
- Builds directly on existing JWT implementation
- Simple and straightforward for both use cases
- Good performance with HMAC validation
- Easy to understand and debug
- Minimal external dependencies

### Cons
- Less standardized than OAuth 2.0
- No built-in token refresh for users
- API keys require manual rotation
- Less flexible for complex authorization scenarios

### Example Implementation Details

#### Gin Middleware Snippet
```go
func APIKeyAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        signature := c.GetHeader("X-API-Signature")
        timestamp := c.GetHeader("X-API-Timestamp")
        
        if !validateHMAC(apiKey, signature, timestamp, c.Request) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
            c.Abort()
            return
        }
        
        c.Set("client_id", getClientFromAPIKey(apiKey))
        c.Set("auth_type", "api_key")
        c.Next()
    }
}
```

#### Terraform Provider Schema
```go
func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "host": schema.StringAttribute{
                Required: true,
            },
            "api_key": schema.StringAttribute{
                Required:  true,
                Sensitive: true,
            },
            "api_secret": schema.StringAttribute{
                Required:  true,
                Sensitive: true,
            },
        },
    }
}
```

## Proposal 3: Enhanced JWT with HMAC for M2M

### High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   End Users     │    │  Terraform       │    │   API Server    │
│   (JWT Tokens)  │    │  Provider        │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │ Standard JWT          │ HMAC-Signed JWT      │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ User Sessions   │    │ Service Accounts │    │ JWT Validation  │
│ + Refresh       │    │ + HMAC Keys      │    │ + HMAC Verify   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Integration Points

#### Gin Integration
- **Unified JWT Middleware**: Single middleware handling both user and service JWTs
- **HMAC for Service Tokens**: Service accounts use HMAC-SHA256 signed JWTs
- **Token Type Detection**: Automatic detection of token type based on claims
- **Service Account Management**: Admin endpoints for managing service accounts

#### Terraform Provider Integration
```hcl
provider "ginpizza" {
  host          = "https://api.example.com"
  service_token = var.service_token  # HMAC-signed JWT
}
```

### Suggested Technologies
- **JWT**: Existing `github.com/golang-jwt/jwt/v5`
- **HMAC**: `crypto/hmac` with SHA-256
- **Service Accounts**: New database model for service accounts with HMAC secrets
- **Token Refresh**: Extend existing JWT implementation

### Pros
- Single authentication mechanism (JWT) for both use cases
- Leverages existing JWT infrastructure
- Strong security with HMAC signatures
- Easy to implement token refresh for users
- Good performance and scalability

### Cons
- Requires careful token type detection
- Service accounts need separate key management
- May confuse developers familiar with standard OAuth flows
- Less standard than dedicated M2M solutions

### Example Implementation Details

#### Gin Middleware Snippet
```go
func UnifiedJWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := extractBearerToken(c)
        
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if claims, ok := token.Claims.(jwt.MapClaims); ok {
                if clientID, exists := claims["client_id"]; exists {
                    // Service account token - use HMAC
                    return getHMACSecret(clientID.(string)), nil
                }
            }
            // User token - use RSA public key
            return getPublicKey(), nil
        })
        
        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            c.Set("user_id", claims["sub"])
            c.Set("client_id", claims["client_id"])
            c.Set("auth_type", getAuthType(claims))
        }
        
        c.Next()
    }
}
```

#### Terraform Provider Schema
```go
func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "host": schema.StringAttribute{
                Required: true,
            },
            "service_token": schema.StringAttribute{
                Required:  true,
                Sensitive: true,
                Description: "HMAC-signed JWT service token",
            },
        },
    }
}
```

## Recommendation

For this project, **Proposal 2 (JWT for Users + API Keys for M2M)** is recommended as the best starting point because:

1. It builds directly on the existing JWT implementation
2. Provides clear separation between user and machine authentication
3. Is simpler to implement and maintain than full OAuth 2.0
4. Meets the requirements effectively
5. Can be extended to OAuth 2.0 later if needed

## Implementation Next Steps

1. Extend the User model with password hashing
2. Create API Key and Service Account models
3. Implement dual authentication middleware
4. Add admin endpoints for API key management
5. Update Terraform provider with authentication logic
6. Add comprehensive tests and documentation

## References

- [HashiCorp Terraform Provider Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [OAuth 2.0 Specification](https://oauth.net/2/)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [API Key Authentication Best Practices](https://cloud.google.com/endpoints/docs/openapi/when-why-api-key)
