package role

import (
	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	roleService Service
}

func NewHandler(s Service) *Handler {
	return &Handler{roleService: s}
}

// GetRolesByService handles the retrieval of roles for a specific service
func (h *Handler) GetRolesByService(c *fiber.Ctx) error {
	serviceID := c.Query("service_id")
	if serviceID == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "service_id is required", fiber.StatusBadRequest))
	}

	roles, err := h.roleService.GetRolesByService(serviceID)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"roles": roles,
	}, "Roles retrieved successfully")
}

// CreateRole handles the creation of a new role
func (h *Handler) CreateRole(c *fiber.Ctx) error {
	var req struct {
		ServiceID   string `json:"service_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Bitmask     uint64 `json:"bitmask"`
		IsDefault   bool   `json:"is_default"`
		Priority    int    `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	if req.ServiceID == "" || req.Name == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "service_id and name are required", fiber.StatusBadRequest))
	}

	serviceUUID, err := uuid.Parse(req.ServiceID)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "Invalid service_id format", fiber.StatusBadRequest))
	}

	role := &Role{
		ServiceID:   serviceUUID,
		Name:        req.Name,
		Description: req.Description,
		Bitmask:     req.Bitmask,
		IsDefault:   req.IsDefault,
		Priority:    req.Priority,
	}

	if err := h.roleService.CreateRole(role); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"role": role,
	}, "Role created successfully", fiber.StatusCreated)
}

// UpdateRole handles the update of a role
func (h *Handler) UpdateRole(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Bitmask     uint64 `json:"bitmask"`
		IsDefault   bool   `json:"is_default"`
		Priority    int    `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	role, err := h.roleService.GetRole(id)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", "Role not found", fiber.StatusNotFound))
	}

	role.Name = req.Name
	role.Description = req.Description
	role.Bitmask = req.Bitmask
	role.IsDefault = req.IsDefault
	role.Priority = req.Priority

	if err := h.roleService.UpdateRole(role); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"role": role,
	}, "Role updated successfully")
}

// DeleteRole handles the deletion of a role
func (h *Handler) DeleteRole(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	if err := h.roleService.DeleteRole(id); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, nil, "Role deleted successfully")
}

// AssignRole handles assigning a role to a user
func (h *Handler) AssignRole(c *fiber.Ctx) error {
	var req struct {
		UserID string `json:"user_id"`
		RoleID string `json:"role_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	if req.UserID == "" || req.RoleID == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "user_id and role_id are required", fiber.StatusBadRequest))
	}

	if err := h.roleService.AssignRole(req.UserID, req.RoleID); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, nil, "Role assigned successfully")
}
