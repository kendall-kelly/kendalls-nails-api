# Custom Nails API - Requirements Documentation

This directory contains the complete requirements documentation for the Custom Nails API project, organized into focused sections for easy navigation and maintenance.

## Table of Contents

### Part 1: Functional Requirements
What the system does - features, behaviors, and business logic.

1. [**User Roles**](./01-user-roles.md) - Customer, Nail Technician, and Anonymous User roles
2. [**Order Management**](./02-order-management.md) - Order submission, status workflow, review process, and history
3. [**Payment**](./03-payment.md) - Payment model and pricing structure
4. [**Design Gallery & Sharing**](./04-design-gallery.md) - Public gallery, design reuse, and comments
5. [**Communication**](./05-communication.md) - Direct messaging and notifications
6. [**Data Models**](./06-data-models.md) - High-level data structure overview
7. [**API Endpoints**](./07-api-endpoints.md) - Complete list of REST endpoints
8. [**Business Rules**](./08-business-rules.md) - Core business rules, future enhancements, and open questions

### Part 2: Non-Functional Requirements
How the system should be built - technical choices, architecture, and quality attributes.

9. [**Backend Framework**](./09-backend-framework.md) - Go, Gin, GORM, and REST architecture
10. [**Database**](./10-database.md) - PostgreSQL configuration, migrations, naming conventions, and indexing
11. [**Authentication**](./11-authentication.md) - Auth0 integration, JWT tokens, RBAC, and security
12. [**File Storage**](./12-file-storage.md) - AWS S3 configuration, image validation, and access control
13. [**API Design**](./13-api-design.md) - REST conventions, response formats, pagination, and error handling
14. [**Deployment & Infrastructure**](./14-deployment.md) - Heroku setup, environment variables, scaling, and costs

## Quick Reference

### Key Technologies
- **Backend**: Go + Gin framework + GORM
- **Database**: PostgreSQL (Heroku Postgres)
- **Authentication**: Auth0 (JWT-based)
- **File Storage**: AWS S3
- **Hosting**: Heroku
- **API Style**: RESTful (JSON)

## Document Maintenance

When updating requirements:
1. Update the specific section file
2. Ensure cross-references between documents remain valid
3. Keep this README in sync with major changes