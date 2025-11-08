package middlewares

import (
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
