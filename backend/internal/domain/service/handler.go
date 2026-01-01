package service

import (
	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	serviceService ServiceInterface
}

// NewHandler creates a Handler configured with the provided ServiceInterface.
func NewHandler(s ServiceInterface) *Handler {
	return &Handler{serviceService: s}
}

// CreateService handles the creation of a new service
func (h *Handler) CreateService(c *fiber.Ctx) error {
	var req struct {
		Slug          string   `json:"slug"`
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		Domain        string   `json:"domain"`
		RedirectURIs  []string `json:"redirect_uris"`
		AllowedScopes []string `json:"allowed_scopes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	if req.Slug == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "Slug is required", fiber.StatusBadRequest))
	}
	if req.Name == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "Name is required", fiber.StatusBadRequest))
	}

	svc, err := h.serviceService.Create(req.Slug, req.Name, req.Description, req.Domain, req.RedirectURIs, req.AllowedScopes)
	if err != nil {
		if err == ErrServiceClientIDExists || err == ErrServiceDomainExists {
			return utils.ErrorResponse(c, utils.NewAPIError("DUPLICATE_RESOURCE", err.Error(), fiber.StatusConflict))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service created successfully", fiber.StatusCreated)
}

// GetService handles the retrieval of a service by ID
func (h *Handler) GetService(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	svc, err := h.serviceService.FindByID(id)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", err.Error(), fiber.StatusNotFound))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service retrieved successfully")
}

// GetServiceByClientID handles the retrieval of a service by client_id
func (h *Handler) GetServiceByClientID(c *fiber.Ctx) error {
	clientID := c.Params("client_id")
	if clientID == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "client_id is required", fiber.StatusBadRequest))
	}

	svc, err := h.serviceService.FindByClientID(clientID)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", err.Error(), fiber.StatusNotFound))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service retrieved successfully")
}

// ListServices handles the retrieval of all services
func (h *Handler) ListServices(c *fiber.Ctx) error {
	activeOnly := c.Query("active") == "true"

	var services []*Service
	var err error

	if activeOnly {
		services, err = h.serviceService.FindActive()
	} else {
		services, err = h.serviceService.FindAll()
	}

	if err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	responses := make([]*ServiceResponse, len(services))
	for i, svc := range services {
		responses[i] = svc.ToResponse()
	}

	return utils.SuccessResponse(c, fiber.Map{
		"services": responses,
		"count":    len(responses),
	}, "Services retrieved successfully")
}

// UpdateService handles the update of a service
func (h *Handler) UpdateService(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Domain      *string `json:"domain"`
		Active      *bool   `json:"active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, utils.NewAPIError("INVALID_BODY", "Invalid request body", fiber.StatusBadRequest))
	}

	svc, err := h.serviceService.Update(id, req.Name, req.Description, req.Domain, req.Active)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", err.Error(), fiber.StatusNotFound))
		}
		if err == ErrCannotUpdateSystemService {
			return utils.ErrorResponse(c, utils.NewAPIError("FORBIDDEN", err.Error(), fiber.StatusForbidden))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service updated successfully")
}

// DeleteService handles the deletion of a service
func (h *Handler) DeleteService(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, utils.NewAPIError("VALIDATION_ERROR", "ID is required", fiber.StatusBadRequest))
	}

	err := h.serviceService.Delete(id)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, utils.NewAPIError("RESOURCE_NOT_FOUND", err.Error(), fiber.StatusNotFound))
		}
		if err == ErrCannotDeleteSystemService {
			return utils.ErrorResponse(c, utils.NewAPIError("FORBIDDEN", err.Error(), fiber.StatusForbidden))
		}
		return utils.ErrorResponse(c, utils.NewAPIError("INTERNAL_SERVER_ERROR", err.Error(), fiber.StatusInternalServerError))
	}

	return utils.SuccessResponse(c, nil, "Service deleted successfully")
}
