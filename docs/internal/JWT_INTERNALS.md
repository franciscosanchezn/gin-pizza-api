# JWT Internals

This document provides a deep dive into the JWT (JSON Web Token) implementation, OAuth2 service account model, and authentication architecture of the Pizza API.

---

## Table of Contents

- [JWT Token Structure](#jwt-token-structure)
- [Token Claims Explained](#token-claims-explained)
- [OAuth Client Service Account Model](#oauth-client-service-account-model)
- [Creator Attribution Flow](#creator-attribution-flow)
- [Token Generation Process](#token-generation-process)
- [Token Validation Process](#token-validation-process)
- [Security Considerations](#security-considerations)

---

## JWT Token Structure

The API issues **JWT (JSON Web Tokens)** for authentication. JWTs are stateless, self-contained tokens that encode user identity and permissions.

### Token Format

A JWT consists of three parts separated by dots (`.`):

```
header.payload.signature
```

**Example token:**
```
eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxIiwicm9sZSI6ImFkbWluIiwiYXVkIjoiZGV2LWNsaWVudCIsInNjb3BlIjoicmVhZCB3cml0ZSIsImV4cCI6MTY5OTYzMjAwMCwiaWF0IjoxNjk5NjI4NDAwfQ.signature_here
```

### Header

```json
{
  "alg": "HS512",
  "typ": "JWT"
}
```

- **`alg`**: Signing algorithm (HS512 - HMAC with SHA-512)
- **`typ`**: Token type (JWT)

### Payload

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

### Signature

The signature is generated using:
```
HMACSHA512(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  JWT_SECRET
)
```

This ensures the token hasn't been tampered with.

---

## Token Claims Explained

### Standard JWT Claims

#### `uid` (User ID) - **Custom Claim**

- **Type:** String
- **Description:** The ID of the User record associated with the OAuth client
- **Purpose:** Creator attribution for resources (e.g., `Pizza.CreatedBy`)
- **Example:** `"1"`

**Why a string?** GORM uses `uint` for primary keys, but JWT standard recommends strings for flexibility.

#### `role` (User Role) - **Custom Claim**

- **Type:** String
- **Description:** The role of the associated user
- **Values:** `"admin"` or `"user"`
- **Purpose:** Authorization (determines access to protected endpoints)
- **Example:** `"admin"`

**Role-based permissions:**
- `admin`: Full CRUD access to all pizzas
- `user`: Read-only access (future feature)

#### `aud` (Audience) - **Standard Claim**

- **Type:** String
- **Description:** The OAuth client ID that requested the token
- **Purpose:** Token validation (ensures token is used by intended client)
- **Example:** `"dev-client"`

#### `scope` (Token Scopes) - **Custom Claim**

- **Type:** String (space-separated)
- **Description:** OAuth2 scopes granted to the token
- **Values:** `"read write"`
- **Purpose:** Future fine-grained permissions
- **Example:** `"read write"`

**Currently:** All tokens receive both `read` and `write` scopes. Future versions may restrict scopes per client.

#### `exp` (Expiration Time) - **Standard Claim**

- **Type:** Integer (Unix timestamp)
- **Description:** When the token expires
- **Default:** 3600 seconds (1 hour) from issuance
- **Example:** `1699632000`

**Validation:** Tokens are rejected if `exp < current_time`.

#### `iat` (Issued At) - **Standard Claim**

- **Type:** Integer (Unix timestamp)
- **Description:** When the token was issued
- **Purpose:** Audit logging, token age calculation
- **Example:** `1699628400`

---

## OAuth Client Service Account Model

OAuth clients in this API are **service accounts** rather than end-user authentication mechanisms. This model is designed for **machine-to-machine communication** (e.g., Terraform providers, CI/CD pipelines).

### Entity Relationships

```
┌─────────────────┐
│ OAuth Client    │
│ - ClientID      │
│ - ClientSecret  │
│ - UserID (FK)   │──┐
└─────────────────┘  │
                     │
                     │ References
                     │
                     ▼
              ┌─────────────┐
              │ User        │
              │ - ID (PK)   │
              │ - Username  │
              │ - Role      │
              └─────────────┘
                     │
                     │ Owns
                     │
                     ▼
              ┌─────────────┐
              │ Pizza       │
              │ - ID (PK)   │
              │ - CreatedBy │
              └─────────────┘
```

### Why This Model?

#### 1. Stable Identity for Resource Ownership

**Without User association:**
- OAuth clients can be deleted/recreated
- Resource ownership would be orphaned
- No stable identity for audit trails

**With User association:**
- User provides persistent identity
- Resources remain owned even if client is rotated
- Audit trails remain intact

#### 2. Role-Based Access Control (RBAC)

**Leverages User roles:**
- `admin`: Full CRUD permissions
- `user`: Read-only (future)

**Avoids OAuth client role duplication:**
- Don't need separate role field on `OAuthClient`
- Don't need separate permission system

#### 3. Audit Trail and Attribution

**Creator attribution flow:**
```
JWT Token → uid claim → User.ID → Pizza.CreatedBy
```

**Benefits:**
- Know which service account created each resource
- Support ownership-based authorization
- Comply with audit requirements

#### 4. Flexibility

**Multiple clients, same user:**
```
terraform-client-1 → terraform-user (admin)
terraform-client-2 → terraform-user (admin)
```

**Different users, different permissions:**
```
ci-client → ci-user (user role)
admin-client → admin-user (admin role)
```

---

## Creator Attribution Flow

### How Resources Are Attributed

When a client creates a pizza, the `uid` claim from their JWT is used to populate `Pizza.CreatedBy`.

**Step-by-step:**

1. **Client authenticates:**
   ```bash
   POST /api/v1/oauth/token
   Client ID: terraform-client
   Client Secret: secret123
   ```

2. **Server generates JWT:**
   - Looks up OAuth client → finds `UserID = 1`
   - Generates token with `uid = "1"`

3. **Client creates pizza:**
   ```bash
   POST /api/v1/pizzas
   Authorization: Bearer <token>
   Body: {"name": "Margherita", "price": 10.99}
   ```

4. **Middleware extracts `uid`:**
   - Validates token
   - Extracts `uid = "1"` from claims
   - Stores in Gin context: `ctx.Set("uid", "1")`

5. **Controller creates pizza:**
   ```go
   uid := ctx.GetString("uid")
   pizza.CreatedBy = convertToUint(uid) // CreatedBy = 1
   db.Create(&pizza)
   ```

6. **Pizza record stored:**
   ```json
   {
     "id": 42,
     "name": "Margherita",
     "price": 10.99,
     "created_by": 1,
     "created_at": "2025-11-11T10:30:00Z"
   }
   ```

### Ownership Verification

**Future feature:** Only the creator (or admins) can delete a pizza.

```go
func (c *controller) DeletePizza(ctx *gin.Context, id int) {
    uid := ctx.GetString("uid")
    role := ctx.GetString("role")
    
    pizza := getPizzaByID(id)
    
    // Check ownership
    if pizza.CreatedBy != convertToUint(uid) && role != "admin" {
        ctx.JSON(403, gin.H{"error": "insufficient_permissions"})
        return
    }
    
    db.Delete(&pizza)
}
```

---

## Token Generation Process

### OAuth2 Client Credentials Flow

**Endpoint:** `POST /api/v1/oauth/token`

**Request:**
```http
POST /api/v1/oauth/token HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Authorization: Basic <base64(client_id:client_secret)>

grant_type=client_credentials
```

**Server-side processing:**

1. **Parse credentials:**
   - Extract `client_id` and `client_secret` from Basic Auth header
   - Or from request body (`client_id` and `client_secret` fields)

2. **Validate client:**
   ```go
   client := db.FindOAuthClient(clientID)
   if !bcrypt.Compare(clientSecret, client.Secret) {
       return error("invalid_client")
   }
   ```

3. **Lookup associated user:**
   ```go
   user := db.FindUser(client.UserID)
   ```

4. **Generate JWT:**
   ```go
   token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
       "uid":   strconv.Itoa(user.ID),
       "role":  user.Role,
       "aud":   client.ClientID,
       "scope": "read write",
       "exp":   time.Now().Add(1 * time.Hour).Unix(),
       "iat":   time.Now().Unix(),
   })
   
   signedToken, _ := token.SignedString([]byte(JWT_SECRET))
   ```

5. **Return token:**
   ```json
   {
     "access_token": "eyJhbGc...",
     "token_type": "Bearer",
     "expires_in": 3600,
     "scope": "read write"
   }
   ```

---

## Token Validation Process

### Middleware Flow

**Every protected endpoint:**
```
Request → Middleware → Controller
          ↓
          [Validate JWT]
```

**Validation steps:**

1. **Extract token:**
   ```go
   authHeader := ctx.GetHeader("Authorization")
   tokenString := strings.TrimPrefix(authHeader, "Bearer ")
   ```

2. **Parse and verify signature:**
   ```go
   token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
       return []byte(JWT_SECRET), nil
   })
   
   if err != nil || !token.Valid {
       return error("invalid_token")
   }
   ```

3. **Check expiration:**
   ```go
   claims := token.Claims.(jwt.MapClaims)
   exp := claims["exp"].(float64)
   
   if time.Now().Unix() > int64(exp) {
       return error("token_expired")
   }
   ```

4. **Extract claims:**
   ```go
   uid := claims["uid"].(string)
   role := claims["role"].(string)
   
   ctx.Set("uid", uid)
   ctx.Set("role", role)
   ```

5. **Role-based authorization:**
   ```go
   if role != "admin" {
       return error("insufficient_permissions")
   }
   ```

6. **Pass to controller:**
   ```go
   ctx.Next() // Continue to controller
   ```

---

## Security Considerations

### Token Security

**Strong JWT secrets:**
```bash
# Generate secure secret (minimum 32 bytes)
openssl rand -base64 32
```

**Environment variable:**
```env
JWT_SECRET=bXlTdXBlclNlY3JldEpXVFNlY3JldEtleUhlcmU=
```

**Never hardcode secrets in code.**

### Token Lifetime

**Default:** 1 hour (3600 seconds)

**Trade-offs:**
- **Shorter lifetime**: More secure (less time for stolen token to be used)
- **Longer lifetime**: Better UX (fewer re-authentications)

**For machine-to-machine communication** (Terraform providers):
- 1 hour is reasonable
- Providers should cache tokens and refresh before expiry

### Token Rotation

**Best practice:** Refresh tokens before expiration.

**Terraform provider implementation:**
```go
if time.Until(tokenExpiry) < 5*time.Minute {
    token = refreshToken()
}
```

### HTTPS Requirement

**Always use HTTPS in production.**

- HTTP transmits tokens in plaintext
- Tokens can be intercepted (man-in-the-middle attacks)
- HTTPS encrypts the entire request/response

### Secret Storage

**OAuth client secrets:**
- Stored as bcrypt hashes (not plaintext)
- Use cost factor 10+ for production

**JWT signing secret:**
- Store in environment variables or secret management system
- Rotate periodically (requires invalidating all existing tokens)

### Token Revocation

**Current limitation:** Stateless JWT cannot be revoked.

**Workarounds:**
1. **Short expiration times** (current approach)
2. **Token blacklist** (requires server-side storage)
3. **Rotate JWT_SECRET** (invalidates all tokens)

**Future improvement:** Implement refresh token + short-lived access tokens.

---

## Token Characteristics Summary

| Property | Value | Rationale |
|----------|-------|-----------|
| **Type** | JWT | Stateless, no server storage |
| **Signing Algorithm** | HS512 | Secure HMAC with SHA-512 |
| **Expiration** | 1 hour | Balance security and UX |
| **Scopes** | `read write` | Future fine-grained permissions |
| **Revocation** | Not supported | Stateless by design |
| **Storage** | Client-side | Bearer token in Authorization header |

---

## Additional Resources

- [Development Guide](DEVELOPMENT.md) - Project structure, coding standards
- [Operations Guide](OPERATIONS.md) - Deployment and troubleshooting
- [Contributing Guide](CONTRIBUTING.md) - Contribution process
- [API Contract](../API_CONTRACT.md) - API specifications
- [OAuth 2.0 RFC](https://datatracker.ietf.org/doc/html/rfc6749) - OAuth2 specification
- [JWT RFC](https://datatracker.ietf.org/doc/html/rfc7519) - JWT specification
