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
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, "invalid_body", fiber.StatusBadRequest)
	}

	if req.Code == "" {
		return utils.ErrorResponse(c, "code is required", fiber.StatusBadRequest)
	}
	if req.Name == "" {
		return utils.ErrorResponse(c, "name is required", fiber.StatusBadRequest)
	}

	svc, err := h.serviceService.Create(req.Code, req.Name, req.Description)
	if err != nil {
		if err == ErrServiceCodeExists {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusConflict)
		}
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service created successfully", fiber.StatusCreated)
}

// GetService handles the retrieval of a service by ID
func (h *Handler) GetService(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, "id is required", fiber.StatusBadRequest)
	}

	svc, err := h.serviceService.FindByID(id)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusNotFound)
		}
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service retrieved successfully")
}

// GetServiceByCode handles the retrieval of a service by code
func (h *Handler) GetServiceByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return utils.ErrorResponse(c, "code is required", fiber.StatusBadRequest)
	}

	svc, err := h.serviceService.FindByCode(code)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusNotFound)
		}
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
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
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
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
		return utils.ErrorResponse(c, "id is required", fiber.StatusBadRequest)
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Active      *bool   `json:"active"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, "invalid_body", fiber.StatusBadRequest)
	}

	svc, err := h.serviceService.Update(id, req.Name, req.Description, req.Active)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusNotFound)
		}
		if err == ErrCannotUpdateSystemService {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusForbidden)
		}
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
	}

	return utils.SuccessResponse(c, fiber.Map{
		"service": svc.ToResponse(),
	}, "Service updated successfully")
}

// DeleteService handles the deletion of a service
func (h *Handler) DeleteService(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return utils.ErrorResponse(c, "id is required", fiber.StatusBadRequest)
	}

	err := h.serviceService.Delete(id)
	if err != nil {
		if err == ErrServiceNotFound {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusNotFound)
		}
		if err == ErrCannotDeleteSystemService {
			return utils.ErrorResponse(c, err.Error(), fiber.StatusForbidden)
		}
		return utils.ErrorResponse(c, err.Error(), fiber.StatusInternalServerError)
	}

	return utils.SuccessResponse(c, nil, "Service deleted successfully")
}
