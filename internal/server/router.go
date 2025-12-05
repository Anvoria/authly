package server

import "github.com/gofiber/fiber/v2"

// SetupRoutes sets up the routes for the application
func SetupRoutes(app *fiber.App) {
	api := app.Group("/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})
}