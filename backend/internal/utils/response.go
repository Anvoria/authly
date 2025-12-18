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

// ErrorResponse sends an error JSON response with a failure flag and message.
// If an explicit HTTP status code is provided it is used; otherwise 500 Internal Server Error is sent.
// The JSON body contains the fields "success": false and "error": <message>.
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

// OIDCErrorResponse sends an OpenID Connect error response as JSON.
// It sets the HTTP status to 400 (Bad Request) unless an explicit status code
// is provided via the optional variadic `code` argument. The JSON body contains
// the keys "error" and "error_description" with the provided values.
// It returns any error encountered while writing the response.
func OIDCErrorResponse(c *fiber.Ctx, message, errorDescription string, code ...int) error {
	statusCode := fiber.StatusBadRequest
	if len(code) > 0 {
		statusCode = code[0]
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"error":             message,
		"error_description": errorDescription,
	})
}
