package auth

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/Anvoria/authly/internal/utils"
)

type Handler struct {
	authService AuthService
	userService user.Service
}

// NewHandler creates a new Handler configured with the provided AuthService and user.Service.
func NewHandler(s AuthService, userService user.Service) *Handler {
	return &Handler{authService: s, userService: userService}
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req user.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, ErrInvalidBody.Error(), fiber.StatusBadRequest)
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

func (h *Handler) Register(c *fiber.Ctx) error {
	var req user.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, ErrInvalidBody.Error(), fiber.StatusBadRequest)
	}

	res, err := h.authService.Register(req)
	if err != nil {
		return utils.ErrorResponse(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.SuccessResponse(c, fiber.Map{
		"user": res,
	}, "User registered successfully")
}

func (h *Handler) GetUserInfo(c *fiber.Ctx) error {
	identity := GetIdentity(c)
	if identity == nil {
		return utils.ErrorResponse(c, ErrUnauthorized.Error(), fiber.StatusUnauthorized)
	}

	userInfo, err := h.userService.GetUserInfo(identity.UserID)
	if err != nil {
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
	}

	return utils.SuccessResponse(c, userInfo.ToResponse(), "User info retrieved successfully")
}
