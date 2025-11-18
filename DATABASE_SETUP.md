# Database Setup Guide

This guide covers setting up PostgreSQL for the Custom Nails API project.

## Option 1: Docker (Recommended for Development)

The easiest way to get started is using Docker Compose.

### Prerequisites
- Docker Desktop installed ([Download here](https://www.docker.com/products/docker-desktop))

### Steps

1. **Start PostgreSQL container:**
   ```bash
   docker-compose up -d
   ```

2. **Verify the database is running:**
   ```bash
   docker-compose ps
   ```
   You should see the `kendalls-nails-postgres` container running.

3. **Test database connection (optional):**
   ```bash
   docker exec -it kendalls-nails-postgres psql -U postgres -d kendalls_nails
   ```
   Type `\q` to exit.

4. **Stop the database when done:**
   ```bash
   docker-compose down
   ```

5. **To remove all data (reset database):**
   ```bash
   docker-compose down -v
   ```

## Option 2: Local PostgreSQL Installation

### macOS (using Homebrew)

1. **Install PostgreSQL:**
   ```bash
   brew install postgresql@15
   ```

2. **Start PostgreSQL service:**
   ```bash
   brew services start postgresql@15
   ```

3. **Create the database:**
   ```bash
   createdb kendalls_nails
   ```

4. **Verify connection:**
   ```bash
   psql -d kendalls_nails
   ```

### Linux (Ubuntu/Debian)

1. **Install PostgreSQL:**
   ```bash
   sudo apt update
   sudo apt install postgresql postgresql-contrib
   ```

2. **Start PostgreSQL service:**
   ```bash
   sudo systemctl start postgresql
   sudo systemctl enable postgresql
   ```

3. **Create database and user:**
   ```bash
   sudo -u postgres psql
   ```

   In the PostgreSQL prompt:
   ```sql
   CREATE DATABASE kendalls_nails;
   CREATE USER postgres WITH PASSWORD 'postgres';
   GRANT ALL PRIVILEGES ON DATABASE kendalls_nails TO postgres;
   \q
   ```

### Windows

1. **Download PostgreSQL installer:**
   Visit [PostgreSQL Downloads](https://www.postgresql.org/download/windows/)

2. **Run the installer:**
   - Choose version 15.x
   - Set password for postgres user: `postgres`
   - Default port: 5432

3. **Create the database:**
   - Open pgAdmin (installed with PostgreSQL)
   - Create a new database named `kendalls_nails`

## Environment Configuration

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit `.env` if needed:**
   The default DATABASE_URL should work for both Docker and local installations:
   ```
   DATABASE_URL=postgresql://postgres:postgres@localhost:5432/kendalls_nails?sslmode=disable
   ```

## Running the Application

1. **Make sure PostgreSQL is running** (Docker or local installation)

2. **Run the server:**
   ```bash
   go run main.go
   ```

3. **The application will automatically:**
   - Connect to the database
   - Create the `users` table via auto-migration
   - Start the web server on port 8080

4. **Test the database connection:**
   ```bash
   curl http://localhost:8080/api/v1/database/status
   ```

   Expected response:
   ```json
   {
     "success": true,
     "message": "Database connected",
     "tables": ["users"]
   }
   ```

## Verifying Database Tables

### Using Docker:
```bash
docker exec -it kendalls-nails-postgres psql -U postgres -d kendalls_nails -c "\dt"
```

### Using Local PostgreSQL:
```bash
psql -U postgres -d kendalls_nails -c "\dt"
```

You should see the `users` table listed.

### Inspect the users table structure:
```bash
# Docker
docker exec -it kendalls-nails-postgres psql -U postgres -d kendalls_nails -c "\d users"

# Local
psql -U postgres -d kendalls_nails -c "\d users"
```

Expected columns:
- `id` (primary key, auto-increment)
- `email` (unique, not null)
- `role` (default: 'customer')
- `created_at`
- `updated_at`
- `deleted_at` (for soft deletes)

## Troubleshooting

### Connection refused error
- **Docker:** Make sure the container is running: `docker-compose ps`
- **Local:** Make sure PostgreSQL service is running
  - macOS: `brew services list`
  - Linux: `sudo systemctl status postgresql`

### Port 5432 already in use
- Check if another PostgreSQL instance is running
- Change the port in `docker-compose.yml` (e.g., `"5433:5432"`)
- Update `DATABASE_URL` in `.env` accordingly

### Permission denied
- Make sure the postgres user has the correct password
- Check the DATABASE_URL format in `.env`

### Database doesn't exist
- For Docker: The database is created automatically
- For local: Run `createdb kendalls_nails` or create it via SQL

## Test Database Setup

The project uses a **separate test database** to ensure isolation between development and testing environments.

### Quick Setup

```bash
# Create the test database (one-time setup)
./scripts/setup-test-db.sh
```

### Manual Setup

If you prefer to create the test database manually:

```bash
# Using Docker
docker exec kendalls-nails-postgres psql -U postgres -c "CREATE DATABASE kendalls_nails_test;"

# Using local PostgreSQL
createdb kendalls_nails_test
# OR
psql -U postgres -c "CREATE DATABASE kendalls_nails_test;"
```

### Running Tests

Tests automatically use the dedicated test database (`kendalls_nails_test`):

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test suite
go test -v ./models
go test -v -run TestDatabase
```

### Resetting Test Database

If you need a clean test database:

```bash
./scripts/reset-test-db.sh
```

### Test Database Configuration

Tests use `kendalls_nails_test` database by default. You can override this with the `TEST_DATABASE_URL` environment variable:

```bash
export TEST_DATABASE_URL="postgresql://postgres:postgres@localhost:5432/my_custom_test_db?sslmode=disable"
go test ./...
```

### Database Isolation

- **Development database**: `kendalls_nails` (port 5432)
- **Test database**: `kendalls_nails_test` (port 5432, same server)

Both databases run on the same PostgreSQL server but are completely isolated from each other. Tests will never affect your development data.

## Next Steps

With the database set up, you're ready to:
1. Test the API endpoints
2. Run the test suite with `go test ./...`
3. Move on to Iteration 3: Configuration & Environment Variables
4. Continue with Iteration 4: Auth0 Integration

See [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for the full roadmap.
