package auth

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Anvoria/authly/internal/utils"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Handler struct {
	authService *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{authService: s}
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, "invalid_body", fiber.StatusBadRequest)
	}

	res, err := h.authService.Login(
		req.Username,
		req.Password,
		c.Get("User-Agent"),
		c.IP(),
	)

	if err != nil {
		return utils.ErrorResponse(c, err.Error(), fiber.StatusUnauthorized)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    fmt.Sprintf("%s:%s", res.RefreshSID, res.RefreshToken),
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: "None",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	return utils.SuccessResponse(c, fiber.Map{
		"access_token": res.AccessToken,
		"user":         res.User,
	}, "Login successful")
}
