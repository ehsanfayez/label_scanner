package middlewares

import (
	"log"
	"scanner/config"
	"scanner/internal/utils"
	"slices"
	"strings"

	pasetoware "github.com/gofiber/contrib/paseto"
	"github.com/gofiber/fiber/v2"
)

func PasetoMiddleware(privateKeySeed string) func(*fiber.Ctx) error {
	privateKey := utils.LoadPrivateKey(privateKeySeed)
	return pasetoware.New(pasetoware.Config{
		TokenPrefix: "Bearer",
		PrivateKey:  privateKey,
		PublicKey:   privateKey.Public(),
		ContextKey:  "claims",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.SendStatus(fiber.StatusUnauthorized)
		},
	})
}

func IPMiddleware(c *fiber.Ctx) error {
	ip := c.IP()
	allowedIPs := config.GetConfig().OCRConfig.IPs
	ips := strings.Split(ip, ",")
	for _, ip := range ips {
		if slices.Contains(allowedIPs, strings.TrimSpace(ip)) {
			return c.Next()
		}
	}

	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"message": "Forbidden",
	})
}

func OAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Refresh OIDC config if needed (cached, non-blocking)
		go utils.RefreshOIDCProviderIfNeeded(c.Context())

		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Missing Authorization header")
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid Authorization header format")
		}

		tokenString := parts[1]

		// Verify the token using go-oidc (verifies signature, expiry, issuer, etc.)
		currentVerifier := utils.GetVerifier()
		idToken, err := currentVerifier.Verify(c.Context(), tokenString)
		if err != nil {
			log.Printf("Token verification failed: %v", err)
			if strings.Contains(err.Error(), "token is expired") {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"message": "Token is expired",
				})
			}

			log.Printf("Token verification failed: %v", err)
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		}

		// Extract custom claims (roles, permissions, etc.)
		var claims utils.TokenClaims
		if err := idToken.Claims(&claims); err != nil {
			log.Printf("Failed to parse token claims: %v", err)
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token claims")
		}

		// Validate required scopes if configured
		if err := utils.ValidateScopes(&claims); err != nil {
			log.Printf("Scope validation failed: %v", err)
			return fiber.NewError(fiber.StatusForbidden, "Insufficient scopes")
		}

		// Add claims to context for use in handlers
		c.Locals("claims", &claims)
		c.Locals("subject", idToken.Subject)

		// Call the next handler
		return c.Next()
	}
}
