package oicd

import (
	"github.com/gofiber/fiber/v2"
)

func OpenIDConfigurationHandler(domain string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"issuer": domain,

			"authorization_endpoint": domain + "/oauth/authorize",
			"token_endpoint":         domain + "/oauth/token",
			"userinfo_endpoint":      domain + "/oauth/userinfo",
			"jwks_uri":               domain + "/.well-known/jwks.json",

			"scopes_supported": []string{
				"openid", "profile", "email", "tp", "ke",
			},

			"response_types_supported": []string{"code"},
			"grant_types_supported": []string{
				"authorization_code",
				"refresh_token",
			},

			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}
}
