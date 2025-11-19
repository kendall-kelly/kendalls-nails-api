# Testing Auth0 Integration (Iteration 4)

## Overview
This guide explains how to test the Auth0 JWT authentication integration implemented in Iteration 4.

## Prerequisites

1. **Auth0 Account Setup**
   - Create a free Auth0 account at https://auth0.com
   - Create a new API in the Auth0 Dashboard
   - Note your Auth0 Domain and API Identifier (Audience)

2. **Environment Configuration**
   - Copy `.env.example` to `.env.development`
   - Update with your Auth0 credentials:
     ```
     AUTH0_DOMAIN=your-tenant.auth0.com
     AUTH0_AUDIENCE=your-api-identifier
     ```

## Endpoints

### 1. Public Endpoints (No Authentication Required)
- `GET /api/v1/health` - Health check
- `GET /api/v1/database/status` - Database status

### 2. Protected Endpoint (Authentication Required)
- `GET /api/v1/protected` - Requires valid JWT token

## Testing Steps

### Test 1: Access Protected Endpoint Without Token (Should Fail)

```bash
curl http://localhost:8080/api/v1/protected
```

**Expected Response:** 401 Unauthorized
```json
{
  "success": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "Failed to validate JWT."
  }
}
```

### Test 2: Access Protected Endpoint With Invalid Token (Should Fail)

```bash
curl -H "Authorization: Bearer invalid-token" \
  http://localhost:8080/api/v1/protected
```

**Expected Response:** 401 Unauthorized

### Test 3: Get a Valid Token from Auth0

#### Option A: Using Auth0 Dashboard (Quick Test)
1. Go to your Auth0 Dashboard
2. Navigate to Applications > APIs
3. Select your API
4. Click on the "Test" tab
5. Copy the access token from the response

#### Option B: Using Postman
1. Create a new request in Postman
2. Go to Authorization tab
3. Select "OAuth 2.0"
4. Configure with your Auth0 settings
5. Get new access token

#### Option C: Using curl (for testing applications)
```bash
curl --request POST \
  --url https://YOUR_DOMAIN.auth0.com/oauth/token \
  --header 'content-type: application/json' \
  --data '{
    "client_id":"YOUR_CLIENT_ID",
    "client_secret":"YOUR_CLIENT_SECRET",
    "audience":"YOUR_API_IDENTIFIER",
    "grant_type":"client_credentials"
  }'
```

### Test 4: Access Protected Endpoint With Valid Token (Should Succeed)

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8080/api/v1/protected
```

**Expected Response:** 200 OK
```json
{
  "success": true,
  "message": "You have accessed a protected endpoint",
  "data": {
    "user_id": "auth0|...",
    "issuer": "https://your-tenant.auth0.com/",
    "subject": "auth0|..."
  }
}
```

## Common Issues

### Issue: "Failed to parse the issuer url"
- **Cause:** AUTH0_DOMAIN not set in environment
- **Solution:** Ensure `.env.development` exists with correct AUTH0_DOMAIN

### Issue: "invalid audience"
- **Cause:** Token was not issued for this API
- **Solution:** Ensure AUTH0_AUDIENCE matches the API Identifier in Auth0

### Issue: "token expired"
- **Cause:** Access token has expired
- **Solution:** Generate a new token

### Issue: "unable to find appropriate key"
- **Cause:** Token signing key doesn't match Auth0's JWKS
- **Solution:** Ensure token is from the correct Auth0 tenant

## What You've Learned

After completing this iteration, you now understand:

- ✅ How JWT authentication works with Auth0
- ✅ Middleware pattern in Gin for protecting routes
- ✅ Extracting and validating JWT claims
- ✅ HTTP Authorization headers and Bearer tokens
- ✅ The difference between public and protected endpoints
- ✅ How to handle authentication errors gracefully

## Next Steps

Iteration 4 is complete! You can now:
1. Proceed to Iteration 5: User Management & Profile
2. Implement role-based access control using JWT claims
3. Create user registration webhooks from Auth0
