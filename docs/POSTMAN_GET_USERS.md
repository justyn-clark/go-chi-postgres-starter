# How to Call GET /api/users in Postman

The `GET /api/users` endpoint requires authentication. Here are two ways to call it:

## Method 1: Using JWT Token (Recommended for User Authentication)

### Step 1: Get a JWT Token

1. **Create a new request in Postman:**
   - Method: `POST`
   - URL: `http://localhost:8080/api/auth/login`
   - Body (raw JSON):

     ```json
     {
       "email": "testuser@example.com",
       "password": "password123"
     }
     ```

2. **Send the request** and copy the `token` from the response

### Step 2: Use the Token to Get Users

1. **Create a new request:**
   - Method: `GET`
   - URL: `http://localhost:8080/api/users`
   - Optional query params: `?limit=10&offset=0`

2. **Go to Authorization tab:**
   - Type: Select **"Bearer Token"**
   - Token: Paste your JWT token (without "Bearer")
‰
3. **Send the request**

### Example with Query Parameters

```text
GET http://localhost:8080/api/users?limit=20&offset=0
Authorization: Bearer <your-jwt-token>
```

---

## Method 2: Using API_ACCESS_TOKEN (For Service-to-Service)

### Step 1: Start Server with API_ACCESS_TOKEN

```bash
export API_ACCESS_TOKEN="my-secret-token"
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter?sslmode=disable"
go run ./cmd/api
```

### Step 2: Use API_ACCESS_TOKEN in Postman

1. **Create a new request:**
   - Method: `GET`
   - URL: `http://localhost:8080/api/users`
   - Optional query params: `?limit=10&offset=0`

2. **Go to Headers tab** (NOT Authorization):
   - Key: `X-API-Token`
   - Value: `my-secret-token` (must match what you set on server)

3. **Send the request**

### Example

```text
GET http://localhost:8080/api/users?limit=10&offset=0
X-API-Token: my-secret-token
```

---

## Query Parameters

Both methods support pagination:

- `limit` (optional) - Number of users to return (default: 10)
- `offset` (optional) - Number of users to skip (default: 0)

### Examples

```text
GET /api/users                    # First 10 users
GET /api/users?limit=20           # First 20 users
GET /api/users?limit=10&offset=10 # Users 11-20
```

---

## Quick Test with curl

### Using JWT Token

```bash
# 1. Get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"testuser@example.com","password":"password123"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

# 2. Get users
curl -X GET "http://localhost:8080/api/users?limit=10&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

### Using API_ACCESS_TOKEN

```bash
# Start server with: export API_ACCESS_TOKEN="test-token"

# Get users
curl -X GET "http://localhost:8080/api/users?limit=10&offset=0" \
  -H "X-API-Token: test-token"
```

---

## Expected Response

```json
[
  {
    "id": "db710410-295a-4064-a262-092afc0ff8e8",
    "email": "testuser@example.com",
    "full_name": "Test User",
    "created_at": "2025-11-09T17:05:09.705029-08:00",
    "updated_at": "2025-11-09T17:05:09.705029-08:00"
  }
]
```

---

## Troubleshooting

### "authorization header required"

- **JWT Method:** Make sure you're using the Authorization tab with "Bearer Token" type
- **API_ACCESS_TOKEN Method:** Make sure `API_ACCESS_TOKEN` is set on the server and matches the header value

### "invalid or expired token"

- Get a fresh JWT token by logging in again
- Tokens expire after 7 days

### Empty array `[]`

- No users in database yet
- Register a user first: `POST /api/auth/register`
