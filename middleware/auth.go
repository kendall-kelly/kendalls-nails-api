package middleware

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
)

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// HasScope checks whether our claims have a specific scope.
func (c CustomClaims) HasScope(expectedScope string) bool {
	result := strings.Split(c.Scope, " ")
	for i := range result {
		if result[i] == expectedScope {
			return true
		}
	}

	return false
}

// EnsureValidToken is a middleware that will check the validity of our JWT.
func EnsureValidToken(cfg *config.Config) gin.HandlerFunc {
	issuerURL, err := url.Parse("https://" + cfg.Auth0Domain + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{cfg.Auth0Audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Encountered error while validating JWT: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, writeErr := w.Write([]byte(`{"success":false,"error":{"code":"INVALID_TOKEN","message":"Failed to validate JWT."}}`)); writeErr != nil {
			log.Printf("Failed to write error response: %v", writeErr)
		}
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	return func(c *gin.Context) {
		var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			// Store the validated claims in Gin context
			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

			// Extract user_id from sub claim
			userID := token.RegisteredClaims.Subject
			c.Set("user_id", userID)
			c.Set("validated_claims", token)

			c.Next()
		}

		// Use the JWT middleware to check the token
		middleware.CheckJWT(handler).ServeHTTP(c.Writer, c.Request)
	}
}

// GetUserID extracts the user ID from the Gin context
func GetUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", &AuthError{Code: "MISSING_USER_ID", Message: "User ID not found in context"}
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", &AuthError{Code: "INVALID_USER_ID", Message: "User ID is not a string"}
	}

	return userIDStr, nil
}

// GetClaims extracts the validated JWT claims from the Gin context
func GetClaims(c *gin.Context) (*validator.ValidatedClaims, error) {
	claims, exists := c.Get("validated_claims")
	if !exists {
		return nil, &AuthError{Code: "MISSING_CLAIMS", Message: "Claims not found in context"}
	}

	validatedClaims, ok := claims.(*validator.ValidatedClaims)
	if !ok {
		return nil, &AuthError{Code: "INVALID_CLAIMS", Message: "Claims are not in the expected format"}
	}

	return validatedClaims, nil
}

// RequireScope is a middleware that checks if the token has a specific scope
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaims(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_CLAIMS",
					"message": "Could not retrieve token claims",
				},
			})
			c.Abort()
			return
		}

		customClaims := claims.CustomClaims.(*CustomClaims)
		if !customClaims.HasScope(scope) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_SCOPE",
					"message": "Insufficient permissions to access this resource",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthError represents an authentication error
type AuthError struct {
	Code    string
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
