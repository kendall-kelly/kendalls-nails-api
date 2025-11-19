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
   go build
   ```

### Run the application

Run the application:

   ```bash
   go run main.go
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