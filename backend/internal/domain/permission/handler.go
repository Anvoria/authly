package permission

import (
	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	permissionService ServiceInterface
}

func NewHandler(s ServiceInterface) *Handler {
	return &Handler{permissionService: s}
}

// ListPermissions handles the retrieval of permissions for a specific service
func (h *Handler) ListPermissions(c *fiber.Ctx) error {
	serviceID := c.Query("service_id")
	if serviceID == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "service_id is required", fiber.StatusBadRequest))
	}

	limit := 10
	offset := 0
	if l := c.QueryInt("limit"); l > 0 {
		limit = l
	}
	if o := c.QueryInt("offset"); o >= 0 {
		offset = o
	}

	var resource *string
	if r := c.Query("resource"); r != "" {
		resource = &r
	}

	permissions, count, err := h.permissionService.ListPermissions(serviceID, resource, limit, offset)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	responses := make([]*PermissionResponse, len(permissions))
	for i, p := range permissions {
		responses[i] = p.ToResponse()
	}

	return utils.SuccessResponse(c, fiber.Map{
		"permissions": responses,
		"count":       count,
		"limit":       limit,
		"offset":      offset,
	}, "Permissions retrieved successfully")
}

// CreatePermission handles the creation of a new permission definition
func (h *Handler) CreatePermission(c *fiber.Ctx) error {
	var req struct {
		ServiceID string  `json:"service_id"`
		Name      string  `json:"name"`
		Bit       uint8   `json:"bit"`
		Resource  *string `json:"resource"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	if req.ServiceID == "" || req.Name == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "service_id and name are required", fiber.StatusBadRequest))
	}

	perm, err := h.permissionService.CreatePermission(req.ServiceID, req.Name, req.Bit, req.Resource)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"permission": perm.ToResponse(),
	}, "Permission created successfully", fiber.StatusCreated)
}

// UpdatePermission handles the update of a permission
func (h *Handler) UpdatePermission(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	var req struct {
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	perm, err := h.permissionService.UpdatePermission(id, req.Name, req.Active)
	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"permission": perm.ToResponse(),
	}, "Permission updated successfully")
}

// DeletePermission handles the deletion of a permission
func (h *Handler) DeletePermission(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	if err := h.permissionService.DeletePermission(id); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, nil, "Permission deleted successfully")
}
