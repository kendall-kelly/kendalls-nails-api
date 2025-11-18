# CLAUDE.md - Custom Nails API

## Project Overview

## Tech Stack

- **Language**: Go (Golang)
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **ORM**: GORM (gorm.io/gorm)
- **Database**: PostgreSQL (Heroku Postgres in production)
- **Authentication**: Auth0 (JWT-based)
- **File Storage**: AWS S3
- **Deployment**: Heroku
- **API Style**: RESTful JSON API

## Documentation Structure

All project documentation follows a modular structure:

### Requirements (`/requirements/`)
Complete functional and non-functional requirements split into focused documents:
- Start with `requirements/README.md` for overview
- Functional requirements (01-08): User roles, order management, payment, gallery, etc.
- Non-functional requirements (09-14): Framework, database, auth, deployment, etc.

### Implementation Plan (`IMPLEMENTATION_PLAN.md`)
21 iterations organized into 7 phases, designed for incremental development:
- Each iteration is small, focused, and independently testable
- Starts with foundation (hello world, database, auth)
- Progresses through core features (orders, files, messaging, gallery)
- Ends with polish and deployment

**When implementing**: Follow the iteration order. Each builds on previous work.

## Naming Conventions

### Database
- **Tables**: plural, snake_case (`users`, `orders`, `design_comments`)
- **Columns**: snake_case (`created_at`, `user_id`, `image_url`)
- **Primary keys**: `id` (auto-increment)
- **Foreign keys**: `{table}_id` (e.g., `customer_id`, `technician_id`)
- **Timestamps**: `created_at`, `updated_at` (GORM managed)

### Go Code
- **Packages**: lowercase, short (e.g., `models`, `controllers`, `middleware`)
- **Models**: PascalCase, singular (`User`, `Order`, `Message`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase

### API
- **Base path**: `/api/v1`
- **Resources**: plural, lowercase (`/orders`, `/users`, `/designs`)
- **IDs in path**: `/:id` (e.g., `/api/v1/orders/:id`)

## API Response Format

All endpoints use consistent JSON structure:

**Success:**
```json
{
  "success": true,
  "data": { /* resource or array */ }
}
```

**Error:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": { /* optional field errors */ }
  }
}
```

## Development Guidelines

### Error Handling
- Return appropriate HTTP status codes (see `requirements/13-api-design.md`)
- Use centralized error handling middleware
- Provide helpful error messages for debugging
- Never expose sensitive info in errors (stack traces, DB details)

### Security Best Practices
- **Never skip JWT validation** on protected endpoints
- **Always check authorization**: User can only access their own resources (unless technician/admin)
- **Validate all inputs**: Use Gin binding/validation
- **Prevent SQL injection**: Use GORM parameterized queries (never raw SQL with user input)
- **Rate limiting**: Prevent abuse (see iteration 20)

### Testing Approach
After each iteration:
1. Manual API testing with Postman/curl
2. Automated unit, integration, and acceptance tests
3. Test edge cases (invalid input, missing auth, wrong roles)
4. Regression test previous features

## Common Commands

```bash
# Run the server
go run main.go

# Database access
psql -U postgres -d kendalls_nails

# Test endpoints
curl http://localhost:8080/api/v1/health
curl -H "Authorization: Bearer {token}" http://localhost:8080/api/v1/orders
```

## Project Structure (Once Implemented)

```
kendalls-nails-api/
├── main.go                 # Application entry point
├── config/                 # Configuration and env loading
├── models/                 # GORM models (User, Order, Message, etc.)
├── controllers/            # Request handlers (OrderController, UserController)
├── middleware/             # Auth, logging, error handling, rate limiting
├── routes/                 # Route definitions
├── services/               # Business logic (S3Service, AuthService)
├── utils/                  # Helper functions
├── .env                    # Local environment variables (git ignored)
├── .env.example            # Template for environment variables
└── requirements/           # Requirements documentation (already exists)
```

## Environment Variables

Required for local development (add to `.env`):

```
DATABASE_URL=postgresql://user:password@localhost:5432/kendalls_nails
PORT=8080
AUTH0_DOMAIN=your-tenant.auth0.com
AUTH0_AUDIENCE=your-api-identifier
JWT_SECRET=your-secret-key
AWS_REGION=us-east-1
AWS_S3_BUCKET=kendalls-nails-uploads
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
LOG_LEVEL=debug
```

## Key Design Decisions

1. **Go over Node.js**: Better performance, type safety, simpler concurrency
2. **Auth0 over custom auth**: Saves development time, enterprise-grade security
3. **PostgreSQL over MongoDB**: Relational data with clear relationships
4. **S3 over local storage**: Scalable, reliable, Heroku-friendly
5. **Heroku over AWS/GCP**: Simpler deployment, lower ops overhead for MVP

## When Working on This Project

1. **Always check requirements first**: See `requirements/` folder for detailed specs
2. **Follow the implementation plan**: Use `IMPLEMENTATION_PLAN.md` iteration order
3. **Keep responses consistent**: Follow API design patterns in `requirements/13-api-design.md`
4. **Index foreign keys**: Critical for query performance
5. **Test role-based access**: Every protected endpoint should verify user permissions
6. **Validate state transitions**: Especially for order status workflow
7. **Handle file uploads carefully**: Validate format, size, and storage location

## Open Questions & Future Enhancements

See `requirements/08-business-rules.md` for:
- Payment integration details (Stripe integration planned)
- Real-time notifications (WebSockets consideration)
- Email notification triggers
- Admin dashboard features

## Getting Help

- Requirements questions: Check `/requirements/*.md` files
- Implementation questions: Reference `IMPLEMENTATION_PLAN.md` iteration notes
- API design questions: See `requirements/13-api-design.md`
- Database questions: See `requirements/10-database.md`
