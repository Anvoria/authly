package user

import (
	"strconv"

	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	userService Service
}

func NewAdminHandler(s Service) *AdminHandler {
	return &AdminHandler{userService: s}
}

// ListUsers handles the retrieval of all users with pagination
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	limit := 10
	offset := 0

	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	users, count, err := h.userService.ListUsers(limit, offset)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	responses := make([]*UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	return utils.SuccessResponse(c, fiber.Map{
		"users":  responses,
		"count":  count,
		"limit":  limit,
		"offset": offset,
	}, "Users retrieved successfully")
}

// UpdateUser handles updating a user
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	var req struct {
		Email    *string `json:"email"`
		Username *string `json:"username"`
		IsActive *bool   `json:"is_active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	user, err := h.userService.UpdateUser(id, req.Email, req.Username, req.IsActive)
	if err != nil {
		if err == ErrUserNotFound {
			return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", err.Error(), fiber.StatusNotFound))
		}
		if err == ErrEmailExists || err == ErrUsernameExists {
			return utils.ErrorResponse(c, utils.NewAPIError("DUPLICATE_RESOURCE", err.Error(), fiber.StatusConflict))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"user": user.ToResponse(),
	}, "User updated successfully")
}

// DeleteUser handles deleting a user
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	err := h.userService.DeleteUser(id)
	if err != nil {
		if err == ErrUserNotFound {
			return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", err.Error(), fiber.StatusNotFound))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, nil, "User deleted successfully")
}
