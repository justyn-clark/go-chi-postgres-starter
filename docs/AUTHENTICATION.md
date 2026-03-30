# Authentication & Authorization Guide

This guide reflects the current auth flow and route protections in the codebase.

## Overview

The API currently includes:

- JWT login and authenticated requests
- Role-aware authorization (`user` and `admin`)
- Password reset request flow
- Password reset confirmation flow
- Authenticated password change flow
- Optional `API_ACCESS_TOKEN` header support for some service-to-service requests

## JWT token behavior

JWTs are generated in `cmd/api/services/user_service.go` and include:

- `user_id`
- `email`
- `role`
- `exp`
- `iat`

Current expiration is **7 days**.

If you change expiration behavior, update `generateToken` in `cmd/api/services/user_service.go`.

## Public auth endpoints

These routes do not require prior authentication:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/request-password-reset`
- `POST /api/auth/reset-password`

## Protected auth endpoint

Requires JWT Bearer auth:

- `POST /api/auth/change-password`

## User and admin authorization model

### Authenticated user

- `GET /api/users/me`

### Owner or admin

- `GET /api/users/{id}`
- `PUT /api/users/{id}`

### Admin only

- `GET /api/users`
- `POST /api/users`
- `DELETE /api/users/{id}`
- `PUT /api/users/{id}/role`

A newly registered user gets the default `user` role.

## Login flow

### Request

`POST /api/auth/login`

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

### Response

```json
{
  "token": "<jwt>",
  "user": {
    "id": "...",
    "email": "user@example.com",
    "full_name": "User Example",
    "role": "user"
  }
}
```

## Password reset flow

### 1. Request password reset

`POST /api/auth/request-password-reset`

```json
{
  "email": "user@example.com"
}
```

Response:

```json
{
  "message": "If an account with that email exists, a password reset link has been sent"
}
```

Notes:

- The handler always returns success-style messaging to reduce email enumeration risk.
- The current service implementation stores reset state in the database.
- The current implementation prints a localhost reset link to server output rather than sending email.

### 2. Reset password with token

`POST /api/auth/reset-password`

```json
{
  "token": "reset-token",
  "password": "newSecurePassword123"
}
```

Response:

```json
{
  "message": "Password has been reset successfully"
}
```

## Change password for authenticated users

`POST /api/auth/change-password`

Header:

```text
Authorization: Bearer <jwt>
```

Request:

```json
{
  "current_password": "oldPassword123",
  "new_password": "newSecurePassword123"
}
```

## API access token

If `API_ACCESS_TOKEN` is set, the JWT middleware also accepts:

```text
X-API-Token: <token>
```

Important behavior:

- This bypasses the login/JWT requirement for middleware-protected routes.
- It does **not** populate a user role.
- Because of that, it does **not** satisfy admin-only authorization checks.

So `X-API-Token` may be useful for non-admin protected requests, but it will not grant access to routes guarded by `RequireAdmin`.

## Making a user admin

The repo supports admin roles. A direct SQL update is the simplest current path:

```sql
UPDATE users SET role = 'admin' WHERE email = 'admin@example.com';
```

After that, log in again so the new JWT includes the updated role claim.

## Required migration state

The current auth model depends on both checked-in migrations:

- `001_initial_schema`
- `002_add_roles_and_password_reset`

Run them with:

```bash
make migrate-up
```

## Quick examples

### Register

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","full_name":"Test User","password":"password123"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

### Get your own profile

```bash
curl -X GET http://localhost:8080/api/users/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Promote a user to admin

```bash
curl -X PUT http://localhost:8080/api/users/USER_ID/role \
  -H "Authorization: Bearer ADMIN_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role":"admin"}'
```
