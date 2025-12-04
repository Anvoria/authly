package utils

import (
	"github.com/gofiber/fiber/v2"
)

// SuccessResponse sends a success JSON response
func SuccessResponse(c *fiber.Ctx, data any, message string, code ...int) error {
	statusCode := fiber.StatusOK
	if len(code) > 0 {
		statusCode = code[0]
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"success": true,
		"data":    data,
		"message": message,
	})
}

// ErrorResponse sends an error JSON response
func ErrorResponse(c *fiber.Ctx, message string, code ...int) error {
	statusCode := fiber.StatusInternalServerError
	if len(code) > 0 {
		statusCode = code[0]
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}
