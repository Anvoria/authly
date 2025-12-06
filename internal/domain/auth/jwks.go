package auth

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// JWKSHandler constructs a Fiber handler that serves the JSON Web Key Set (JWKS) from the provided KeyStore.
// The handler retrieves the JWKS from ks, marshals it to JSON, and writes the raw JSON with Content-Type "application/json".
// If marshaling fails the handler responds with HTTP 500 and a JSON body `{"error":"failed to marshal JWKS"}`.
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