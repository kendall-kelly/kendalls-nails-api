# Custom Nails API - Implementation Plan

## Overview

This implementation plan breaks down the Custom Nails API project into small, iterative steps that build on each other. Each iteration delivers working, testable functionality and can be explained to a junior engineer in a single session.

## Guiding Principles

1. **Build incrementally** - Each iteration adds one piece of functionality
2. **Test as you go** - Every iteration should be manually testable
3. **Foundation first** - Start with infrastructure, then core features
4. **Deliver value early** - Focus on the critical order flow first
5. **Keep it simple** - Avoid over-engineering; add complexity only when needed

---

## Phase 1: Foundation (Iterations 1-4)

Goal: Set up the basic infrastructure and prove the technology stack works.

### Iteration 1: Project Setup & Hello World API
**Duration**: 1-2 hours
**Complexity**: Beginner

**What you'll build:**
- Initialize Go module
- Set up Gin web server
- Create a simple "Hello World" endpoint
- Add basic logging

**Deliverable:**
```bash
GET /api/v1/health
Response: {"success": true, "message": "Custom Nails API is running"}
```

**How to test:**
- Run the server with `go run main.go`
- Visit `http://localhost:8080/api/v1/health` in browser or use curl
- Should see the success message

**What a junior engineer learns:**
- How to initialize a Go project
- Basic Gin routing
- JSON response formatting
- Running a Go web server

---

### Iteration 2: Database Setup & First Model
**Duration**: 2-3 hours
**Complexity**: Beginner

**What you'll build:**
- Install PostgreSQL locally (or use Docker)
- Configure GORM connection
- Create a simple `User` model with basic fields (ID, Email, Role, CreatedAt)
- Auto-migrate the database
- Add a test endpoint to check database connectivity

**Deliverable:**
```bash
GET /api/v1/database/status
Response: {"success": true, "message": "Database connected", "tables": ["users"]}
```

**How to test:**
- Start PostgreSQL
- Run the server
- Check the `/database/status` endpoint
- Use a database client (psql, TablePlus, pgAdmin) to verify the `users` table was created

**What a junior engineer learns:**
- How to connect Go to PostgreSQL
- GORM basics (models, auto-migration)
- Database connection strings and environment variables
- How to inspect database tables

---

### Iteration 3: Configuration & Environment Variables
**Duration**: 1-2 hours
**Complexity**: Beginner

**What you'll build:**
- Create a config package to load environment variables
- Set up `.env` file for local development
- Add configuration for:
  - Database URL
  - Server port
  - JWT secret (for later)
  - Log level
- Add `.env.example` template

**Deliverable:**
- Server reads configuration from environment variables
- `.env.example` file for other developers
- Config validation (fail fast if required vars are missing)

**How to test:**
- Create `.env` file with database credentials
- Start server and verify it connects using env vars
- Try removing a required var and confirm server refuses to start

**What a junior engineer learns:**
- Environment variable best practices
- Configuration management
- Separating config from code
- Never committing secrets to git

---

### Iteration 4: Auth0 Integration & JWT Middleware
**Duration**: 3-4 hours
**Complexity**: Intermediate

**What you'll build:**
- Set up Auth0 account and application
- Create JWT validation middleware
- Protect an endpoint with authentication
- Extract user info from JWT token

**Deliverable:**
```bash
GET /api/v1/protected
Header: Authorization: Bearer {token}
Response: {"success": true, "user_id": "auth0|123...", "role": "customer"}
```

**How to test:**
- Create a test user in Auth0 dashboard
- Get a JWT token (use Auth0 test feature or Postman)
- Call `/protected` endpoint with valid token (should work)
- Call `/protected` endpoint without token (should get 401)
- Call `/protected` endpoint with invalid token (should get 401)

**What a junior engineer learns:**
- How JWT authentication works
- Middleware pattern in Gin
- Auth0 integration
- Extracting claims from JWT tokens
- HTTP authentication headers

---

## Phase 2: Core Order Flow (Iterations 5-9)

Goal: Implement the essential order submission and review workflow.

### Iteration 5: User Management & Profile
**Duration**: 2-3 hours
**Complexity**: Beginner-Intermediate

**What you'll build:**
- Enhance `User` model with full fields (Name, Email, Role, Auth0ID)
- Implement user creation endpoint (called when Auth0 user registers)
- Implement "Get my profile" endpoint
- Implement "Update my profile" endpoint

**Deliverable:**
```bash
POST /api/v1/users (create user from Auth0 webhook/manual call)
GET /api/v1/users/me (get current user profile)
PUT /api/v1/users/me (update current user profile)
```

**How to test:**
- Create a user via POST endpoint
- Get user profile using JWT token
- Update profile and verify changes
- Query database to confirm data is stored correctly

