# Authentication Service Implementation Guide

## Table of Contents
1. [Understanding the Current Implementation](#understanding-the-current-implementation)
2. [Authentication vs. Authorization](#authentication-vs-authorization)
3. [Authentication Fundamentals](#authentication-fundamentals)
4. [JWT Deep Dive](#jwt-deep-dive)
5. [OAuth 2.0 Grant Types Overview](#oauth-20-grant-types-overview)
6. [Implementation Guide](#implementation-guide)
7. [Security Best Practices](#security-best-practices)
8. [Terraform Provider Integration](#terraform-provider-integration)
9. [References](#references)

## Understanding the Current Implementation

The current implementation in our `gin-pizza-api` uses:
- JWT-based authentication with a test endpoint (`/test-token`) for generating tokens.
- A basic JWT middleware for validating tokens (`internal/middleware/middleware.go`).
- A role-based access control (RBAC) middleware for authorization (`internal/middleware/role.go`).

This is a great starting point. We will build upon this foundation to create a production-ready authentication and authorization system.

## Authentication vs. Authorization

It's crucial to understand the difference between these two concepts:

-   **Authentication (AuthN)** is the process of verifying a user's identity. It answers the question, "Who are you?". This is typically done with a username/password, biometrics, or a token.
-   **Authorization (AuthZ)** is the process of verifying what a specific user has permission to do. It answers the question, "What are you allowed to do?". This happens *after* successful authentication.

In our project, JWTs are used for authentication, and the `RequireRole` middleware is for authorization.

## Authentication Fundamentals

### 1. User Authentication Flow
1.  **Registration**: A new user provides their credentials (e.g., email and password). The application hashes the password using a strong algorithm and stores the user's record in the database.
2.  **Login**: The user provides their credentials again. The server verifies the password against the stored hash. If they match, the server issues a set of JWTs (an access token and a refresh token).
3.  **Authenticated Requests**: For subsequent requests to protected endpoints, the client sends the access token in the `Authorization` header. The API gateway or a middleware validates the token.
4.  **Token Refresh**: When the access token expires, the client uses the refresh token to request a new pair of tokens without requiring the user to log in again.

### 2. Password Security
Storing passwords securely is non-negotiable.

-   **Hashing**: Never store passwords in plain text. Always use a strong, one-way hashing algorithm. **bcrypt** is an excellent choice because it's slow by design (making brute-force attacks difficult) and automatically handles salt generation.
-   **Salting**: A salt is a random string added to the password before hashing. It ensures that two identical passwords will have different hashes, preventing rainbow table attacks. `bcrypt` handles this for you.
-   **Peppering**: A pepper is a secret key added to the password along with the salt. It's stored separately from the database (e.g., in an environment variable) and adds another layer of security.

```go
// Example of adding a pepper
import "golang.org/x/crypto/bcrypt"

var pepper = "my-super-secret-pepper" // Load this from config/env

func hashPassword(password string) (string, error) {
    // bcrypt handles the salt internally
    hashed, err := bcrypt.GenerateFromPassword([]byte(password+pepper), bcrypt.DefaultCost)
    return string(hashed), err
}

func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+pepper))
    return err == nil
}
```

## JWT Deep Dive

### 1. JWT Structure (`header.payload.signature`)
-   **Header**: Contains metadata like the signing algorithm (`alg`) and token type (`typ`).
-   **Payload**: Contains the "claims" – statements about the user and token.
    -   **Registered Claims**: Standard claims like `iss` (issuer), `sub` (subject, usually user ID), `aud` (audience), `exp` (expiration time), `iat` (issued at).
    -   **Custom Claims**: You can add your own data, like `role` or `username`.
-   **Signature**: A cryptographic signature to verify the token's integrity.

### 2. Signing Algorithms: HS256 vs. RS256
-   **HS256 (Symmetric)**: Uses a single secret key to both sign and verify tokens. It's faster but means any service that can verify a token can also create one. This is what's currently used in the project.
-   **RS256 (Asymmetric)**: Uses a private key to sign tokens and a public key to verify them. This is more secure in distributed systems because you can share the public key widely for verification without exposing the ability to create new tokens.

For a single monolithic API, HS256 is often sufficient. For microservices or when third parties need to validate tokens, RS256 is the better choice.

### 3. Token Invalidation
By design, JWTs are stateless. Once issued, they are valid until they expire. This can be a problem if a user's permissions change or a token is compromised. Here are common strategies to handle this:

-   **Short-Lived Access Tokens**: Keep access tokens very short-lived (e.g., 5-15 minutes). This minimizes the window of opportunity for misuse. This is the most common and simplest approach.
-   **Blocklisting (Deny List)**: Maintain a list of invalidated tokens (e.g., in a Redis cache). Before validating a token, check if it's on the blocklist. This adds state back into a stateless system but provides immediate revocation.
-   **Refresh Token Revocation**: When a user logs out, invalidate their refresh token in the database. This prevents them from generating new access tokens.

## OAuth 2.0 Grant Types Overview

OAuth 2.0 is a framework for authorization. Understanding its "grant types" (flows) provides context for how different applications handle authentication.

-   **Authorization Code Grant**: The most common flow for web and mobile apps. It redirects the user to an authorization server to log in and then returns an authorization code, which is exchanged for tokens. It's very secure because tokens are not exposed in the browser.
-   **Resource Owner Password Credentials Grant**: The user provides their password directly to the application, which exchanges it for an access token. **This is the flow you are building**. It should only be used for trusted, first-party applications.
-   **Client Credentials Grant**: Used for machine-to-machine (M2M) communication, where there is no user. The application authenticates itself with a client ID and secret to get a token. **This is ideal for your future Terraform provider.**
-   **Implicit Grant**: A simplified flow where the access token is returned directly to the client. It's less secure and generally discouraged in favor of the Authorization Code Grant with PKCE.

## Implementation Guide

### 1. User Management System

A more robust `User` model:
```go
import "gorm.io/gorm"

type User struct {
    gorm.Model // Includes ID, CreatedAt, UpdatedAt, DeletedAt
    Email        string    `json:"email" gorm:"uniqueIndex;not null"`
    Password     string    `json:"-"`
    Role         string    `json:"role" gorm:"default:'user'"`
    IsVerified   bool      `json:"is_verified" gorm:"default:false"`
    LastLoginAt  time.Time `json:"last_login_at"`
}
```

### 2. JWT Token Generation
A more detailed claims struct:
```go
import "github.com/golang-jwt/jwt/v5"

type JWTClaims struct {
    UserID uint   `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// In your token generation function:
claims := JWTClaims{
    UserID: user.ID,
    Role:   user.Role,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), // Access Token
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        NotBefore: jwt.NewNumericDate(time.Now()),
        Issuer:    "gin-pizza-api",
        Subject:   fmt.Sprintf("%d", user.ID),
        Audience:  []string{"pizza-consumers"},
    },
}
```

### 3. Refresh Token Mechanism
-   **Refresh Token Rotation**: For enhanced security, issue a new refresh token every time it's used. If a refresh token is ever compromised and used, the legitimate user's subsequent attempt will fail, signaling a potential breach.
-   **Storage**: Store a hash of the refresh token in the database, associated with the user, to track its validity.

### 4. Secure Password Reset Flow
1.  User requests a password reset for their email.
2.  Generate a secure, random, single-use token with a short expiration (e.g., 15-30 minutes).
3.  Store a hash of this token in the database with its expiration time.
4.  Send the token to the user's email in a link (e.g., `https://example.com/reset-password?token=...`).
5.  When the user clicks the link, they are prompted to enter a new password.
6.  The application verifies the token from the URL, checks that it's not expired, and allows the password update.
7.  After a successful update, invalidate the reset token immediately.

### 5. Audit Logging
Create a middleware to log important events.
```go
func AuditLogMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        startTime := time.Now()
        c.Next() // Process the request

        userID, _ := c.Get("userID") // Set by the auth middleware

        log.WithFields(log.Fields{
            "user_id":     userID,
            "method":      c.Request.Method,
            "path":        c.Request.URL.Path,
            "status_code": c.Writer.Status(),
            "latency_ms":  time.Since(startTime).Milliseconds(),
            "ip_address":  c.ClientIP(),
        }).Info("API Request")
    }
}
```

## Security Best Practices

1.  **Use HTTPS Everywhere**: Encrypt all traffic between the client and the server to prevent man-in-the-middle attacks.
2.  **CORS (Cross-Origin Resource Sharing)**: If your API will be called from a web browser on a different domain, you need to configure CORS. Be restrictive.
    ```go
    // In Gin, use the CORS middleware
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"https://your-frontend.com"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge: 12 * time.Hour,
    }))
    ```
3.  **Security Headers**: Add security headers to your responses to protect against common attacks like XSS and clickjacking.
    -   `Strict-Transport-Security`: Enforces HTTPS.
    -   `X-Frame-Options`: Prevents clickjacking.
    -   `X-Content-Type-Options`: Prevents MIME-sniffing.
    -   `Content-Security-Policy`: Prevents XSS.
4.  **Input Validation**: Always validate and sanitize all user input to prevent injection attacks (SQLi, XSS, etc.).
5.  **Client-Side Token Storage**:
    -   **localStorage**: Accessible via JavaScript, making it vulnerable to XSS attacks.
    -   **HttpOnly Cookies**: Not accessible via JavaScript, providing better protection against XSS. This is generally the recommended approach for web applications.

## Terraform Provider Integration

### Authentication for a Terraform Provider
For machine-to-machine communication like a Terraform provider, the **Client Credentials Grant** is the ideal OAuth 2.0 flow.

1.  **Create a "Service Account" or "Machine User"**: In your user system, create a special type of user that represents the Terraform provider. This user won't have a password but will have a `client_id` and a `client_secret`.
2.  **Token Endpoint**: Create a new endpoint (e.g., `/oauth/token`) that accepts a `POST` request with `grant_type=client_credentials`, `client_id`, and `client_secret`.
3.  **Provider Configuration**: The Terraform provider will be configured with the client ID and secret. It will call the token endpoint to get an access token and then use that token for all subsequent API calls.

```hcl
# terraform-provider-ginpizza configuration
provider "ginpizza" {
  host          = "http://localhost:8080"
  client_id     = var.pizza_api_client_id
  client_secret = var.pizza_api_client_secret
}
```

This is more secure and standard than using a long-lived static API token for the provider.

### API Stability and Versioning
-   **API Versioning**: Your API paths should be versioned (e.g., `/api/v1/pizzas`). This allows you to introduce breaking changes in a new version (`/api/v2`) without breaking existing clients like the Terraform provider.
-   **Consistent Error Responses**: Standardize your error responses so the provider can parse them reliably.

```json
// Standard error response
{
    "error": {
        "code": "INVALID_INPUT",
        "message": "The request body is malformed.",
        "details": "Field 'price' must be a positive number."
    }
}
```

## References

1.  [JSON Web Tokens (JWT)](https://jwt.io/introduction/)
2.  [OAuth 2.0 Specification](https://oauth.net/2/)
3.  [Go Bcrypt Package](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
4.  [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
5.  [Terraform: Writing Custom Providers](https://developer.hashicorp.com/terraform/plugin/framework)
6.  [Auth0: Which OAuth 2.0 flow should I use?](https://auth0.com/docs/get-started/authentication-and-authorization-flow/which-oauth-2-0-flow-should-i-use)
7.  [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
