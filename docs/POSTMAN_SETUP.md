# Postman Setup Guide

This guide reflects the current auth and authorization behavior in the repo.

## Get a JWT token

### 1. Start the API

```bash
make run
```

### 2. Log in

Create a `POST` request to:

```text
http://localhost:8080/api/auth/login
```

Body:

```json
{
  "email": "testuser@example.com",
  "password": "password123"
}
```

Copy the `token` field from the response.

## Use the token in Postman

### Option A: Bearer Token

1. Open your request
2. Go to **Authorization**
3. Select **Bearer Token**
4. Paste the JWT token

### Option B: Manual header

Add:

```text
Authorization: Bearer YOUR_TOKEN_HERE
```

## Good first authenticated request

Use `GET /api/users/me`:

```text
http://localhost:8080/api/users/me
```

A normal freshly registered user can access this route.

## Important authorization note

`GET /api/users` is **admin only** in the current codebase.

So if you log in as a normal user and then call `/api/users`, a `403 Forbidden` response is expected.

## API access token mode

If the server is started with `API_ACCESS_TOKEN` set, the JWT middleware also accepts:

```text
X-API-Token: <token>
```

### Example server startup

```bash
export API_ACCESS_TOKEN="my-secret-token"
make run
```

### In Postman

Use the **Headers** tab, not Bearer auth:

```text
X-API-Token: my-secret-token
```

## Important limitation of API access token mode

The API access token bypass does **not** populate user role context.

That means it does **not** grant access to admin-only routes such as:

- `GET /api/users`
- `POST /api/users`
- `DELETE /api/users/{id}`
- `PUT /api/users/{id}/role`

Use a real admin JWT for those routes.

## Troubleshooting

### "invalid or expired token"

- Get a fresh token via login
- Ensure the server is using the same `JWT_SECRET`
- Make sure the header is exactly `Authorization: Bearer <token>`

### `403 Forbidden` on `/api/users`

That usually means you authenticated successfully but are not an admin.

### `401 Unauthorized` with `X-API-Token`

- Confirm `API_ACCESS_TOKEN` is set in the server environment
- Confirm the header value matches exactly

## Quick curl checks

### JWT flow

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"testuser@example.com","password":"password123"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

curl -X GET http://localhost:8080/api/users/me \
  -H "Authorization: Bearer $TOKEN"
```

### API access token flow

```bash
export API_ACCESS_TOKEN="test-token"
make run

curl -X GET http://localhost:8080/api/health \
  -H "X-API-Token: test-token"
```
