# Testing Guide

This starter includes comprehensive integration tests for all API endpoints.

## Test Coverage

All API endpoints are tested:

### Health Endpoint

- ✅ Health check returns OK status
- ✅ Database health monitoring

### Authentication Endpoints

- ✅ Register new user
- ✅ Register duplicate email (conflict handling)
- ✅ Register with invalid data (validation)
- ✅ Login with valid credentials
- ✅ Login with invalid credentials

### User Management Endpoints

- ✅ List users (requires authentication)
- ✅ List users with pagination
- ✅ Get user by ID (requires authentication)
- ✅ Get non-existent user (404 handling)
- ✅ Create user (requires authentication)
- ✅ Update user (requires authentication)
- ✅ Delete user (requires authentication)
- ✅ All endpoints verify authentication requirements

## Running Tests

### Prerequisites

1. **Test Database**: Create a test database

   ```bash
   createdb go_api_starter_test
   ```

2. **Run Migrations**: Apply migrations to test database

   ```bash
   export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/go_api_starter_test?sslmode=disable"
   make migrate-up
   ```

### Running All Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with verbose output
go test -v ./tests/...
```

### Running Specific Tests

```bash
# Run only health endpoint tests
go test -v ./tests -run TestHealthEndpoint

# Run only authentication tests
go test -v ./tests -run TestAuthEndpoints

# Run only user management tests
go test -v ./tests -run TestUserEndpoints
```

### Test Database Configuration

Tests use the following database URL priority:

1. `TEST_DATABASE_URL` environment variable
2. `DATABASE_URL` environment variable
3. Default: `postgresql://postgres:postgres@localhost:5432/go_api_starter_test?sslmode=disable`

```bash
# Use custom test database
export TEST_DATABASE_URL="postgresql://user:pass@localhost:5432/my_test_db?sslmode=disable"
go test ./tests/...
```

## Test Structure

Tests are organized in `tests/api_test.go`:

- **setupTestRouter()**: Creates a test router with all routes configured
- **teardownTest()**: Cleans up test data after each test
- **TestHealthEndpoint()**: Tests health check endpoint
- **TestAuthEndpoints()**: Tests authentication (register/login)
- **TestUserEndpoints()**: Tests user CRUD operations

## Test Features

### Table-Driven Tests

Tests use table-driven patterns for multiple scenarios:

```go
testCases := []struct {
    name    string
    request models.RegisterRequest
    want    int
}{
    // Multiple test cases
}
```

### Authentication Testing

Tests verify:

- JWT token generation on login
- Token validation on protected routes
- Unauthorized access rejection

### Error Handling

Tests verify proper HTTP status codes:

- `200` - Success
- `201` - Created
- `400` - Bad Request (validation errors)
- `401` - Unauthorized
- `404` - Not Found
- `409` - Conflict (duplicate resources)

### Data Cleanup

Tests clean up data using `TRUNCATE` to ensure isolation between test runs.

## CI/CD Integration

Tests run automatically in CI/CD (GitHub Actions):

- Uses PostgreSQL 16 in GitHub Actions
- Tests run on every push and pull request
- See `.github/workflows/ci.yml` for configuration

## Writing New Tests

When adding new endpoints, follow this pattern:

```go
func TestNewEndpoint(t *testing.T) {
    router := setupTestRouter(t)
    defer teardownTest(t)

    t.Run("test case name", func(t *testing.T) {
        // Setup
        req := httptest.NewRequest("METHOD", "/api/endpoint", body)
        req.Header.Set("Authorization", "Bearer "+token)
        w := httptest.NewRecorder()

        // Execute
        router.ServeHTTP(w, req)

        // Assert
        assert.Equal(t, http.StatusOK, w.Code)
        // ... more assertions
    })
}
```

## Best Practices

1. **Use `require` for setup**: Fail fast if setup fails
2. **Use `assert` for validations**: Continue testing even if one assertion fails
3. **Clean up data**: Always call `teardownTest()` to prevent test pollution
4. **Test both success and error cases**: Verify error handling
5. **Test authentication**: Verify protected routes require auth
6. **Use descriptive test names**: Make it clear what each test validates

## Troubleshooting

### Database Connection Errors

If tests fail with database connection errors:

1. Ensure PostgreSQL is running
2. Check database exists: `psql -l | grep go_api_starter_test`
3. Verify connection string format
4. Check database permissions

### Migration Errors

If tests fail with migration errors:

1. Ensure migrations are up to date: `make migrate-up`
2. Check migration files exist in `migrations/`
3. Verify database schema matches migrations

### Test Isolation Issues

If tests interfere with each other:

1. Ensure `teardownTest()` is called in each test
2. Use unique test data (unique emails, IDs)
3. Consider using database transactions for better isolation
