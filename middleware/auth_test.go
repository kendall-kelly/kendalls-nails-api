package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCustomClaims_HasScope(t *testing.T) {
	tests := []struct {
		name          string
		scope         string
		expectedScope string
		want          bool
	}{
		{
			name:          "has exact scope",
			scope:         "read:messages",
			expectedScope: "read:messages",
			want:          true,
		},
		{
			name:          "has scope in multiple scopes",
			scope:         "read:messages write:messages delete:messages",
			expectedScope: "write:messages",
			want:          true,
		},
		{
			name:          "does not have scope",
			scope:         "read:messages",
			expectedScope: "write:messages",
			want:          false,
		},
		{
			name:          "empty scope",
			scope:         "",
			expectedScope: "read:messages",
			want:          false,
		},
		{
			name:          "partial match should not work",
			scope:         "read:messages",
			expectedScope: "read",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := CustomClaims{Scope: tt.scope}
			got := claims.HasScope(tt.expectedScope)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupFunc func(*gin.Context)
		wantID    string
		wantErr   bool
	}{
		{
			name: "successfully extracts user ID",
			setupFunc: func(c *gin.Context) {
				c.Set("user_id", "auth0|123456")
			},
			wantID:  "auth0|123456",
			wantErr: false,
		},
		{
			name: "user ID not found in context",
			setupFunc: func(c *gin.Context) {
				// Don't set user_id
			},
			wantID:  "",
			wantErr: true,
		},
		{
			name: "user ID is not a string",
			setupFunc: func(c *gin.Context) {
				c.Set("user_id", 12345) // Set as int instead of string
			},
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupFunc(c)

			gotID, err := GetUserID(c)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, gotID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}
		})
	}
}

func TestGetClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupFunc func(*gin.Context)
		wantErr   bool
	}{
		{
			name: "successfully extracts claims",
			setupFunc: func(c *gin.Context) {
				claims := &validator.ValidatedClaims{
					RegisteredClaims: validator.RegisteredClaims{
						Issuer:  "https://test.auth0.com/",
						Subject: "auth0|123456",
					},
					CustomClaims: &CustomClaims{
						Scope: "read:messages",
					},
				}
				c.Set("validated_claims", claims)
			},
			wantErr: false,
		},
		{
			name: "claims not found in context",
			setupFunc: func(c *gin.Context) {
				// Don't set validated_claims
			},
			wantErr: true,
		},
		{
			name: "claims are not the expected type",
			setupFunc: func(c *gin.Context) {
				c.Set("validated_claims", "invalid")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupFunc(c)

			claims, err := GetClaims(c)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestRequireScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requiredScope  string
		setupFunc      func(*gin.Context)
		wantStatusCode int
		wantAborted    bool
	}{
		{
			name:          "has required scope",
			requiredScope: "read:messages",
			setupFunc: func(c *gin.Context) {
				claims := &validator.ValidatedClaims{
					CustomClaims: &CustomClaims{
						Scope: "read:messages write:messages",
					},
				}
				c.Set("validated_claims", claims)
			},
			wantStatusCode: 0, // Should not write status, continues to next handler
			wantAborted:    false,
		},
		{
			name:          "missing required scope",
			requiredScope: "delete:messages",
			setupFunc: func(c *gin.Context) {
				claims := &validator.ValidatedClaims{
					CustomClaims: &CustomClaims{
						Scope: "read:messages write:messages",
					},
				}
				c.Set("validated_claims", claims)
			},
			wantStatusCode: http.StatusForbidden,
			wantAborted:    true,
		},
		{
			name:          "claims not in context",
			requiredScope: "read:messages",
			setupFunc: func(c *gin.Context) {
				// Don't set validated_claims
			},
			wantStatusCode: http.StatusUnauthorized,
			wantAborted:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			tt.setupFunc(c)

			handler := RequireScope(tt.requiredScope)
			handler(c)

			if tt.wantAborted {
				assert.True(t, c.IsAborted())
				assert.Equal(t, tt.wantStatusCode, w.Code)
			} else {
				assert.False(t, c.IsAborted())
			}
		})
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{
		Code:    "TEST_ERROR",
		Message: "This is a test error",
	}

	assert.Equal(t, "This is a test error", err.Error())
}
