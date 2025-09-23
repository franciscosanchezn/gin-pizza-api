# JWT Scope Claim Issue Findings (2025-09-23)

## Summary
- Build error in `internal/middleware/middleware.go` line ~88: `claims.Scope` is undefined on type `generates.JWTAccessClaims`.
- The current version of `github.com/go-oauth2/oauth2/v4` (v4.5.4 as of Aug 20, 2025) defines `JWTAccessClaims` with only `jwt.RegisteredClaims` (no custom fields like `scope`).
- The default JWT access token generator in go-oauth2 sets only standard claims: `aud` (client ID), `sub` (user ID), and `exp`. It does not embed `scope` in the JWT claims by default.

## Evidence
- pkg.go.dev (v4.5.4): `JWTAccessClaims` contains only `jwt.RegisteredClaims`.
  - https://pkg.go.dev/github.com/go-oauth2/oauth2/v4/generates
- Source (v4.5.4): `generates/jwt_access.go`
  - https://github.com/go-oauth2/oauth2/blob/v4.5.4/generates/jwt_access.go
  - Relevant snippet shows:
    ```go
    type JWTAccessClaims struct {
        jwt.RegisteredClaims
    }
    // Token(...): sets Audience (client ID), Subject (user ID), ExpiresAt
    ```

## Root Cause
- Our middleware assumes a `Scope` field exists on `JWTAccessClaims` (`setOAuth2Context` uses `claims.Scope`), which is not part of the library's struct. This mismatch leads to a compile-time error.

## Impact
- Middleware fails to build.
- Even after fixing compilation, scopes will not be available from JWT claims unless we explicitly include them (via custom generator) or fetch scopes from another source.

## Remediation Options

1) Minimal change: Stop assuming `claims.Scope`
- Parse JWT as `jwt.MapClaims` and read standard claims (`sub`, `aud`). Attempt to read `scope` only if present.
- Pros: Quick, no changes to token issuance. Cons: `scope` often absent with default generator.

Example changes:
```go
func parseOAuth2Token(tokenString string, jwtSecret []byte) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, jwt.ErrSignatureInvalid
        }
        return jwtSecret, nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims, nil
    }
    return nil, jwt.ErrInvalidKey
}

func setOAuth2Context(c *gin.Context, claims jwt.MapClaims) {
    // subject -> userID
    if sub, ok := claims["sub"].(string); ok { c.Set("userID", sub) }

    // audience -> clientID (string or array)
    switch aud := claims["aud"].(type) {
    case string:
        c.Set("clientID", aud)
    case []interface{}:
        if len(aud) > 0 {
            if s, ok := aud[0].(string); ok { c.Set("clientID", s) }
        }
    }

    // optional scope if present
    if sc, ok := claims["scope"].(string); ok { c.Set("scopes", sc) }
    c.Set("auth_type", "oauth2")
}
```

2) Preferred: Embed `scope` into JWT by using a custom generator
- Implement a custom `JWTAccessGenerate` that adds a `Scope` (or `scp`) field into claims from `data.TokenInfo.GetScope()`.
- Register with manager: `manager.MapAccessGenerate(NewCustomJWTAccessGenerate(...))`.
- Pros: Scopes are self-contained in tokens. Cons: Small implementation overhead.

Example skeleton:
```go
// internal/auth/custom_jwt_generate.go
package auth

import (
    "context"
    goauth "github.com/go-oauth2/oauth2/v4"
    "github.com/golang-jwt/jwt/v5"
    "github.com/go-oauth2/oauth2/v4/errors"
    "strings"
)

type ScopeClaims struct {
    jwt.RegisteredClaims
    Scope string `json:"scope,omitempty"`
}

type CustomJWTAccessGenerate struct {
    SignedKeyID  string
    SignedKey    []byte
    SignedMethod jwt.SigningMethod
}

func NewCustomJWTAccessGenerate(kid string, key []byte, method jwt.SigningMethod) *CustomJWTAccessGenerate {
    return &CustomJWTAccessGenerate{kid, key, method}
}

func (a *CustomJWTAccessGenerate) Token(ctx context.Context, data *goauth.GenerateBasic, isGenRefresh bool) (string, string, error) {
    claims := &ScopeClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Audience:  jwt.ClaimStrings{data.Client.GetID()},
            Subject:   data.UserID,
            ExpiresAt: jwt.NewNumericDate(data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn())),
        },
        Scope: data.TokenInfo.GetScope(),
    }
    token := jwt.NewWithClaims(a.SignedMethod, claims)
    // sign key handling (HS/RS/ES/Ed) similar to upstream...
    // return access, refresh, nil
    return "", "", errors.New("implement signing like upstream")
}
```
Register in `NewOAuthService`:
```go
manager.MapAccessGenerate(NewCustomJWTAccessGenerate("", []byte(jwtSecret), jwt.SigningMethodHS512))
```
Then update middleware to parse `jwt.MapClaims` or a matching struct and read `scope`.

3) Lookup scopes server-side via DB/token store
- After validating JWT, query `oauth_tokens` by `access_token` to retrieve `Scopes`.
- Pros: No change to token format. Cons: Requires DB access in middleware and an extra query per request.

## Recommendation
- Short-term: remove the direct reference to `claims.Scope`; parse as `jwt.MapClaims` and read `sub`/`aud`, set scopes only if present.
- Mid-term: implement a custom generator to embed `scope` into tokens; update middleware accordingly.
- Long-term: migrate protected route checks to rely on a dedicated OAuth validation middleware that understands our token format, including scopes and roles.

## Action Items
- [ ] Fix middleware to compile by dropping `claims.Scope` (or switching to `MapClaims` parsing with optional `scope`).
- [ ] Decide whether to embed `scope` into JWTs (custom generator) or fetch from DB.
- [ ] If embedding scopes, implement and wire `CustomJWTAccessGenerate`, update middleware to read `scope` claim.
- [ ] Add tests to cover middleware behavior with and without `scope` claim present.
