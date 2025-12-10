package middlewares

import (
	"fmt"
	"log"
	"net/http"
	"scanner/config"
	"scanner/internal/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func WebserviceMiddleware() func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cfg := config.GetConfig()
		headerAPI := c.Get(cfg.Webservice.HeaderKey)
		if headerAPI == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API key",
			})
		}

		if headerAPI != cfg.Webservice.ApiKey {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API key",
			})
		}

		ip := c.IP()
		var ips []string
		if strings.Contains(ip, ",") {
			ips = strings.Split(ip, ",")
			for i, ip := range ips {
				ips[i] = strings.TrimSpace(ip)
			}
		} else {
			ips = []string{strings.TrimSpace(ip)}
		}

		log.Println(ips)
		err := checkClientIP(ips, cfg.Webservice.AllowedIPs)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Next()
	}
}

func checkClientIP(clientIps []string, allowedIPs []string) error {
	for _, allowedIP := range allowedIPs {
		for _, clientIp := range clientIps {
			if clientIp == allowedIP {
				return nil
			}
		}
	}

	return fmt.Errorf("invalid IP address")
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
