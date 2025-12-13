package oicd

import (
	"github.com/gofiber/fiber/v2"
)

func OpenIDConfigurationHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{})
	}
}
