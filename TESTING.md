# Testing Guide

This document explains the testing strategy and setup for the Custom Nails API.

## Test Database Isolation

The project uses **separate databases** for development and testing to ensure complete isolation:

- **Development Database**: `kendalls_nails` - Used when running the application
- **Test Database**: `kendalls_nails_test` - Used exclusively by the test suite

Both databases run on the same PostgreSQL server instance, but they are completely isolated from each other.

## Quick Start

### 1. Setup Test Database (One-time)

```bash
./scripts/setup-test-db.sh
```

Or manually:

```bash
docker exec kendalls-nails-postgres psql -U postgres -c "CREATE DATABASE kendalls_nails_test;"
```

### 2. Run Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Structure

The project follows a three-tier testing approach:

### 1. Unit Tests

**Location**: `*_test.go` files in each package
**Purpose**: Test individual functions and structs in isolation
**Examples**:
- [models/user_test.go](models/user_test.go) - User model tests
- [config/database_test.go](config/database_test.go) - Database configuration tests

**Run unit tests**:
```bash
go test -v ./models
go test -v ./config
```

### 2. Integration Tests

**Location**: [integration_test.go](integration_test.go)
**Purpose**: Test components working together (e.g., database + models)
**Examples**:
- Database connection and migration
- User creation and querying
- Database constraints (uniqueness, etc.)

**Run integration tests**:
```bash
go test -v -run TestDatabase
go test -v -run TestCreate
```

### 3. Acceptance Tests

**Location**: [acceptance_test.go](acceptance_test.go)
**Purpose**: End-to-end tests validating acceptance criteria
**Examples**:
- Full API endpoint tests
- Response format validation
- Performance benchmarks

**Run acceptance tests**:
```bash
go test -v -run Acceptance
go test -v -run TestFullAPI
```

## Test Coverage

Current test coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Current Coverage**:
- `config` package: 100%
- `models` package: 100%
- Overall project: ~48%

## Database Isolation Tests

Special tests verify that development and test databases remain isolated:

**[database_isolation_test.go](database_isolation_test.go)**:
- `TestDatabaseIsolation` - Verifies tests use the correct database
- `TestDevelopmentDatabaseUnaffectedByTests` - Ensures dev DB protection

Run these tests:
```bash
go test -v -run TestDatabaseIsolation
go test -v -run TestDevelopmentDatabase
```

## Utilities

### Reset Test Database

If your test database gets into a bad state:

```bash
./scripts/reset-test-db.sh
```

This will:
1. Drop the `kendalls_nails_test` database
2. Recreate it fresh
3. Tests will auto-migrate tables on next run

### Skip Database Tests

If you need to run tests without a database:

```bash
SKIP_DB_TESTS=true go test ./...
```

### Custom Test Database

Override the test database URL:

```bash
export TEST_DATABASE_URL="postgresql://user:pass@localhost:5432/my_test_db?sslmode=disable"
go test ./...
```

## Testing Best Practices

### 1. Test Isolation

Each test should:
- ✅ Clean up after itself
- ✅ Not depend on other tests
- ✅ Use `setupTestDB()` and `defer cleanupTestDB()`
- ✅ Be able to run in any order

### 2. Test Naming

- Unit tests: `TestFunctionName`
- Integration tests: `TestFeatureIntegration`
- Acceptance tests: `TestFeatureAcceptance`

### 3. Assertions

Use `testify` assertions:
```go
assert.Equal(t, expected, actual, "description")
require.NoError(t, err, "description") // fails fast
```

### 4. Test Data

- Use descriptive email addresses: `test-feature@example.com`
- Clean up in `defer cleanupTestDB(t)`
- Avoid hardcoded IDs (use auto-increment)

## Continuous Integration

When setting up CI/CD:

1. **GitHub Actions example**:
```yaml
- name: Setup test database
  run: |
    docker compose up -d
    sleep 5
    ./scripts/setup-test-db.sh

- name: Run tests
  run: go test ./... -v -coverprofile=coverage.out

- name: Upload coverage
  uses: codecov/codecov-action@v3
```

2. **Required environment variables**:
```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/kendalls_nails?sslmode=disable
TEST_DATABASE_URL=postgresql://postgres:postgres@localhost:5432/kendalls_nails_test?sslmode=disable
```

## Test Maintenance

### Adding New Tests

1. Create test file: `feature_test.go`
2. Use `setupTestDB(t)` for database tests
3. Add cleanup: `defer cleanupTestDB(t)`
4. Follow naming conventions
5. Update this document if adding new test categories

### Debugging Tests

```bash
# Run specific test with verbose output
go test -v -run TestSpecificTest

# Run with race detection
go test -race ./...

# Run with timeout
go test -timeout 30s ./...
```

## Common Issues

### Tests failing with "connection refused"

- Ensure PostgreSQL is running: `docker compose ps`
- Check DATABASE_URL in your environment

### Tests failing with "database does not exist"

- Run setup script: `./scripts/setup-test-db.sh`

### Tests affecting development database

- This should NEVER happen - see [database_isolation_test.go](database_isolation_test.go)
- If it does, it's a bug - file an issue immediately

## Next Steps

- Add more integration tests as features are implemented
- Set up CI/CD pipeline
- Add performance benchmarks
- Consider adding end-to-end tests with real HTTP clients

See [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for the development roadmap.
