package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v2"
)

var (
	verifier      *oidc.IDTokenVerifier
	provider      *oidc.Provider
	authority     string
	configMutex   sync.RWMutex
	lastFetchTime time.Time

	// Optional validation settings
	expectedAudience string
	requiredScopes   []string
)

// InitOIDCProvider initializes the OIDC provider and token verifier
func InitOIDCProvider(ctx context.Context, authorityURL, audience, scopes string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	authority = authorityURL
	expectedAudience = audience

	// Parse required scopes
	if scopes != "" {
		requiredScopes = strings.Split(scopes, ",")
		for i, scope := range requiredScopes {
			requiredScopes[i] = strings.TrimSpace(scope)
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Discover OIDC configuration from well-known endpoint
	var err error
	provider, err = oidc.NewProvider(ctx, authority)
	if err != nil {
		return fmt.Errorf("failed to fetch OIDC configuration: %w", err)
	}

	// Create a verifier for JWT tokens
	// Note: go-oidc automatically fetches and caches JWKS
	config := &oidc.Config{}

	if expectedAudience != "" {
		config.ClientID = expectedAudience
		log.Printf("Audience validation enabled: %s", expectedAudience)
	} else {
		config.SkipClientIDCheck = true
		log.Printf("Audience validation disabled (skipping aud claim check)")
	}

	verifier = provider.Verifier(config)

	lastFetchTime = time.Now()
	log.Printf("OIDC provider initialized successfully from %s", authority)

	if len(requiredScopes) > 0 {
		log.Printf("Required scopes: %v", requiredScopes)
	}

	return nil
}

// RefreshOIDCProviderIfNeeded refreshes the OIDC configuration if needed (cached for 1 hour)
func RefreshOIDCProviderIfNeeded(ctx context.Context) {
	configMutex.RLock()
	elapsed := time.Since(lastFetchTime)
	configMutex.RUnlock()

	// Refresh every hour
	if elapsed > time.Hour {
		log.Println("Refreshing OIDC configuration...")
		if err := InitOIDCProvider(ctx, authority, expectedAudience, strings.Join(requiredScopes, ",")); err != nil {
			log.Printf("Failed to refresh OIDC configuration: %v", err)
		}
	}
}

// GetVerifier returns the current OIDC verifier (thread-safe)
func GetVerifier() *oidc.IDTokenVerifier {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return verifier
}

// GetRequiredScopes returns the required scopes (thread-safe)
func GetRequiredScopes() []string {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return requiredScopes
}

// ValidateScopes checks if the token contains all required scopes
func ValidateScopes(claims *TokenClaims) error {
	configMutex.RLock()
	currentRequiredScopes := requiredScopes
	configMutex.RUnlock()

	if len(currentRequiredScopes) == 0 {
		return nil // No scope validation required
	}

	// Check against permissions since that's what we extract from the token
	for _, required := range currentRequiredScopes {
		found := false
		for _, perm := range claims.Permissions {
			if perm == required {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing required scope: %s", required)
		}
	}

	return nil
}

type AuthContext struct {
	Username string
	Roles    []string
}

type TokenClaims struct {
	Subject     string   `json:"sub"`
	Issuer      string   `json:"iss"`
	Audience    []string `json:"aud"`
	Expiry      int64    `json:"exp"`
	IssuedAt    int64    `json:"iat"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Username    string   `json:"preferred_username"`
}

func GetUsernameFromContext(c *fiber.Ctx) string {
	username, _ := GetUserInfoFromContext(c)
	return username
}

func GetUserInfoFromContext(c *fiber.Ctx) (string, []string) {
	// Handle claims from context
	if claims := c.Locals("claims"); claims != nil {
		// Handle string type (JSON string)
		if claimsStr, ok := claims.(string); ok {
			// Try to unmarshal as AuthContext first
			var authCtx AuthContext
			if err := json.Unmarshal([]byte(claimsStr), &authCtx); err == nil && authCtx.Username != "" {
				return authCtx.Username, authCtx.Roles
			}

			// Try to unmarshal as TokenClaims
			var tokenClaims TokenClaims
			if err := json.Unmarshal([]byte(claimsStr), &tokenClaims); err == nil && tokenClaims.Username != "" {
				return tokenClaims.Username, tokenClaims.Roles
			}
		}

		// Handle AuthContext type (pointer)
		if authCtx, ok := claims.(*AuthContext); ok {
			return authCtx.Username, authCtx.Roles
		}

		if tokenClaims, ok := claims.(*TokenClaims); ok {
			return tokenClaims.Username, tokenClaims.Roles
		}
	}

	return "", []string{}
}
