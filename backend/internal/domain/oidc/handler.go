package oidc

import (
	"log/slog"
	"net/url"
	"strings"

	"github.com/Anvoria/authly/internal/domain/auth"

	"github.com/Anvoria/authly/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service ServiceInterface
}

// NewHandler creates a Handler that serves OIDC HTTP endpoints backed by the provided service.
// The returned Handler delegates OIDC operations to the given ServiceInterface.
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

// OpenIDConfigurationHandler returns an HTTP handler that serves the OpenID Connect discovery document for the provided domain.
// The handler responds with a JSON object containing the issuer, authorization/token/userinfo/jwks endpoints, supported scopes,
// OpenIDConfigurationHandler returns an HTTP handler that serves the OpenID Connect discovery
// (/.well-known/openid-configuration) document for the given issuer domain.
// 
// The handler responds with a JSON object containing the issuer and endpoint URLs (authorization,
// token, userinfo, jwks), supported scopes, supported response and grant types, subject types,
// and supported ID token signing algorithms. The provided domain is used as the issuer base URL
// for all advertised endpoints.
func OpenIDConfigurationHandler(domain string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"issuer": domain,

			"authorization_endpoint": domain + "/v1/oauth/authorize",
			"token_endpoint":         domain + "/v1/oauth/token",
			"userinfo_endpoint":      domain + "/v1/oauth/userinfo",
			"jwks_uri":               domain + "/.well-known/jwks.json",

			"scopes_supported": []string{
				"openid", "profile", "email",
			},

			"response_types_supported": []string{"code"},
			"grant_types_supported": []string{
				"authorization_code",
				"refresh_token",
				"password",
				"client_credentials",
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
		slog.Error("Failed to parse query parameters", "error", err)
		return utils.ErrorResponse(c, "invalid_request: malformed parameters", fiber.StatusBadRequest)
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
		oidcErr := MapErrorToOIDC(err)
		if oidcErr.Code == ErrorCodeServerError {
			slog.Error("Authorize endpoint error", "error", err)
		}
		return utils.ErrorResponse(c, oidcErr.Code, oidcErr.StatusCode)
	}

	// Redirect to redirect_uri with code and state
	u, err := url.Parse(req.RedirectURI)
	if err != nil {
		slog.Error("Failed to parse redirect_uri", "error", err, "redirect_uri", req.RedirectURI)
		return utils.ErrorResponse(c, "invalid_redirect_uri", fiber.StatusBadRequest)
	}

	q := u.Query()
	q.Set("code", res.Code)
	if res.State != "" {
		q.Set("state", res.State)
	}
	u.RawQuery = q.Encode()

	return c.Redirect(u.String(), fiber.StatusFound)
}

