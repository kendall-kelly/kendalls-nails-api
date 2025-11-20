package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kendall-kelly/kendalls-nails-api/config"
)

// Auth0UserInfo represents the user information returned from Auth0's /userinfo endpoint
type Auth0UserInfo struct {
	Sub   string `json:"sub"`   // Auth0 user ID
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Auth0Service handles interactions with Auth0 API
type Auth0Service struct {
	domain     string
	httpClient *http.Client
}

// NewAuth0Service creates a new Auth0 service instance
func NewAuth0Service(cfg *config.Config) *Auth0Service {
	return &Auth0Service{
		domain: cfg.Auth0Domain,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetUserInfo fetches user information from Auth0's /userinfo endpoint
// accessToken is the JWT access token from the Authorization header
func (s *Auth0Service) GetUserInfo(accessToken string) (*Auth0UserInfo, error) {
	// Construct the userinfo endpoint URL
	// If domain already includes a protocol (for testing), use it as-is
	var url string
	if strings.HasPrefix(s.domain, "http://") || strings.HasPrefix(s.domain, "https://") {
		url = fmt.Sprintf("%s/userinfo", s.domain)
	} else {
		url = fmt.Sprintf("https://%s/userinfo", s.domain)
	}

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add the access token to the Authorization header
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// Execute the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't override the return value
			// In production, you might want to use a proper logger here
			_ = closeErr
		}
	}()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var userInfo Auth0UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	return &userInfo, nil
}
