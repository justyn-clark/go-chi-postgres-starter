# Authentication & User Management Guide

## Overview

This API now includes comprehensive authentication features:

- **Password Reset** - Users can reset forgotten passwords
- **Admin Roles** - Role-based access control
- **JWT Token Management** - 7-day expiration with role-based claims
- **Password Change** - Logged-in users can change their passwords

## JWT Token Expiration

**Current Settings:**

- **Expiration**: 7 days (168 hours)
- **Location**: `cmd/api/services/user_service.go` line 130
- **Token includes**: `user_id`, `email`, `role`, `exp`, `iat`

### How Token Expiration Works

1. **Token Generation**: When a user logs in, a JWT is created with:
   - `exp`: Expiration timestamp (7 days from now)
   - `iat`: Issued at timestamp

2. **Token Validation**: The middleware automatically checks:
   - Token signature (using JWT_SECRET)
   - Expiration time
   - Token format

3. **Expired Tokens**: When a token expires:
   - API returns `401 Unauthorized` with "invalid or expired token"
   - User must login again to get a new token

### Changing Token Expiration

To change the expiration time, edit `cmd/api/services/user_service.go`:

```go
"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
// Change to:
"exp": time.Now().Add(time.Hour * 24).Unix(), // 1 day
// Or:
"exp": time.Now().Add(time.Hour * 24 * 30).Unix(), // 30 days
```

## Password Reset Flow

### 1. Request Password Reset

**Endpoint**: `POST /api/auth/request-password-reset`

**Request:**

```json
{
  "email": "user@example.com"
}
```

**Response:**

```json
{
  "message": "If an account with that email exists, a password reset link has been sent"
}
```

**What happens:**

- Generates a secure reset token (UUID)
- Stores token in database with 1-hour expiration
- **Currently**: Token is printed to console/logs
- **Production**: Should send email with reset link

**Security Note**: Always returns success (even if email doesn't exist) to prevent email enumeration attacks.

### 2. Reset Password

**Endpoint**: `POST /api/auth/reset-password`

**Request:**

```json
{
  "token": "reset-token-from-email",
  "password": "newSecurePassword123"
}
```

**Response:**

```json
{
  "message": "Password has been reset successfully"
}
```

**What happens:**

- Validates token (must exist and not expired)
- Hashes new password
- Updates user password
- Clears reset token

**Token Expiration**: Reset tokens expire after 1 hour

## Change Password (Logged-in Users)

**Endpoint**: `POST /api/auth/change-password`

**Headers**: `Authorization: Bearer <jwt-token>`

**Request:**

```json
{
  "current_password": "oldPassword123",
  "new_password": "newSecurePassword123"
}
```

**Response:**

```json
{
  "message": "Password has been changed successfully"
}
```

## Admin Roles

### User Roles

- **`user`** (default): Regular user
- **`admin`**: Administrator with elevated privileges

### Creating an Admin User

**Option 1: Via Database (Direct)**

```sql
UPDATE users SET role = 'admin' WHERE email = 'admin@example.com';
```

**Option 2: Via API (Requires Admin)**

```bash
PUT /api/users/{user-id}/role
Authorization: Bearer <admin-jwt-token>
Content-Type: application/json

{
  "role": "admin"
}
```

### Admin-Only Endpoints

- `PUT /api/users/{id}/role` - Update user role (admin only)

### Checking User Role

The JWT token includes the user's role. After login, check the `role` field in the token claims.

## Migration Required

**⚠️ IMPORTANT**: Run the migration to add role and password reset fields:

```bash
make migrate-up
```

Or manually:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

This will:

- Add `role` column (default: 'user')
- Add `password_reset_token` column
- Add `password_reset_expires_at` column
- Create indexes for performance

## API Endpoints Summary

### Public Endpoints (No Auth)

- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login and get JWT
- `POST /api/auth/request-password-reset` - Request password reset
- `POST /api/auth/reset-password` - Reset password with token

### Protected Endpoints (Require JWT)

- `POST /api/auth/change-password` - Change password (logged-in users)
- `GET /api/users` - List users
- `POST /api/users` - Create user
- `GET /api/users/{id}` - Get user
- `PUT /api/users/{id}` - Update user
- `DELETE /api/users/{id}` - Delete user

### Admin-Only Endpoints (Require Admin Role)

- `PUT /api/users/{id}/role` - Update user role

## Example Workflows

### Forgot Password Workflow

1. User requests reset:

```bash
curl -X POST http://localhost:8080/api/auth/request-password-reset \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

2. Check server logs for reset token (or email in production)

3. User resets password:

```bash
curl -X POST http://localhost:8080/api/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "token": "token-from-logs",
    "password": "newPassword123"
  }'
```

4. User logs in with new password:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "newPassword123"
  }'
```

### Making a User Admin

1. Login as existing admin (or create one via database)

2. Update user role:

```bash
curl -X PUT http://localhost:8080/api/users/{user-id}/role \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

### Token Expired? Login Again

When you get `401 Unauthorized` with "invalid or expired token":

1. Login again:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'
```

2. Use the new token in subsequent requests

## Security Best Practices

1. **Password Reset Tokens**:
   - Expire after 1 hour
   - Single-use (cleared after password reset)
   - Secure UUID generation

2. **JWT Tokens**:
   - Include role for authorization
   - 7-day expiration (adjustable)
   - Signed with JWT_SECRET

3. **Email Enumeration Prevention**:
   - Password reset always returns success
   - Doesn't reveal if email exists

4. **Role-Based Access**:
   - Admin middleware protects sensitive endpoints
   - Role checked from JWT token

## Production Considerations

1. **Email Service**: Implement email sending for password reset (currently logs to console)

2. **Token Refresh**: Consider implementing refresh tokens for better UX

3. **Rate Limiting**: Add rate limiting to password reset endpoints

4. **Audit Logging**: Log admin actions (role changes, etc.)

5. **Password Policy**: Consider adding password complexity requirements
