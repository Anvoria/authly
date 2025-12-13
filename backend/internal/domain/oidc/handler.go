package oidc

import (
	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service ServiceInterface
}

// NewHandler creates a new OIDC handler
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

// OpenIDConfigurationHandler returns the OpenID Connect configuration
func OpenIDConfigurationHandler(domain string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"issuer": domain,

			"authorization_endpoint": domain + "/v1/oauth/authorize",
			"token_endpoint":         domain + "/v1/oauth/token",
			"userinfo_endpoint":      domain + "/v1/oauth/userinfo",
			"jwks_uri":               domain + "/v1/.well-known/jwks.json",

			"scopes_supported": []string{
				"openid", "profile", "email", "tp", "ke",
			},

			"response_types_supported": []string{"code"},
			"grant_types_supported": []string{
				"authorization_code",
				"refresh_token",
			},

			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}
}

// Authorize handles the OAuth2/OIDC authorization request
func (h *Handler) Authorize(c *fiber.Ctx) error {
	var req AuthorizeRequest
	if err := c.QueryParser(&req); err != nil {
		return utils.ErrorResponse(c, "invalid_request: "+err.Error(), fiber.StatusBadRequest)
	}

	// Validate required fields
	if req.ResponseType == "" {
		return utils.ErrorResponse(c, "invalid_request: response_type is required", fiber.StatusBadRequest)
	}
	if req.ClientID == "" {
		return utils.ErrorResponse(c, "invalid_request: client_id is required", fiber.StatusBadRequest)
	}
	if req.RedirectURI == "" {
		return utils.ErrorResponse(c, "invalid_request: redirect_uri is required", fiber.StatusBadRequest)
	}
	if req.Scope == "" {
		return utils.ErrorResponse(c, "invalid_request: scope is required", fiber.StatusBadRequest)
	}

	identity, ok := c.Locals(auth.IdentityKey).(*auth.Identity)
	if !ok || identity == nil {
		return utils.ErrorResponse(c, "login_required", fiber.StatusUnauthorized)
	}

	userID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return utils.ErrorResponse(c, "invalid_user_id", fiber.StatusInternalServerError)
	}

	// Call service
	res, err := h.service.Authorize(&req, userID)
	if err != nil {
		switch err {
		case ErrInvalidClientID:
			return utils.ErrorResponse(c, "invalid_client_id", fiber.StatusBadRequest)
		case ErrInvalidRedirectURI:
			return utils.ErrorResponse(c, "invalid_redirect_uri", fiber.StatusBadRequest)
		case ErrInvalidScope:
			return utils.ErrorResponse(c, "invalid_scope", fiber.StatusBadRequest)
		case ErrInvalidResponseType:
			return utils.ErrorResponse(c, "unsupported_response_type", fiber.StatusBadRequest)
		case ErrInvalidCodeChallenge:
			return utils.ErrorResponse(c, "invalid_code_challenge", fiber.StatusBadRequest)
		case ErrInvalidCodeChallengeMethod:
			return utils.ErrorResponse(c, "unsupported_code_challenge_method", fiber.StatusBadRequest)
		case ErrClientNotActive:
			return utils.ErrorResponse(c, "client_not_active", fiber.StatusBadRequest)
		case ErrUnauthorizedClient:
			return utils.ErrorResponse(c, "unauthorized_client", fiber.StatusBadRequest)
		default:
			return utils.ErrorResponse(c, "server_error: "+err.Error(), fiber.StatusInternalServerError)
		}
	}

	// Redirect to redirect_uri with code and state
	redirectURL := req.RedirectURI + "?code=" + res.Code
	if res.State != "" {
		redirectURL += "&state=" + res.State
	}

	return c.Redirect(redirectURL, fiber.StatusFound)
}
