package auth

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Anvoria/authly/internal/domain/permission"
	"github.com/Anvoria/authly/internal/domain/user"
	"github.com/Anvoria/authly/internal/utils"
)

type Handler struct {
	authService       AuthService
	userService       user.Service
	permissionService permission.ServiceInterface
}

// NewHandler creates a new Handler configured with the provided AuthService and user.Service.
func NewHandler(s AuthService, userService user.Service, permissionService permission.ServiceInterface) *Handler {
	return &Handler{
		authService:       s,
		userService:       userService,
		permissionService: permissionService,
	}
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
		Name:     "session",
		Value:    fmt.Sprintf("%s:%s", res.RefreshSID, res.RefreshToken),
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: "Lax",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	return utils.SuccessResponse(c, fiber.Map{
		"user": res.User,
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

// Me returns the current authenticated user information based on session cookie
func (h *Handler) Me(c *fiber.Ctx) error {
	identity, ok := c.Locals(IdentityKey).(*Identity)
	if !ok || identity == nil {
		return utils.ErrorResponse(c, "not_authenticated", fiber.StatusUnauthorized)
	}

	user, err := h.userService.GetUserInfo(identity.UserID)
	if err != nil {
		return utils.ErrorResponse(c, "user_not_found", fiber.StatusNotFound)
	}

	permissions, err := h.permissionService.BuildScopes(identity.UserID)
	if err != nil {
		permissions = make(map[string]uint64)
	}

	return utils.SuccessResponse(c, fiber.Map{
		"user":        user.ToResponse(),
		"permissions": permissions,
	}, "User information retrieved successfully")
}