// Token handles the OAuth2 token request (authorization code exchange)
func (h *Handler) Token(c *fiber.Ctx) error {
	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		slog.Error("Failed to parse token request body", "error", err)
		return utils.OIDCErrorResponse(c, ErrorCodeInvalidRequest, "malformed request body")
	}

	// Validate common required fields
	if req.GrantType == "" {
		return utils.OIDCErrorResponse(c, ErrorCodeInvalidRequest, "grant_type is required")
	}
	if req.ClientID == "" {
		return utils.OIDCErrorResponse(c, ErrorCodeInvalidRequest, "client_id is required")
	}

	switch req.GrantType {
	case "refresh_token":
		if req.RefreshToken == "" {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidRequest, "refresh_token is required")
		}

		res, err := h.service.RefreshToken(&req)
		if err != nil {
			oidcErr := MapErrorToOIDC(err)
			if oidcErr.Code == ErrorCodeServerError {
				slog.Error("Token endpoint error (refresh_token)", "error", err)
			}
			return utils.OIDCErrorResponse(c, oidcErr.Code, oidcErr.Description, oidcErr.StatusCode)
		}
		return c.Status(fiber.StatusOK).JSON(res)

	case "client_credentials":
		res, err := h.service.ClientCredentialsGrant(&req)
		if err != nil {
			oidcErr := MapErrorToOIDC(err)
			if oidcErr.Code == ErrorCodeServerError {
				slog.Error("Token endpoint error (client_credentials)", "error", err)
			}
			return utils.OIDCErrorResponse(c, oidcErr.Code, oidcErr.Description, oidcErr.StatusCode)
		}
		return c.Status(fiber.StatusOK).JSON(res)

	case "password":
		res, err := h.service.PasswordGrant(&req)
		if err != nil {
			oidcErr := MapErrorToOIDC(err)
			if oidcErr.Code == ErrorCodeServerError {
				slog.Error("Token endpoint error (password)", "error", err)
			}
			return utils.OIDCErrorResponse(c, oidcErr.Code, oidcErr.Description, oidcErr.StatusCode)
		}
		return c.Status(fiber.StatusOK).JSON(res)

	case "authorization_code":
		if req.Code == "" {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidRequest, "code is required")
		}
		if req.RedirectURI == "" {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidRequest, "redirect_uri is required")
		}

		// Get session from cookie
		sessionCookie := c.Cookies("session")
		if sessionCookie == "" {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidGrant, "Session cookie is required", fiber.StatusUnauthorized)
		}

		// Parse session cookie: format is "sessionID:secret"
		parts := strings.SplitN(sessionCookie, ":", 2)
		if len(parts) != 2 {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidGrant, "Invalid session cookie format", fiber.StatusUnauthorized)
		}

		sessionIDStr := parts[0]
		refreshSecret := parts[1]

		if sessionIDStr == "" || refreshSecret == "" {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidGrant, "Invalid session cookie", fiber.StatusUnauthorized)
		}

		// Parse session ID as UUID
		sessionID, err := uuid.Parse(sessionIDStr)
		if err != nil {
			return utils.OIDCErrorResponse(c, ErrorCodeInvalidGrant, "Invalid session ID format", fiber.StatusUnauthorized)
		}

		// Call service with existing session
		res, err := h.service.ExchangeCode(&req, sessionID, refreshSecret)
		if err != nil {
			oidcErr := MapErrorToOIDC(err)
			if oidcErr.Code == ErrorCodeServerError {
				slog.Error("Token endpoint error", "error", err)
			}
			return utils.OIDCErrorResponse(c, oidcErr.Code, oidcErr.Description, oidcErr.StatusCode)
		}

		return c.Status(fiber.StatusOK).JSON(res)

	default:
		return utils.OIDCErrorResponse(c, ErrorCodeUnsupportedGrantType, "unsupported grant_type")
	}
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
		slog.Error("UserInfo endpoint error", "error", err)
		return utils.OIDCErrorResponse(c, "server_error", "internal_server_error", fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(userInfo)
}

// ValidateAuthorization validates an OAuth2/OIDC authorization request
// This endpoint does not require authentication and is used by frontend to validate request parameters
func (h *Handler) ValidateAuthorization(c *fiber.Ctx) error {
	var req AuthorizeRequest
	if err := c.QueryParser(&req); err != nil {
		slog.Error("Failed to parse query parameters for validation", "error", err)
		return c.Status(fiber.StatusOK).JSON(&ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "Failed to parse query parameters",
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
		slog.Error("Failed to parse confirmation request body", "error", err)
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_request",
			ErrorDescription: "Failed to parse request body",
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
		oidcErr := MapErrorToOIDC(err)
		if oidcErr.Code == ErrorCodeServerError {
			slog.Error("ConfirmAuthorization endpoint error", "error", err)
		}
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            oidcErr.Code,
			ErrorDescription: oidcErr.Description,
		})
	}

	// Build redirect URI with code and state
	u, err := url.Parse(req.RedirectURI)
	if err != nil {
		slog.Error("Failed to parse redirect_uri in ConfirmAuthorization", "error", err, "redirect_uri", req.RedirectURI)
		return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
			Success:          false,
			Error:            "invalid_redirect_uri",
			ErrorDescription: "Invalid redirect_uri format",
		})
	}

	q := u.Query()
	q.Set("code", res.Code)
	if res.State != "" {
		q.Set("state", res.State)
	}

	u.RawQuery = q.Encode()

	return c.Status(fiber.StatusOK).JSON(&ConfirmAuthorizationResponse{
		Success:     true,
		RedirectURI: u.String(),
	})
}