**What a junior engineer learns:**
- CRUD operations in Go/GORM
- Request body validation
- Updating records in database
- Extracting user ID from JWT in middleware

---

### Iteration 6: Order Model & Create Order
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Create `Order` model (without file upload yet)
- Fields: Description, Quantity, Status, Price, CustomerID, TechnicianID
- Implement "Create order" endpoint (customers only)
- Default status = "submitted"

**Deliverable:**
```bash
POST /api/v1/orders
Body: {"description": "Pink nails with glitter", "quantity": 2}
Response: {"success": true, "data": {"id": 1, "status": "submitted", ...}}
```

**How to test:**
- Create order with customer JWT token (should work)
- Try to create order with technician token (should fail with 403)
- Try to create order without auth (should fail with 401)
- Verify order appears in database with correct customer_id
- Verify status is set to "submitted"

**What a junior engineer learns:**
- Creating related records (Order belongs to User)
- Role-based access control (RBAC)
- Default values in database models
- Validation (quantity must be positive)

---

### Iteration 7: List & Get Orders (with Filtering)
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Implement "List orders" endpoint with role-based filtering:
  - Customers see only their orders
  - Technicians see orders assigned to them + unassigned orders
- Implement "Get single order" endpoint
- Add basic pagination (limit/offset)

**Deliverable:**
```bash
GET /api/v1/orders (list orders based on role)
GET /api/v1/orders/:id (get single order)
```

**How to test:**
- Create multiple orders from different customer accounts
- As customer: verify you see only your own orders
- As technician: verify you see assigned orders (and unassigned ones)
- Try to access another customer's order by ID (should fail with 403)
- Test pagination with ?page=1&limit=10

**What a junior engineer learns:**
- Query filtering based on user role
- Authorization checks (can user access this resource?)
- Pagination implementation
- GORM query builder

---

### Iteration 8: Order Review (Accept/Reject)
**Duration**: 3-4 hours
**Complexity**: Intermediate

**What you'll build:**
- Implement "Review order" endpoint (technicians only)
- Accept flow: Set status to "accepted", set price
- Reject flow: Set status to "rejected", require feedback message
- Order assignment: When reviewing, assign order to technician

**Deliverable:**
```bash
PUT /api/v1/orders/:id/review
Body: {
  "action": "accept",
  "price": 45.00
}
OR
Body: {
  "action": "reject",
  "feedback": "Design is too complex for current materials"
}
```

**How to test:**
- Create an order as customer
- Accept it as technician with a price
- Verify order status changed to "accepted" and price is set
- Create another order and reject it with feedback
- Verify rejected order has feedback
- Try to review an already-reviewed order (should fail with 422)
- Try to accept without price (should fail validation)

**What a junior engineer learns:**
- State transitions and business logic
- Conditional validation (price required for accept, feedback for reject)
- Update operations with complex logic
- Status workflow implementation

---

### Iteration 9: Order Status Updates
**Duration**: 2-3 hours
**Complexity**: Beginner-Intermediate

