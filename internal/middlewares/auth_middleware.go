package middlewares

import (
	"fmt"
	"scanner/config"
	"scanner/internal/utils"
	"slices"

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
	fmt.Println(ip, allowedIPs)
	if len(allowedIPs) > 0 && !slices.Contains(allowedIPs, ip) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Forbidden",
		})
	}

	return c.Next()
}
