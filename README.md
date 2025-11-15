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

### For New Developers

**First time?** Read these in order:
1. [User Roles](./requirements/01-user-roles.md) - Who uses the system
2. [Order Management](./requirements/02-order-management.md) - Core workflow
3. [Backend Framework](./requirements/09-backend-framework.md) - Tech stack
4. [API Endpoints](./requirements/07-api-endpoints.md) - What to implement

## Project Status

This project is currently in the requirements and planning phase.