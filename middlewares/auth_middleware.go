package middlewares

import (
	"scanner/utils"

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

func WsPasetoMiddleware(privateKeySeed string) func(*fiber.Ctx) error {
	privateKey := utils.LoadPrivateKey(privateKeySeed)
	return pasetoware.New(pasetoware.Config{
		PrivateKey:  privateKey,
		PublicKey:   privateKey.Public(),
		TokenLookup: [2]string{"query", "token"},
		ContextKey:  "username",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.SendStatus(fiber.StatusUnauthorized)
		},
	})
}
