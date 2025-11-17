# API Contract Documentation

## Overview

This document defines the formal contract for the Pizza API, version 1.0. It is intended for developers building integrations (such as Terraform providers) who need predictable, well-defined API behavior.

**Target Audience:**
- Terraform provider developers
- API integration developers
- System architects evaluating the API

**Document Version:** 1.0.0  
**API Version:** 1.0.0  
**Last Updated:** November 11, 2025

---

## API Versioning

### Current Version

- **Version:** v1 (1.0.0)
- **Base Path:** `/api/v1/`
- **Status:** Stable

### Versioning Strategy

The API uses **URL-based versioning** to ensure backward compatibility:

```
https://pizza-api.local/api/v1/pizzas
                           ^^^^
                        version prefix
```

### Stability Guarantees

**For v1 APIs:**
- **✅ Guaranteed:** No breaking changes to existing endpoints
- **✅ Guaranteed:** New optional fields may be added to responses
- **✅ Guaranteed:** New query parameters may be added (optional)
- **✅ Guaranteed:** New endpoints may be added
- **❌ Not guaranteed:** Response field order
- **❌ Not guaranteed:** Error message text (error codes are stable)

**Breaking Changes:**
If a breaking change is necessary, a new version (v2) will be released with:
- Minimum 6-month deprecation notice for v1
- Migration guide provided
- Both versions running concurrently during transition period

### Deprecation Policy

Deprecated endpoints/fields will be marked with:
- HTTP `Warning` header in responses
- Documentation updates
- Changelog entry

**Current Deprecations:** None

---

## Authentication

### OAuth2 Client Credentials Flow

The API uses **OAuth2 Client Credentials** grant type (RFC 6749, Section 4.4) for machine-to-machine authentication.

**Token Endpoint:** `POST /api/v1/oauth/token`

**Request Format:**
```http
POST /api/v1/oauth/token HTTP/1.1
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET
```

**Successful Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```

**Error Response (400/401):**
```json
{
  "error": "invalid_client",
  "error_description": "Client authentication failed"
}
```

### Token Usage

Include the access token in the `Authorization` header using the Bearer scheme:

```http
Authorization: Bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...
```

### Token Lifecycle

| Property | Value | Notes |
|----------|-------|-------|
| **Type** | JWT (JSON Web Token) | Stateless, self-contained |
| **Algorithm** | HS512 | HMAC with SHA-512 |
| **Expiration** | 3600 seconds (1 hour) | Configurable via `JWT_EXPIRATION` env var |
| **Refresh** | Not supported | Request new token after expiration |
| **Revocation** | Not supported | Tokens expire naturally |

### JWT Claims

```json
{
  "uid": "1",           // User ID for creator attribution
  "role": "admin",      // User role (admin/user)
  "aud": "client-id",   // OAuth client ID
  "scope": "read write",// Token scopes
  "exp": 1699632000,    // Expiration (Unix timestamp)
  "iat": 1699628400     // Issued at (Unix timestamp)
}
```

### Authentication Error Codes

| Error Code | HTTP Status | Description | Retry Strategy |
|------------|-------------|-------------|----------------|
| `invalid_client` | 401 | Client ID or secret incorrect | Do not retry (fix credentials) |
| `unsupported_grant_type` | 400 | Grant type is not `client_credentials` | Do not retry (fix request) |
| `invalid_token` | 401 | Token malformed or signature invalid | Obtain new token |
| `expired_token` | 401 | Token has expired | Obtain new token |
| `insufficient_permissions` | 403 | User role lacks required permissions | Do not retry (requires admin role) |

---

## Role-Based Authorization

### User Roles

The API supports two roles with different permission levels:

| Role | Description | Capabilities |
|------|-------------|--------------|
| **USER** | Regular user | Full CRUD on own pizzas, read all pizzas |
| **ADMIN** | Administrator | Full CRUD on all pizzas, OAuth client management |

### Pizza Operations Authorization

#### USER Role

- **Scope:** Full CRUD on own pizzas only
- **CREATE:** Pizza ownership automatically set to USER's `userID`
- **READ:** Can view all pizzas (public endpoints)
- **UPDATE:** Can only update pizzas where `created_by == userID`
- **DELETE:** Can only delete pizzas where `created_by == userID`
- **Forbidden:** Modifying other users' pizzas returns `403 Forbidden`

**Example:**
```json
// User attempts to update admin's pizza
PUT /api/v1/pizzas/1
Authorization: Bearer <user_token>

