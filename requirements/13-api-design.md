# API Design

- **API Style**: RESTful API (Representational State Transfer)
- **Data Format**: JSON for all request and response bodies
- **Base URL**: `/api/v1` (versioning for future compatibility)
- **Content-Type**: `application/json` for request/response headers

## HTTP Methods
Follow standard REST conventions:
- **GET** - Retrieve resource(s) (read-only, no side effects)
- **POST** - Create new resource
- **PUT** - Update existing resource (full replacement)
- **PATCH** - Partial update of resource (optional, use PUT if not needed)
- **DELETE** - Remove resource

## HTTP Status Codes
Use standard HTTP status codes consistently:

**Success Codes:**
- **200 OK** - Successful GET, PUT, or DELETE
- **201 Created** - Successful POST (resource created)
- **204 No Content** - Successful DELETE with no response body

**Client Error Codes:**
- **400 Bad Request** - Invalid request format, validation error
- **401 Unauthorized** - Missing or invalid authentication token
- **403 Forbidden** - Valid token but insufficient permissions for action
- **404 Not Found** - Requested resource does not exist
- **409 Conflict** - Request conflicts with current state (e.g., duplicate email)
- **413 Payload Too Large** - File upload exceeds size limit
- **422 Unprocessable Entity** - Valid format but business rule violation

**Server Error Codes:**
- **500 Internal Server Error** - Unexpected server error
- **503 Service Unavailable** - Server temporarily unavailable

## Response Format
All responses follow a consistent JSON structure:

**Success Response:**
```json
{
  "success": true,
  "data": { /* resource data or array */ },
  "message": "Optional success message"
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": { /* optional field-specific errors */ }
  }
}
```

**Validation Error Example:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": {
      "email": "Email is required",
      "quantity": "Quantity must be at least 1"
    }
  }
}
```

## Pagination
For list endpoints (e.g., `GET /orders`, `GET /designs`):

**Request Parameters:**
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)
- `sort` - Sort field (e.g., `created_at`, `-created_at` for descending)

**Response Format:**
```json
{
  "success": true,
  "data": [ /* array of resources */ ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "totalPages": 8,
    "hasNext": true,
    "hasPrev": false
  }
}
```

## Filtering and Searching
Support query parameters for filtering:
- `status` - Filter by status (e.g., `?status=submitted`)
- `search` - Text search in relevant fields
- `from_date`, `to_date` - Date range filtering

Example: `GET /orders?status=in_production&from_date=2025-01-01`

## Authentication Header
All protected endpoints require JWT token:
```
Authorization: Bearer {jwt_token}
```

## File Upload Endpoints
Use `multipart/form-data` for endpoints accepting file uploads:
- Content-Type: `multipart/form-data`
- File field name: `image`
- Other fields: Included as form fields

## CORS (Cross-Origin Resource Sharing)
- Enable CORS for frontend applications
- Allowed origins: Configured via environment variable
- Allowed methods: GET, POST, PUT, DELETE, OPTIONS
- Allowed headers: Authorization, Content-Type
- Credentials: true (for cookie-based auth if needed)

## Rate Limiting
- Implement rate limiting to prevent abuse
- Suggested limits:
  - Authenticated users: 100 requests/minute
  - Unauthenticated users: 20 requests/minute
  - File uploads: 10 requests/minute
- Return 429 Too Many Requests when limit exceeded

## Versioning Strategy
- Current version: v1 (in URL: `/api/v1/...`)
- Future versions: Increment version number in URL path
- Maintain backward compatibility within same major version
- Document breaking changes when introducing new versions
