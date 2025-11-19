# Testing Guide

This document explains the testing strategy and how to run tests for the Custom Nails API.

## Testing Philosophy

We follow a three-tier testing approach:

1. **Unit Tests** - Test individual functions and methods in isolation
2. **Integration Tests** - Test how components work together
3. **Acceptance Tests** - Test complete user workflows end-to-end

## Test Structure

```
kendalls-nails-api/
├── middleware/
│   ├── auth.go
│   └── auth_test.go              # Unit tests for middleware
├── tests/
│   ├── integration/
│   │   └── auth_integration_test.go   # Integration tests
│   ├── acceptance/
│   │   └── auth_acceptance_test.go    # Acceptance tests
│   └── testutil/
│       └── auth_helpers.go            # Test utilities
```

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Run Tests with Verbose Output
```bash
go test -v ./...
```

### Run Specific Test Suite
```bash
# Unit tests only
go test ./middleware/...

# Integration tests only
go test ./tests/integration/...

# Acceptance tests only
go test ./tests/acceptance/...
```

### Run a Specific Test
```bash
go test -run TestCustomClaims_HasScope ./middleware/...
```

### Generate Coverage Report
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Categories

### 1. Unit Tests (`middleware/auth_test.go`)

Tests individual functions in isolation:

- ✅ `TestCustomClaims_HasScope` - Scope validation logic
- ✅ `TestGetUserID` - User ID extraction from context
- ✅ `TestGetClaims` - Claims extraction from context
- ✅ `TestRequireScope` - Scope-based authorization middleware
- ✅ `TestAuthError` - Error struct functionality

**Run:**
```bash
go test -v ./middleware/
```

### 2. Integration Tests (`tests/integration/auth_integration_test.go`)

Tests how middleware integrates with Gin router:

- ✅ Public endpoints accessible without auth
- ✅ Protected endpoints reject requests without tokens
- ✅ Protected endpoints reject invalid tokens
- ✅ Protected endpoints reject malformed auth headers
- ✅ Error response format consistency

**Run:**
```bash
go test -v ./tests/integration/
```

### 3. Acceptance Tests (`tests/acceptance/auth_acceptance_test.go`)

Tests complete user workflows:

- ✅ Health endpoint accessibility
- ✅ Complete authentication workflow (no auth → invalid token → valid token)
- ✅ Error response format validation
- ✅ Content-Type headers
- ✅ API contract compliance

**Run:**
```bash
go test -v ./tests/acceptance/
```

## Test Utilities

The `tests/testutil` package provides helpful functions for testing:

### MockValidatedClaims
Creates mock JWT claims for testing:
```go
claims := testutil.MockValidatedClaims(
    "auth0|123456",              // subject
    "https://test.auth0.com/",   // issuer
    []string{"read:messages"},   // scopes
)
```

### SetMockAuthContext
Sets up authenticated context in tests:
```go
c, _ := testutil.CreateTestContext()
testutil.SetMockAuthContext(c, "auth0|123", "https://test.auth0.com/", []string{"read:messages"})
```

## Environment Variables for Testing

Tests use these environment variables:

```bash
GO_ENV=test
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/kendalls_nails_test?sslmode=disable
AUTH0_DOMAIN=test.auth0.com
AUTH0_AUDIENCE=https://api.test.com
PORT=8080
```

### Skipping Auth Tests

If you don't have Auth0 configured, you can skip auth tests:

```bash
SKIP_AUTH_TESTS=true go test ./...
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v -cover ./...
      - name: Run tests with race detector
        run: go test -race ./...
```

## Testing Best Practices

### 1. Test Naming
- Use descriptive test names: `TestFunctionName_Scenario`
- Example: `TestGetUserID_WhenUserIDNotFound_ReturnsError`

### 2. Table-Driven Tests
```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    {"valid input", "auth0|123", "auth0|123", false},
    {"empty input", "", "", true},
}
```

### 3. Test Independence
- Each test should be independent
- Use `SetupTest()` and `TearDownTest()` in test suites
- Don't rely on test execution order

### 4. Mock External Dependencies
- Mock Auth0 token validation for unit tests
- Use test doubles for database connections
- Avoid real API calls in tests

### 5. Assert Clearly
```go
// Good
assert.Equal(t, http.StatusOK, resp.StatusCode, "should return 200 OK")

// Not as good
assert.Equal(t, 200, resp.StatusCode)
```

## Coverage Goals

- **Unit Tests**: > 80% coverage for business logic
- **Integration Tests**: Cover all major API workflows
- **Acceptance Tests**: Cover critical user journeys

## Current Test Coverage

Run this command to see current coverage:
```bash
go test -cover ./...
```

## Writing New Tests

When adding new features, follow this checklist:

- [ ] Write unit tests for new functions
- [ ] Write integration tests for new endpoints
- [ ] Write acceptance tests for new workflows
- [ ] Update this documentation if adding new test patterns
- [ ] Ensure tests pass locally before committing
- [ ] Run tests with race detector: `go test -race ./...`

## Common Testing Patterns

### Testing HTTP Handlers

```go
func TestHandler(t *testing.T) {
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

    handler(c)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Testing Middleware

```go
func TestMiddleware(t *testing.T) {
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    middleware := MyMiddleware()
    middleware(c)

    assert.False(t, c.IsAborted())
}
```

### Testing with Mock Auth

```go
func TestProtectedHandler(t *testing.T) {
    c, _ := testutil.CreateTestContext()
    testutil.SetMockAuthContext(c, "auth0|123", "https://test.auth0.com/", []string{})

    handler(c)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Troubleshooting

### Tests Failing with "connection refused"
- Ensure test database is running
- Check `DATABASE_URL` environment variable

### Tests Failing with Auth0 errors
- Set `SKIP_AUTH_TESTS=true` if Auth0 not configured
- Verify `AUTH0_DOMAIN` and `AUTH0_AUDIENCE` are set

### Race Detector Errors
- Run: `go test -race ./...`
- Fix any data races before committing

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Gin Testing Guide](https://github.com/gin-gonic/gin#testing)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
