# Authentication

- **Authentication Provider**: Auth0 (https://auth0.com)
  - Managed authentication service (delegates user credential storage and verification)
  - JWT token-based authentication
  - Built-in security features (password hashing, rate limiting, breach detection)
  - Social login support available (Google, Facebook, etc.) - optional for future
- **User Roles**: Two roles managed via Auth0 custom claims
  - `customer` - Self-registered users who place orders
  - `nail_technician` - Invitation-only users who review and fulfill orders
- **Auth0 Configuration**:
  - **Application Type**: Regular Web Application (Backend API)
  - **Token Type**: JWT (JSON Web Tokens)
  - **Token Location**: Authorization header (`Bearer {token}`)
  - **Custom Claims**: User role stored in JWT payload (e.g., `"role": "customer"`)
  - **User Metadata**: Additional profile data stored in Auth0 user profile
- **Go Integration**:
  - **Library**: go-jwt-middleware or auth0/go-jwt-middleware
  - **Validation**: Verify JWT signature using Auth0 public key (JWKS endpoint)
  - **Middleware**: Gin middleware to validate tokens on protected routes
- **Authentication Flows**:
  - **Customer Registration**:
    1. Frontend calls Auth0 signup API
    2. Auth0 creates user account with `customer` role
    3. Auth0 returns JWT token
    4. Backend receives token, validates, and creates user record in local database
  - **Technician Registration** (Invitation-only):
    1. Admin/existing technician sends invitation via backend endpoint
    2. Backend calls Auth0 Management API to create user with `nail_technician` role
    3. Auth0 sends invitation email with password setup link
    4. New technician sets password and receives JWT token
    5. Backend creates technician record in local database
  - **Login**:
    1. User authenticates via Auth0 (frontend handles Auth0 login UI)
    2. Auth0 returns JWT token containing user ID and role
    3. Frontend includes token in Authorization header for all API requests
    4. Backend validates token and extracts user ID/role for authorization
  - **Logout**:
    - Client-side: Remove token from storage
    - Server-side: Stateless (tokens expire naturally, no server-side session)
- **Role-Based Access Control (RBAC)**:
  - Middleware checks JWT role claim before allowing access to endpoints
  - Examples:
    - Only `customer` can submit orders
    - Only `nail_technician` can review/accept/reject orders
    - Both roles can view their own orders
- **Token Management**:
  - **Access Token Expiration**: 24 hours (configurable in Auth0)
  - **Refresh Tokens**: Optional - can be enabled for longer sessions
  - **Token Validation**: On every API request via middleware
- **Security Considerations**:
  - Auth0 handles: Password hashing, credential storage, token signing
  - Backend handles: Token validation, role-based authorization, business logic
  - Never store passwords in local database (Auth0 manages credentials)
  - Always validate token signature and expiration
  - Check role claim matches required permission for each endpoint