**What you'll build:**
- Implement "Update order status" endpoint (technicians only)
- Valid transitions: accepted → in_production → shipped → delivered
- Validate status transitions (can't go from shipped back to in_production)

**Deliverable:**
```bash
PUT /api/v1/orders/:id/status
Body: {"status": "in_production"}
```

**How to test:**
- Create and accept an order
- Progress order through: accepted → in_production → shipped → delivered
- Try invalid transition (e.g., submitted → shipped) and verify it fails
- Try to update status as customer (should fail with 403)

**What a junior engineer learns:**
- State machine patterns
- Business rule validation
- Preventing invalid state transitions
- Status workflow enforcement

---

## Phase 3: File Upload (Iterations 10-11)

Goal: Add image upload capability for order designs.

### Iteration 10: Local File Upload (No S3 Yet)
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Update "Create order" endpoint to accept multipart/form-data
- Save uploaded files to local `/uploads` directory
- Validate file format (PNG only)
- Validate file size (max 10MB)
- Store file path in Order model
- Add endpoint to serve uploaded files

**Deliverable:**
```bash
POST /api/v1/orders
Content-Type: multipart/form-data
Fields: {image: file, description: "...", quantity: 2}

GET /api/v1/uploads/:filename (serve the uploaded file)
```

**How to test:**
- Upload order with PNG image (should work)
- Try to upload JPEG (should fail)
- Try to upload file larger than 10MB (should fail)
- Get order details and verify image URL is present
- Access image URL and verify file is served

**What a junior engineer learns:**
- Multipart form data handling
- File upload in Go
- File validation
- Serving static files
- Local file storage

---

### Iteration 11: AWS S3 Integration
**Duration**: 3-4 hours
**Complexity**: Intermediate-Advanced

**What you'll build:**
- Install AWS SDK for Go
- Configure S3 bucket (or use LocalStack for local dev)
- Replace local file upload with S3 upload
- Generate presigned URLs for private images
- Update image serving logic

**Deliverable:**
- Files uploaded to S3 instead of local disk
- Order model stores S3 key
- API returns presigned URLs for accessing images

**How to test:**
- Upload order with image
- Verify file appears in S3 bucket
- Get order details and verify presigned URL is returned
- Access presigned URL and verify image loads
- Wait for URL to expire (if testing expiration)

**What a junior engineer learns:**
- AWS S3 basics
- Cloud storage integration
- Presigned URLs
- Environment-specific configurations (local vs cloud)

---

## Phase 4: Communication (Iterations 12-13)

Goal: Enable messaging between customers and technicians.

### Iteration 12: Order Messages
**Duration**: 2-3 hours
**Complexity**: Beginner-Intermediate

**What you'll build:**
- Create `Message` model (OrderID, SenderID, Text, CreatedAt)
- Implement "Send message" endpoint
- Implement "List messages" endpoint for an order

**Deliverable:**
```bash
POST /api/v1/orders/:id/messages
Body: {"text": "Can you make the glitter more subtle?"}

GET /api/v1/orders/:id/messages
Response: {"success": true, "data": [messages...]}
```

**How to test:**
- Create an order
- Send message as customer
- Send reply as technician
- List messages and verify conversation
- Try to send message to someone else's order (should fail)

**What a junior engineer learns:**
- Creating related records (Messages belong to Order and User)
- Conversation threading
- Authorization (can only message on your own orders)

---

### Iteration 13: Message Notifications (Simple Version)
**Duration**: 1-2 hours
**Complexity**: Beginner

**What you'll build:**
- Add `UnreadCount` to Order model
- Increment unread count when message is sent
- Add endpoint to mark messages as read

**Deliverable:**
```bash
GET /api/v1/orders/:id
Response includes: {"unread_messages": 3}

PUT /api/v1/orders/:id/messages/read
(marks all messages as read)
```

**How to test:**
- Send message and verify unread count increases
- Mark as read and verify count resets
- Test from both customer and technician perspective

**What a junior engineer learns:**
- Denormalized counters for performance
- Updating related records
- Simple notification systems

---

## Phase 5: Design Gallery (Iterations 14-16)

Goal: Allow customers to share designs publicly and interact with them.

### Iteration 14: Design Gallery Model & Visibility Toggle
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Create `Design` model (OrderID, IsPublic, CreatedAt)
- Automatically create Design record when order is created
- Default IsPublic = false
- Implement "Toggle visibility" endpoint

**Deliverable:**
```bash
PUT /api/v1/designs/:id/visibility
Body: {"is_public": true}
```

**How to test:**
- Create order (design is private by default)
- Toggle to public
- Toggle back to private
- Try to toggle someone else's design (should fail)

**What a junior engineer learns:**
- One-to-one relationships (Order has one Design)
- Automatic record creation (hooks/callbacks)
- Toggle boolean fields

---

### Iteration 15: Browse Public Designs
**Duration**: 2-3 hours
**Complexity**: Beginner-Intermediate

**What you'll build:**
- Implement "List public designs" endpoint (no auth required)
- Include order image, description, and owner info
- Add pagination
- Implement "Get single design" endpoint

**Deliverable:**
```bash
GET /api/v1/designs (list all public designs)
GET /api/v1/designs/:id (get single design)
```

**How to test:**
- Create several orders and make some public
- Browse designs without authentication (should work)
- Verify only public designs appear
- Verify private designs don't show up

**What a junior engineer learns:**
- Public vs authenticated endpoints
- JOIN queries (Design with Order and User)
- Filtering by boolean flags

---

### Iteration 16: Design Comments
**Duration**: 2-3 hours
**Complexity**: Beginner-Intermediate

**What you'll build:**
- Create `Comment` model (DesignID, UserID, Text, CreatedAt)
- Implement "Add comment" endpoint (requires auth)
- Implement "List comments" endpoint (no auth required for public designs)

**Deliverable:**
```bash
POST /api/v1/designs/:id/comments
Body: {"text": "Love the colors!"}

GET /api/v1/designs/:id/comments
```

**How to test:**
- Create public design
- Add comment as customer
- List comments (with and without auth)
- Try to comment on private design (should fail)
- Try to comment without auth (should fail)

**What a junior engineer learns:**
- Creating related records
- Nested resources (comments belong to designs)
- Mixed auth requirements (read public, write needs auth)

---

## Phase 6: Advanced Features (Iterations 17-18)

Goal: Add remaining features to complete MVP.

### Iteration 17: Reorder Functionality
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Implement "Reorder" endpoint
- Copy design from existing order
- Create new order with copied design (status = submitted)
- Link to original order for tracking

**Deliverable:**
```bash
POST /api/v1/orders/:id/reorder
Body: {"quantity": 3}
Response: {"success": true, "data": {new order...}}
```

**How to test:**
- Create and complete an order
- Reorder it with different quantity
- Verify new order is created with same design
- Verify new order goes through review process

**What a junior engineer learns:**
- Copying records
- Linking related records
- Business logic for reuse

---

### Iteration 18: Technician Invitation System
**Duration**: 3-4 hours
**Complexity**: Advanced

**What you'll build:**
- Implement "Invite technician" endpoint (admin/existing tech only)
- Call Auth0 Management API to create user
- Send invitation email via Auth0
- Restrict who can invite (role check)

**Deliverable:**
```bash
POST /api/v1/auth/technicians/invite
Body: {"email": "tech@example.com", "name": "Jane Doe"}
```

**How to test:**
- Invite a technician as admin
- Receive invitation email
- Complete signup flow
- Verify new technician can log in and review orders
- Try to invite as customer (should fail)

**What a junior engineer learns:**
- External API integration (Auth0 Management API)
- Role-based permissions
- User provisioning
- Email workflows

---

## Phase 7: Polish & Deployment (Iterations 19-21)

Goal: Prepare for production deployment.

### Iteration 19: Error Handling & Validation
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Centralized error handling middleware
- Consistent error response format
- Comprehensive request validation
- Meaningful error messages

**Deliverable:**
- All errors return consistent JSON format
- Validation errors include field-specific details
- 400/422 errors have helpful messages

**How to test:**
- Test invalid requests across all endpoints
- Verify error responses match API design spec
- Verify helpful error messages

**What a junior engineer learns:**
- Error handling patterns
- Input validation
- User-friendly error messages

---

### Iteration 20: Rate Limiting & Security
**Duration**: 2-3 hours
**Complexity**: Intermediate

**What you'll build:**
- Add rate limiting middleware
- Configure CORS properly
- Add security headers (Helmet equivalent)
- Add request logging

**Deliverable:**
- Rate limits enforced (429 errors after threshold)
- CORS configured for frontend
- Security headers in responses

**How to test:**
- Make rapid requests and verify rate limiting
- Test CORS from frontend domain
- Inspect response headers for security headers

**What a junior engineer learns:**
- API security basics
- Rate limiting
- CORS configuration
- Production-ready middleware

---

### Iteration 21: Heroku Deployment
**Duration**: 3-4 hours
**Complexity**: Intermediate-Advanced

**What you'll build:**
- Create Heroku app
- Configure Heroku Postgres
- Set environment variables in Heroku
- Create Procfile
- Deploy application

**Deliverable:**
- API running on Heroku
- Database hosted on Heroku Postgres
- Environment variables configured
- Health check endpoint accessible

**How to test:**
- Deploy to Heroku
- Run migrations
- Test all endpoints on production URL
- Verify database connections work
- Monitor logs

**What a junior engineer learns:**
- Platform-as-a-Service (PaaS) deployment
- Production environment configuration
- Database migrations in production
- Monitoring and logging

---

## Testing Strategy

After each iteration:
1. **Manual testing** - Test the new endpoint with Postman/curl
2. **Database inspection** - Verify data is stored correctly
3. **Regression testing** - Ensure previous features still work
4. **Edge cases** - Test invalid inputs, missing auth, wrong roles

---

## Helpful Commands Cheat Sheet

```bash
# Run the server
go run main.go

# Run with auto-reload (install air first)
air

# Database operations
psql -U postgres -d kendalls_nails

# Test endpoints
curl http://localhost:8080/api/v1/health
curl -H "Authorization: Bearer {token}" http://localhost:8080/api/v1/protected

# Check Git status
git status
git add .
git commit -m "Iteration X: Feature name"
```

---

## Success Criteria

At the end of this plan, you will have:
- ✅ Fully functional REST API with all core features
- ✅ Authentication and authorization working
- ✅ File upload to S3
- ✅ Order workflow (submit → review → production → delivery)
- ✅ Messaging system
- ✅ Public design gallery with comments
- ✅ Deployed to Heroku and accessible via URL
- ✅ Secure, validated, and production-ready

---

## Next Steps After MVP

Once the MVP is complete, consider:
- Real-time notifications (WebSockets or Pusher)
- Email notifications
- Payment integration (Stripe)
- Admin dashboard
- Analytics and reporting
- Mobile app support
- Performance optimization
- Comprehensive automated testing
