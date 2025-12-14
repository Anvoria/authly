package oidc

import (
	"strings"

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
				"openid", "profile", "email",
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

// Token handles the OAuth2 token request (authorization code exchange)
func (h *Handler) Token(c *fiber.Ctx) error {
	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.OIDCErrorResponse(c, "invalid_request", err.Error())
	}

	// Validate required fields
	if req.GrantType == "" {
		return utils.OIDCErrorResponse(c, "invalid_request", "grant_type is required")
	}
	if req.Code == "" {
		return utils.OIDCErrorResponse(c, "invalid_request", "code is required")
	}
	if req.RedirectURI == "" {
		return utils.OIDCErrorResponse(c, "invalid_request", "redirect_uri is required")
	}
	if req.ClientID == "" {
		return utils.OIDCErrorResponse(c, "invalid_request", "client_id is required")
	}

	// Get session from cookie
	sessionCookie := c.Cookies("session")
	if sessionCookie == "" {
		return utils.OIDCErrorResponse(c, "invalid_grant", "Session cookie is required", fiber.StatusUnauthorized)
	}

	// Parse session cookie: format is "sessionID:secret"
	parts := strings.SplitN(sessionCookie, ":", 2)
	if len(parts) != 2 {
		return utils.OIDCErrorResponse(c, "invalid_grant", "Invalid session cookie format", fiber.StatusUnauthorized)
	}

	sessionIDStr := parts[0]
	refreshSecret := parts[1]

	if sessionIDStr == "" || refreshSecret == "" {
		return utils.OIDCErrorResponse(c, "invalid_grant", "Invalid session cookie", fiber.StatusUnauthorized)
	}

	// Parse session ID as UUID
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return utils.OIDCErrorResponse(c, "invalid_grant", "Invalid session ID format", fiber.StatusUnauthorized)
	}

	// Call service with existing session
	res, err := h.service.ExchangeCode(&req, sessionID, refreshSecret)
	if err != nil {
		switch err {
		case ErrInvalidGrant:
			return utils.OIDCErrorResponse(c, "invalid_grant", err.Error())
		case ErrInvalidCode:
			return utils.OIDCErrorResponse(c, "invalid_grant", "The provided authorization code is invalid, expired, or already used", fiber.StatusBadRequest)
		case ErrInvalidClientID:
			return utils.OIDCErrorResponse(c, "invalid_client", "Invalid client_id")
		case ErrInvalidRedirectURI:
			return utils.OIDCErrorResponse(c, "invalid_grant", "redirect_uri mismatch")
		case ErrInvalidCodeVerifier:
			return utils.OIDCErrorResponse(c, "invalid_grant", "code_verifier is invalid")
		case ErrInvalidClientSecret:
			return utils.OIDCErrorResponse(c, "invalid_client", "Invalid client_secret")
		default:
			return utils.OIDCErrorResponse(c, "server_error", err.Error(), fiber.StatusInternalServerError)
		}
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// UserInfo handles the OIDC UserInfo endpoint
// Returns user information based on scopes from the access token
func (h *Handler) UserInfo(c *fiber.Ctx) error {
	identity, ok := c.Locals(auth.IdentityKey).(*auth.Identity)
	if !ok || identity == nil {
		return utils.OIDCErrorResponse(c, "invalid_token", "Invalid or missing access token", fiber.StatusUnauthorized)
	}

	claims, ok := c.Locals("token_claims").(*auth.AccessTokenClaims)
	if !ok || claims == nil {
		return utils.OIDCErrorResponse(c, "invalid_token", "Unable to extract token claims", fiber.StatusUnauthorized)
	}

	scopes := claims.GetRequestedScopes()
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	userInfo, err := h.service.GetUserInfo(identity.UserID, scopes)
	if err != nil {
		return utils.OIDCErrorResponse(c, "server_error", err.Error(), fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(userInfo)
}

// ValidateAuthorization validates an OAuth2/OIDC authorization request
// This endpoint does not require authentication and is used by frontend to validate request parameters
func (h *Handler) ValidateAuthorization(c *fiber.Ctx) error {
	var req AuthorizeRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusOK).JSON(&ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "Failed to parse query parameters: " + err.Error(),
		})
	}

	res := h.service.ValidateAuthorizationRequest(&req)
	return c.Status(fiber.StatusOK).JSON(res)
}

// ConfirmAuthorization handles the OAuth2/OIDC authorization confirmation
// This endpoint is called after user confirms authorization on the frontend
func (h *Handler) ConfirmAuthorization(c *fiber.Ctx) error {
	var req ConfirmAuthorizationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_request",
			ErrorDescription: "Failed to parse request body: " + err.Error(),
		})
	}

	// Validate required fields
	if req.ResponseType == "" {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_request",
			ErrorDescription: "response_type is required",
		})
	}
	if req.ClientID == "" {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_request",
			ErrorDescription: "client_id is required",
		})
	}
	if req.RedirectURI == "" {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_request",
			ErrorDescription: "redirect_uri is required",
		})
	}
	if req.Scope == "" {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_request",
			ErrorDescription: "scope is required",
		})
	}

	// Check if user is authenticated
	identity, ok := c.Locals(auth.IdentityKey).(*auth.Identity)
	if !ok || identity == nil {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "login_required",
			ErrorDescription: "User must be logged in to confirm authorization",
		})
	}

	userID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "server_error",
			ErrorDescription: "Invalid user ID",
		})
	}

	// Convert ConfirmAuthorizationRequest to AuthorizeRequest
	authorizeReq := &AuthorizeRequest{
		ResponseType:        req.ResponseType,
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}

	// Call service to authorize
	res, err := h.service.Authorize(authorizeReq, userID)
	if err != nil {
		var errorCode string
		var errorDesc string
		switch err {
		case ErrInvalidClientID:
			errorCode = "invalid_client"
			errorDesc = "Invalid client_id"
		case ErrInvalidRedirectURI:
			errorCode = "invalid_redirect_uri"
			errorDesc = "The redirect_uri is not allowed for this client"
		case ErrInvalidScope:
			errorCode = "invalid_scope"
			errorDesc = "One or more requested scopes are not allowed"
		case ErrInvalidResponseType:
			errorCode = "unsupported_response_type"
			errorDesc = "Only 'code' response_type is supported"
		case ErrInvalidCodeChallenge:
			errorCode = "invalid_code_challenge"
			errorDesc = "Invalid code_challenge format"
		case ErrInvalidCodeChallengeMethod:
			errorCode = "unsupported_code_challenge_method"
			errorDesc = "Only 'S256' code_challenge_method is supported"
		case ErrClientNotActive:
			errorCode = "unauthorized_client"
			errorDesc = "Client is not active"
		case ErrUnauthorizedClient:
			errorCode = "unauthorized_client"
			errorDesc = "Client is not authorized"
		default:
			errorCode = "server_error"
			errorDesc = err.Error()
		}
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            errorCode,
			ErrorDescription: errorDesc,
		})
	}

	// Build redirect URI with code and state
	redirectURI := req.RedirectURI + "?code=" + res.Code
	if res.State != "" {
		redirectURI += "&state=" + res.State
	}

	return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
		Success:     true,
		RedirectURI: redirectURI,
	})
}