// Response: 403 Forbidden
{
  "error": "You can only update your own pizzas",
  "pizza_owner": 1,
  "your_id": 5
}
```

#### ADMIN Role

- **Scope:** Full CRUD on all pizzas
- **Unrestricted:** Can update/delete any pizza regardless of `created_by`
- **Client Management:** Can create/list/delete OAuth clients (USER cannot)

### Authorization Flow

1. JWT extracted from `Authorization: Bearer <token>` header
2. Middleware validates token signature and expiration
3. Claims (`userID`, `userRole`) set in request context
4. Controller checks ownership: `created_by == userID || userRole == "admin"`
5. Returns `403 Forbidden` with ownership details if unauthorized

### Permission Matrix

| Operation | Endpoint | USER | ADMIN |
|-----------|----------|------|-------|
| List all pizzas | `GET /api/v1/public/pizzas` | ✅ | ✅ |
| Get pizza by ID | `GET /api/v1/public/pizzas/:id` | ✅ | ✅ |
| Create pizza | `POST /api/v1/pizzas` | ✅ Own | ✅ Any |
| Update own pizza | `PUT /api/v1/pizzas/:id` | ✅ Own | ✅ Any |
| Update other's pizza | `PUT /api/v1/pizzas/:id` | ❌ 403 | ✅ Any |
| Delete own pizza | `DELETE /api/v1/pizzas/:id` | ✅ Own | ✅ Any |
| Delete other's pizza | `DELETE /api/v1/pizzas/:id` | ❌ 403 | ✅ Any |
| Create OAuth client | `POST /api/v1/clients` | ❌ 403 | ✅ |
| List OAuth clients | `GET /api/v1/clients` | ❌ 403 | ✅ |
| Delete OAuth client | `DELETE /api/v1/clients/:id` | ❌ 403 | ✅ |

---

## Idempotency Guarantees

### Per-Endpoint Idempotency

| Endpoint | Method | Idempotent? | Details |
|----------|--------|-------------|---------|
| **Create Pizza** | `POST /api/v1/pizzas` | ❌ **No** | Multiple identical requests create multiple pizzas. Provider must track state to avoid duplicates. |
| **Get Pizza** | `GET /api/v1/public/pizzas/:id` | ✅ **Yes** | Naturally idempotent. Safe to retry. |
| **List Pizzas** | `GET /api/v1/public/pizzas` | ✅ **Yes** | Returns current state. Safe to retry. |
| **Update Pizza** | `PUT /api/v1/pizzas/:id` | ✅ **Yes** | Applying same update multiple times is idempotent. Safe to retry. |
| **Delete Pizza** | `DELETE /api/v1/pizzas/:id` | ✅ **Yes** | First delete succeeds (200), subsequent returns 404. Safe to retry. |
| **Create OAuth Client** | `POST /api/v1/clients` | ❌ **No** | Multiple requests create multiple clients. No duplicate detection. |
| **Delete OAuth Client** | `DELETE /api/v1/clients/:id` | ✅ **Yes** | First delete succeeds, subsequent returns 404. Safe to retry. |

### Implications for Terraform Providers

**Non-Idempotent Create Operations:**
- **Provider Responsibility:** Track resource IDs in Terraform state to prevent duplicates
- **Recommended:** Check for existing resources by name before creating
- **Future Enhancement:** Consider using `external_id` field (when added) for correlation

**Idempotent Update/Delete Operations:**
- **Safe to retry** on transient errors (network timeouts, 5xx errors)
- **Handle 404 gracefully** on delete (treat as already deleted, not an error)

**Retry Strategy:**
```
Create:  Do NOT retry (risk of duplicates)
Read:    Retry with exponential backoff
Update:  Retry with exponential backoff
Delete:  Retry; treat 404 as success
```

---

## Error Code Reference

### Standard Error Response Format

Most endpoints use a simple error format:

```json
{
  "error": "error_code_string",
  "message": "Human-readable description"
}
```

**OAuth2 endpoints** follow RFC 6749 format:

```json
{
  "error": "invalid_client",
  "error_description": "Client authentication failed"
}
```

### Common Error Codes

| Code | HTTP Status | Description | Resolution |
|------|-------------|-------------|------------|
| `invalid_client` | 401 | OAuth client credentials incorrect | Verify client_id and client_secret |
| `invalid_token` | 401 | JWT token invalid or malformed | Obtain new access token |
| `expired_token` | 401 | JWT token has expired | Request new access token |
| `insufficient_permissions` | 403 | User role lacks required permissions | Use admin-role OAuth client |
| `Pizza not found` | 404 | Pizza ID does not exist | Verify ID exists via GET /pizzas |
| `Client not found` | 404 | OAuth client ID does not exist | Verify client exists |
| `Unauthorized to delete this pizza` | 403 | Pizza not owned by requesting user | Only creator can delete pizza |
| `binding error` | 400 | Request body validation failed | Check JSON structure and required fields |

### HTTP Status Code Usage

| Status | Usage | Examples |
|--------|-------|----------|
| **200 OK** | Successful GET, PUT, DELETE | Pizza retrieved, updated, or deleted |
| **201 Created** | Successful POST (resource created) | Pizza created, OAuth client created |
| **400 Bad Request** | Invalid request format or validation error | Missing required fields, invalid JSON |
| **401 Unauthorized** | Missing or invalid authentication | No token provided, token expired |
| **403 Forbidden** | Authenticated but insufficient permissions | User role not admin, not pizza owner |
| **404 Not Found** | Resource does not exist | Pizza ID not found, client ID not found |
| **500 Internal Server Error** | Server-side error | Database failure, unexpected panic |

---

## Rate Limits

**Current Status:** No rate limiting implemented

**Future Considerations:**
- If rate limiting is added, expect `429 Too Many Requests` status
- `Retry-After` header will indicate wait time
- Limit will be documented in this section

**Recommended Client Behavior:**
- Implement exponential backoff for 5xx errors
- Respect `Retry-After` header if rate limiting is added
- Avoid parallel requests for the same resource (prevent race conditions)

---

## Concurrency Behavior

### Safe Concurrent Operations

**READ Operations:**
- ✅ Safe to run concurrently (no side effects)
- Last-write-wins for simultaneous updates to same resource

**CREATE Operations:**
- ⚠️ Not safe if duplicate prevention is needed
- Multiple concurrent creates of identical data → multiple resources
- Provider should serialize create operations or implement client-side locking

**UPDATE Operations:**
- ⚠️ Last-write-wins (no optimistic locking)
- Concurrent updates to same resource may result in lost updates
- Provider should serialize updates to same resource

**DELETE Operations:**
- ✅ Safe to run concurrently
- First delete succeeds, subsequent deletes return 404 (idempotent)

### Race Condition Prevention

**No built-in mechanisms:**
- No ETags or If-Match headers for optimistic locking
- No versioning or conflict detection

**Provider Recommendations:**
- Use Terraform's built-in locking mechanisms
- Serialize operations on the same resource
- Accept last-write-wins behavior for updates

---

## Resource Identifiers

### ID Format

All resources use **integer IDs** (auto-incrementing database primary keys):

```json
{
  "id": 42,
  "name": "Margherita",
  ...
}
```

**Properties:**
- Type: Integer (int64)
- Uniqueness: Per resource type (Pizza ID 1 ≠ OAuthClient ID 1)
- Stability: IDs never change once assigned
- Predictability: Sequential, but do not rely on specific values

### ID Management for Providers

**State Tracking:**
- Store the ID returned from CREATE response in Terraform state
- Use the ID for subsequent READ, UPDATE, DELETE operations

**No UUID Support:**
- Future enhancement may add optional `external_id` field for provider correlation
- Currently, provider must map Terraform resource IDs to API integer IDs in state

---

## Data Format

### Request Content-Type

- **OAuth Token Endpoint:** `application/x-www-form-urlencoded`
- **All Other Endpoints:** `application/json`

### Response Content-Type

- **All Endpoints:** `application/json; charset=utf-8`

### Date/Time Format

**ISO 8601 / RFC 3339:**
```json
{
  "created_at": "2025-11-10T12:34:56Z",
  "updated_at": "2025-11-10T12:34:56Z"
}
```

**Timezone:** Always UTC (Z suffix)

### Soft Deletes

Resources use **soft delete** (GORM's `DeletedAt` field):
- Deleted resources are marked with a timestamp, not physically removed
- Deleted resources do not appear in list/get responses
- IDs of deleted resources are not reused

---

## API Stability and Breaking Changes

### Non-Breaking Changes (Allowed in v1)

- Adding new optional query parameters
- Adding new fields to responses (clients must ignore unknown fields)
- Adding new endpoints
- Adding new error codes
- Changing error message text (codes remain stable)

### Breaking Changes (Require v2)

- Removing or renaming fields
- Changing field data types
- Removing endpoints
- Changing authentication mechanism
- Changing URL structure

---

## Support and Contact

For questions about this API contract or integration assistance:

- **Documentation:** See [README.md](../README.md) and [TERRAFORM_PROVIDER.md](./TERRAFORM_PROVIDER.md)
- **Issues:** Open a GitHub issue in the repository
- **Examples:** See [test-api.sh](../scripts/test-api.sh) for working API usage examples

---

**Document Revision History:**

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-11 | Initial API contract documentation |
