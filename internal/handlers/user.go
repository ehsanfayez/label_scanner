package handlers

import (
	"scanner/config"
	"scanner/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func Login(c *fiber.Ctx) error {
	person := &LoginRequest{}
	if err := c.BodyParser(person); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	if errs := utils.Validate(person); len(errs) > 0 && errs[0].Error {
		c.Status(fiber.StatusUnprocessableEntity)
		response := fiber.Map{
			"status":  "false",
			"message": "Please review your input as there are errors in the provided data.",
			"errors":  utils.ValidationError(errs), // Pass the array of error messages directly
		}

		return c.JSON(response)
	}

	username := config.GetConfig().AuthConfig.Username
	password := config.GetConfig().AuthConfig.Password

	if username != person.Username || !utils.CheckPasswordHash(person.Password, password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "username or password is incorrect",
		})
	}

	encryptedToken, err := utils.CreateToken(config.GetConfig().AuthConfig.PrivateKeySeed, person.Username, config.GetConfig().AuthConfig.ExpiresIn)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{"token": encryptedToken})
}
