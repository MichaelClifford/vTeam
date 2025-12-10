package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OAuthProvider represents a generic OAuth provider configuration
type OAuthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// OAuthCallbackData represents the stored OAuth callback data
type OAuthCallbackData struct {
	Provider     string    `json:"provider"`
	UserID       string    `json:"userId"`
	Code         string    `json:"code"`
	State        string    `json:"state"`
	Error        string    `json:"error,omitempty"`
	ErrorDesc    string    `json:"error_description,omitempty"`
	ReceivedAt   time.Time `json:"receivedAt"`
	Consumed     bool      `json:"consumed"`
	ConsumedAt   time.Time `json:"consumedAt,omitempty"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresIn    int64     `json:"expiresIn,omitempty"`
	TokenType    string    `json:"tokenType,omitempty"`
}

// getOAuthProvider retrieves OAuth provider configuration from environment
func getOAuthProvider(provider string) (*OAuthProvider, error) {
	provider = strings.ToLower(provider)

	switch provider {
	case "google":
		clientID := os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("Google OAuth not configured")
		}
		return &OAuthProvider{
			Name:         "google",
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/drive",
				"https://www.googleapis.com/auth/drive.file",
				"https://www.googleapis.com/auth/drive.readonly",
			},
		}, nil

	case "github":
		// GitHub uses existing handler, but we can support it here too
		clientID := os.Getenv("GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("GitHub OAuth not configured")
		}
		return &OAuthProvider{
			Name:         "github",
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     "https://github.com/login/oauth/access_token",
			Scopes:       []string{"repo", "user"},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// HandleOAuth2Callback handles GET /oauth2callback
// This is a generic OAuth2 callback endpoint that can handle multiple providers
func HandleOAuth2Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")
	provider := c.Query("provider")

	// Default to google if no provider specified (for MCP compatibility)
	if provider == "" {
		provider = "google"
	}

	log.Printf("OAuth2 callback received - provider: %s, hasCode: %v, hasState: %v, error: %s",
		provider, code != "", state != "", errorParam)

	// Create callback data record
	callbackData := OAuthCallbackData{
		Provider:   provider,
		Code:       code,
		State:      state,
		Error:      errorParam,
		ErrorDesc:  errorDesc,
		ReceivedAt: time.Now(),
		Consumed:   false,
	}

	// Try to get user ID from session (may not be available for MCP flows)
	if userID, exists := c.Get("userID"); exists && userID != nil {
		callbackData.UserID = userID.(string)
	}

	// Handle OAuth errors
	if errorParam != "" {
		log.Printf("OAuth error received: %s - %s", errorParam, errorDesc)
		// Store the error for MCP to retrieve
		if err := storeOAuthCallback(c.Request.Context(), state, &callbackData); err != nil {
			log.Printf("Failed to store OAuth error: %v", err)
		}
		c.HTML(http.StatusOK, "<html><body><h1>Authorization Error</h1><p>Error: "+errorParam+"</p><p>"+errorDesc+"</p><p>Provider: "+provider+"</p><p>You can close this window.</p></body></html>", nil)
		return
	}

	// Validate required parameters
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	// Get provider configuration
	providerConfig, err := getOAuthProvider(provider)
	if err != nil {
		log.Printf("Failed to get OAuth provider config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth provider not configured"})
		return
	}

	// Exchange code for token
	tokenData, err := exchangeOAuthCode(c.Request.Context(), providerConfig, code)
	if err != nil {
		log.Printf("Failed to exchange OAuth code: %v", err)
		callbackData.Error = "token_exchange_failed"
		callbackData.ErrorDesc = err.Error()
		// Store the failure
		if serr := storeOAuthCallback(c.Request.Context(), state, &callbackData); serr != nil {
			log.Printf("Failed to store OAuth exchange error: %v", serr)
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to exchange authorization code"})
		return
	}

	// Populate token data
	callbackData.AccessToken = tokenData.AccessToken
	callbackData.RefreshToken = tokenData.RefreshToken
	callbackData.ExpiresIn = tokenData.ExpiresIn
	callbackData.TokenType = tokenData.TokenType

	// Store the callback data for MCP or other consumers to retrieve
	if err := storeOAuthCallback(c.Request.Context(), state, &callbackData); err != nil {
		log.Printf("Failed to store OAuth callback: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store OAuth data"})
		return
	}

	log.Printf("OAuth callback stored successfully for state: %s (len=%d)", state[:min(10, len(state))], len(state))

	// Return success page
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
		"<html><body><h1>Authorization Successful!</h1><p>Provider: "+provider+"</p><p>You can close this window.</p><script>window.close();</script></body></html>",
	))
}

// OAuthTokenResponse represents the token exchange response
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// exchangeOAuthCode exchanges an authorization code for an access token
func exchangeOAuthCode(ctx context.Context, provider *OAuthProvider, code string) (*OAuthTokenResponse, error) {
	// Prepare token exchange request
	formData := fmt.Sprintf("code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code,
		provider.ClientID,
		provider.ClientSecret,
		"http://localhost:8000/oauth2callback",
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenURL, strings.NewReader(formData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token received")
	}

	return &tokenResp, nil
}

// storeOAuthCallback stores OAuth callback data in a Secret for retrieval by MCP or other consumers
func storeOAuthCallback(ctx context.Context, state string, data *OAuthCallbackData) error {
	if state == "" {
		// Generate a default key if no state provided
		state = fmt.Sprintf("callback_%d", time.Now().Unix())
	}

	const secretName = "oauth-callbacks"

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Create Secret
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				// Fetch again to get resourceVersion
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch Secret after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get Secret: %w", err)
			}
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		// Serialize callback data
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal callback data: %w", err)
		}

		secret.Data[state] = b

		// Update Secret
		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}

		return nil
	}

	return fmt.Errorf("failed to update Secret after retries")
}

// GetOAuthCallback retrieves OAuth callback data by state
func GetOAuthCallback(ctx context.Context, state string) (*OAuthCallbackData, error) {
	const secretName = "oauth-callbacks"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("callback not found")
		}
		return nil, fmt.Errorf("failed to read Secret: %w", err)
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("callback not found")
	}

	raw, ok := secret.Data[state]
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("callback not found")
	}

	var data OAuthCallbackData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to decode callback data: %w", err)
	}

	return &data, nil
}

// GetOAuthCallbackEndpoint handles GET /oauth2callback/status?state=xxx
// This allows MCP or other consumers to check the status of their OAuth flow
func GetOAuthCallbackEndpoint(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing state parameter"})
		return
	}

	data, err := GetOAuthCallback(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "callback not found"})
		return
	}

	// Don't return sensitive tokens in the response
	response := gin.H{
		"provider":   data.Provider,
		"userId":     data.UserID,
		"receivedAt": data.ReceivedAt,
		"consumed":   data.Consumed,
		"hasToken":   data.AccessToken != "",
	}

	if data.Error != "" {
		response["error"] = data.Error
		response["errorDescription"] = data.ErrorDesc
	}

	c.JSON(http.StatusOK, response)
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
