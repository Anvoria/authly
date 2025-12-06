package auth

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

func JWKSHandler(ks *KeyStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		set := ks.JWKS()

		// Marshal JWKS set to JSON
		data, err := json.Marshal(set)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to marshal JWKS",
			})
		}

		// Set proper content type for JWKS
		c.Set("Content-Type", "application/json")

		// Return raw JSON (not wrapped in success response)
		return c.Send(data)
	}
}
