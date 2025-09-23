# Findings: Data Seeding and Endpoint Testing after Step 4

## Where you are
You're right after Step 4 (Authorization Code Grant Setup). At this point you have:
- OAuth models (`OAuthClient`, `OAuthCode`, `OAuthToken`) and migrations wired in `cmd/main.go`.
- OAuth server and endpoints registered:
  - `POST /api/v1/oauth/token` for `client_credentials` and `authorization_code`
  - `GET /api/v1/oauth/authorize` to issue authorization codes (expects `userID` in context)
- JWT middleware still protects `/api/v1/protected/**` routes.

## What is missing to test endpoints
To exercise both OAuth flows you need initial data and a way to "be authenticated" for the authorization endpoint:

1. OAuth Clients
   - At least one client for Client Credentials:
     - `id` (e.g., `m2m-client`)
     - `secret` (BCrypt-hashed in DB)
     - `scopes` (e.g., `read write`)
     - `grant_types` contains `client_credentials`
   - At least one client for Authorization Code:
     - `id` (e.g., `web-client`)
     - `secret` (BCrypt-hashed)
     - `redirect_uri` (e.g., `http://localhost:3000/callback`)
     - `scopes` (e.g., `read`)
     - `grant_types` contains `authorization_code`

2. A way to set `userID` in context when calling `/api/v1/oauth/authorize`
   - Current code redirects to `/login` if `userID` is not set.
   - There is no `/login` handler implemented yet (Step 6).

3. Optional: Test users
   - A `users` table isn't present in this repo yet, so you don't need persisted users to test. But you do need to inject a `userID` for the authorize step.

## Minimal viable testing approaches

A) Client Credentials flow (no user required)
- Seed one M2M client with BCrypt secret and `client_credentials` grant.
- Call `POST /api/v1/oauth/token` with form-encoded body:
  - `grant_type=client_credentials&client_id=<id>&client_secret=<plaintext-secret>&scope=read`

B) Authorization Code flow (user required)
- Seed one web client with BCrypt secret, `authorization_code` grant, and a valid `redirect_uri`.
- Temporarily create a lightweight test-only middleware/route to simulate an authenticated user by setting `c.Set("userID", "test-user-123")` before hitting `/api/v1/oauth/authorize`.
  - Example: wrap `/api/v1/oauth/authorize` with a dev-only middleware that injects `userID` when an `X-Test-User` header is present.
- Then:
  1) `GET /api/v1/oauth/authorize?client_id=web-client&redirect_uri=http://localhost:3000/callback&response_type=code&scope=read`
  2) Follow redirect, extract `code`
  3) `POST /api/v1/oauth/token` with `grant_type=authorization_code&client_id=web-client&client_secret=<plaintext-secret>&code=<code>&redirect_uri=http://localhost:3000/callback`

## Concrete seeding plan (simple and safe)

Add a one-time seed in `setupDatabase` after OAuth migrations when tables are empty:
- Insert two `oauth_clients` rows with BCrypt-hashed secrets.
- Only run if table `oauth_clients` is empty.

Suggested values:
- M2M client
  - id: `m2m-client`
  - secret (plain for testing): `m2m-secret` -> store BCrypt hash
  - scopes: `read write`
  - grant_types: `client_credentials`
- Web client
  - id: `web-client`
  - secret (plain for testing): `web-secret` -> store BCrypt hash
  - scopes: `read`
  - grant_types: `authorization_code`
  - redirect_uri: `http://localhost:3000/callback`

## How to hash and insert secrets
- Use `golang.org/x/crypto/bcrypt`:
  - `hash, _ := bcrypt.GenerateFromPassword([]byte("m2m-secret"), bcrypt.DefaultCost)`
  - Save `string(hash)` into `Secret`.

## Dev helper to simulate a logged-in user
Until Step 6 is implemented, add a small middleware just for dev/testing that, if header `X-Test-User` is present, sets `c.Set("userID", <value>)`. Apply it to the `/api/v1/oauth/authorize` route only in development.

Pseudo-code for `cmd/main.go` in `setupRoutes`:

```go
// Dev helper: inject userID for /oauth/authorize when X-Test-User is present
func devUserInjector() gin.HandlerFunc {
    return func(c *gin.Context) {
        if v := c.GetHeader("X-Test-User"); v != "" {
            c.Set("userID", v)
        }
        c.Next()
    }
}

// ... inside setupRoutes
auth := v1.Group("/oauth")
{
    auth.POST("/token", oauthService.HandleToken)
    // only in development
    if config.GetEnvWithDefault("APP_ENV", "development") == "development" {
        auth.GET("/authorize", devUserInjector(), oauthService.HandleAuthorize)
    } else {
        auth.GET("/authorize", oauthService.HandleAuthorize)
    }
}
```

## How to test now

1) Client Credentials (works now)

```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials&client_id=m2m-client&client_secret=m2m-secret&scope=read'
```

2) Authorization Code (with header-based user)

```bash
# Step 1: get authorization code
curl -i -X GET "http://localhost:8080/api/v1/oauth/authorize?client_id=web-client&response_type=code&redirect_uri=http://localhost:3000/callback&scope=read" \
  -H 'X-Test-User: test-user-123'

# Step 2: exchange code for token (replace CODE)
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=authorization_code&client_id=web-client&client_secret=web-secret&code=CODE&redirect_uri=http://localhost:3000/callback'
```

## Notes on JWT middleware
- Protected routes under `/api/v1/protected/**` still expect a JWT with `user` and `role` claims.
- The OAuth access tokens your server issues are JWTs (HS512) but claims mapping to this middleware is not implemented yet (Step 7). So use `/test-token` for protected routes or implement claim mapping later.

## Acceptance checklist
- [ ] `oauth_clients` seeded with two test clients (hashed secrets)
- [ ] Dev-only user injector applied to `/oauth/authorize`
- [ ] Client Credentials flow returns 200 with Bearer token
- [ ] Authorization Code flow returns code, then token
- [ ] Documented curl steps verified locally

## Follow-ups (future steps)
- Implement Step 5 (token exchange refresh, PKCE validation as needed)
- Implement Step 6 (user registration/login) and replace dev injector with real session/auth
- Step 7: Update JWT middleware or add an OAuth validator to accept issued tokens and set `userID`/`role`
