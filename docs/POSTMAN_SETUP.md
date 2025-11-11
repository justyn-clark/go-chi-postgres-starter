# Postman Setup Guide

## Getting a Fresh JWT Token

### Step 1: Login to Get Token

1. **Start your API server:**

   ```bash
   make run
   # Or manually:
   export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
   export JWT_SECRET="your-secret-key"
   go run ./cmd/api
   ```

2. **Login via Postman or curl:**

   ```bash
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"testuser@example.com","password":"password123"}'
   ```

3. **Copy the `token` from the response**

### Step 2: Set Token in Postman

#### Option A: Bearer Token (Recommended)

1. In Postman, select your request
2. Go to **Authorization** tab
3. Select **Type: Bearer Token**
4. Paste your token in the **Token** field
5. Postman automatically adds `Authorization: Bearer <token>` header

#### Option B: Manual Header

1. Go to **Headers** tab
2. Add header:
   - Key: `Authorization`
   - Value: `Bearer YOUR_TOKEN_HERE` (include the word "Bearer" and a space!)

### Common Issues

#### "Invalid or expired token"

##### Causes

1. **Token expired** - Tokens expire after 7 days. Get a fresh token.
2. **JWT_SECRET mismatch** - Server restarted with different secret
3. **Wrong format** - Missing "Bearer " prefix or extra spaces
4. **Old token** - Token from previous server instance

##### Solutions

1. **Get a fresh token:**

   ```bash
   # Login again to get new token
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"your-email@example.com","password":"your-password"}'
   ```

2. **Check JWT_SECRET is consistent:**
   - Make sure `JWT_SECRET` environment variable matches between token generation and validation
   - Default: `dev-secret-change-in-production`
   - If you change it, you need a new token

3. **Verify token format in Postman:**
   - Should be: `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`
   - NOT: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...` (missing Bearer)
   - NOT: `Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...` (extra spaces)

#### Token Expiration

Tokens are valid for **7 days**. After that, you need to login again to get a new token.

#### Server Restart

If you restart the server with a different `JWT_SECRET`, all existing tokens become invalid. You must:

1. Use the same `JWT_SECRET` as when the token was generated, OR
2. Get a new token after restart

## Quick Test

```bash
# 1. Get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"testuser@example.com","password":"password123"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

# 2. Test token
curl -X GET http://localhost:8080/api/users \
  -H "Authorization: Bearer $TOKEN"
```

## API Access Token (Service-to-Service Auth)

The `API_ACCESS_TOKEN` is **NOT** a Bearer token. It uses a different header.

### How to Use API_ACCESS_TOKEN in Postman

1. **Set the environment variable:**

   ```bash
   export API_ACCESS_TOKEN="your-api-token-value"
   ```

2. **In Postman:**
   - Go to **Headers** tab (NOT Authorization tab)
   - Add header:
     - Key: `X-API-Token`
     - Value: `your-api-token-value`

3. **This bypasses JWT authentication** - useful for service-to-service calls

### Difference Between JWT Token and API Access Token

| Type | Header | Use Case |
|------|--------|----------|
| **JWT Token** | `Authorization: Bearer <token>` | User authentication (from login) |
| **API Access Token** | `X-API-Token: <token>` | Service-to-service authentication |

### Example: Using API Access Token

```bash
# Set API_ACCESS_TOKEN when starting server
export API_ACCESS_TOKEN="my-secret-api-token"
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
go run ./cmd/api

# In Postman, add header:
# X-API-Token: my-secret-api-token
```

### Troubleshooting: "authorization header required" Error

If you see this error even with `X-API-Token` header set:

1. **Check server has API_ACCESS_TOKEN set:**

   ```bash
   # When starting server, make sure you export it:
   export API_ACCESS_TOKEN="your-token-value"
   go run ./cmd/api
   ```

2. **Verify the token value matches exactly:**
   - Server: `export API_ACCESS_TOKEN="test-token-123"`
   - Postman Header: `X-API-Token: test-token-123` (must match exactly)

3. **Check server logs:**
   - Look for any errors about missing API_ACCESS_TOKEN
   - Default behavior: If `API_ACCESS_TOKEN` is not set, only JWT Bearer tokens work

4. **Quick test:**

   ```bash
   # Start server with token
   export API_ACCESS_TOKEN="test-token"
   go run ./cmd/api
   
   # Test in another terminal
   curl -X GET http://localhost:8080/api/users \
     -H "X-API-Token: test-token"
   ```

## Postman Environment Variables

You can set up Postman environment variables:

1. Create a new Environment in Postman
2. Add variables:
   - Variable: `jwt_token` (for Bearer auth)
   - Variable: `api_token` (for X-API-Token header)
3. In Authorization tab, use: `Bearer {{jwt_token}}`
4. In Headers tab, add: `X-API-Token: {{api_token}}`

This way you can update tokens in one place and they apply to all requests.
