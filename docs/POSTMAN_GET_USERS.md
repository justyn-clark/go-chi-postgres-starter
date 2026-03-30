# How to Call `GET /api/users` in Postman

`GET /api/users` is **admin only** in the current codebase.

If you authenticate as a normal user, this endpoint should return `403 Forbidden`.

## What to use instead as a normal user

If you only need to verify auth is working, use:

- `GET /api/users/me`
- `GET /api/users/{your-user-id}`

## Method 1: Admin JWT (required)

### Step 1: Log in as an admin user

Create a `POST` request to:

```text
http://localhost:8080/api/auth/login
```

Body:

```json
{
  "email": "admin@example.com",
  "password": "password123"
}
```

Copy the returned JWT token.

### Step 2: Call `GET /api/users`

Create a `GET` request to:

```text
http://localhost:8080/api/users?limit=10&offset=0
```

Set Bearer auth with the admin token.

## Why `X-API-Token` is not enough here

The middleware supports `X-API-Token` when `API_ACCESS_TOKEN` is configured, but that bypass does **not** establish admin role context.

Because `GET /api/users` is wrapped with `RequireAdmin`, `X-API-Token` alone is not sufficient for this endpoint.

## Query parameters

- `limit` - number of users to return, default `10`
- `offset` - number of users to skip, default `0`

Examples:

```text
GET /api/users
GET /api/users?limit=20
GET /api/users?limit=10&offset=10
```

## Example curl

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password123"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

curl -X GET "http://localhost:8080/api/users?limit=10&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

## Expected response

```json
[
  {
    "id": "db710410-295a-4064-a262-092afc0ff8e8",
    "email": "testuser@example.com",
    "full_name": "Test User",
    "role": "user",
    "created_at": "2025-11-09T17:05:09.705029-08:00",
    "updated_at": "2025-11-09T17:05:09.705029-08:00"
  }
]
```

## Troubleshooting

### `403 Forbidden`

You authenticated, but the JWT belongs to a non-admin user.

### `401 Unauthorized`

- Missing or invalid Bearer token
- Expired JWT
- Wrong `JWT_SECRET` on the server
