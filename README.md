# Custom Nails API

Backend API for an online application enabling customers to order custom-designed nails. The platform connects customers with nail technicians who review, price, create, and ship custom nail orders.

## Key Technologies

- **Backend**: Go (Golang) with Gin framework and GORM ORM
- **Database**: PostgreSQL (via Heroku Postgres)
- **Authentication**: Auth0 (JWT-based)
- **File Storage**: AWS S3 (for design images)
- **Hosting**: Heroku
- **API Style**: RESTful with JSON

## Core Features

- Customer self-registration and order submission
- Nail technician invitation-based registration
- Order review and pricing workflow
- Design gallery with public/private sharing
- Direct messaging between customers and technicians
- PNG image upload and storage

## Documentation

📖 **[View Complete Requirements Documentation](./requirements/README.md)**

The requirements are organized into 14 focused sections covering both functional requirements (what the system does) and non-functional requirements (how it's built).

## Getting Started

### Install Go

Follow the [instructions](https://go.dev/doc/install) to install Go.

### Open Docker Desktop

Open Docker Desktop. If you don't have it installed, follow the [instructions](https://docs.docker.com/desktop/) to install it.

### Start backing services

This step starts up the required backing services for the application, such as the Postgres database:

   ```bash
   docker-compose up -d
   ```

### Copy default environment variables

This creates a local set of environment variables that you can customize for your machine. The defaults are probably fine for now:

   ```bash
   cp .env.example .env
   ```

### Setup Auth0 and configure environment variables

1. **Auth0 Account Setup**
   - Create a free Auth0 account at https://auth0.com
   - Create a new API in the Auth0 Dashboard
   - Note your Auth0 Domain and API Identifier (Audience)

2. **Environment Configuration**
   - Update `.env` with your Auth0 credentials:
     ```
     AUTH0_DOMAIN=https://[YOUR TENANT].us.auth0.com
     AUTH0_AUDIENCE=your-api-audience
     ```

Ensure your Auth0 settings are correct by following the [Testing Auth0](docs/TESTING_AUTH0.md) steps.

### Build the application

Compile the application:

   ```bash
   make build
   ```

### Run the application

Run the application:

   ```bash
   make run
   ```

### Verify the application started successfully

Send a request to the `/health` endpoint to verify that the application is running:

   ```bash
   curl localhost:8080/api/v1/health
   ```

The expected response is:

   ```bash
   {
     "message": "Custom Nails API is running",
     "success": true
   }
   ```

## Running Tests

The project uses a dedicated test database to ensure tests don't interfere with development data.

### Test Database Setup

The test database (`kendalls_nails_test`) is automatically created when you first start the PostgreSQL container with `docker-compose up -d`.

If the database already exists and you need to recreate it manually:

```bash
docker exec kendalls-nails-postgres psql -U postgres -c "CREATE DATABASE kendalls_nails_test;"
```

### Running Tests

Run all tests:

```bash
make test
```

Run tests with coverage:

```bash
make test-coverage
```

Run specific tests:

```bash
GO_ENV=test go test -v ./controllers -run TestCreateOrder
```

**Note**: Tests automatically use the `kendalls_nails_test` database when `GO_ENV=test` is set. The Makefile test commands set this automatically.

### Safety Features

Tests include a built-in safety check that prevents them from running without `GO_ENV=test`. This protects against accidental data loss by ensuring tests never run against development or production databases.

If you try to run tests without setting `GO_ENV=test`, you'll see an error:

```
╔════════════════════════════════════════════════════════════════╗
║                    SAFETY CHECK FAILED                         ║
║                                                                ║
║  Tests must run with GO_ENV=test to prevent data loss!        ║
║                                                                ║
║  Current GO_ENV: ""                                            ║
║                                                                ║
║  To run tests safely:                                          ║
║    make test                                                   ║
║    GO_ENV=test go test ./...                                   ║
╚════════════════════════════════════════════════════════════════╝
```

Always use `make test` or explicitly set `GO_ENV=test` when running tests.