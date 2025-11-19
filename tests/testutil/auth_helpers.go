package testutil

import (
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
)

// MockValidatedClaims creates a mock ValidatedClaims for testing
func MockValidatedClaims(subject, issuer string, scopes []string) *validator.ValidatedClaims {
	scopeString := ""
	if len(scopes) > 0 {
		for i, scope := range scopes {
			if i > 0 {
				scopeString += " "
			}
			scopeString += scope
		}
	}

	return &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Issuer:  issuer,
			Subject: subject,
		},
		CustomClaims: &middleware.CustomClaims{
			Scope: scopeString,
		},
	}
}

// SetMockAuthContext sets up a mock authenticated context for testing
func SetMockAuthContext(c *gin.Context, userID string, issuer string, scopes []string) {
	claims := MockValidatedClaims(userID, issuer, scopes)
	c.Set("user_id", userID)
	c.Set("validated_claims", claims)
}

// CreateTestContext creates a test Gin context
func CreateTestContext() (*gin.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	c, engine := gin.CreateTestContext(nil)
	return c, engine
}
