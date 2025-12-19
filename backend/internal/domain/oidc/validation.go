package oidc

import (
	"encoding/base64"
	"errors"
	"slices"
	"strings"

	"gorm.io/gorm"
)

// ValidateAuthorizationRequest validates an OAuth2/OIDC authorization request without requiring authentication
// Returns a response indicating if the request is valid and includes client information if valid
func (s *Service) ValidateAuthorizationRequest(req *AuthorizeRequest) *ValidateAuthorizationRequestResponse {
	// Validate response_type
	if req.ResponseType == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "response_type is required",
		}
	}
	if req.ResponseType != "code" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "unsupported_response_type",
			ErrorDescription: "Only 'code' response_type is supported",
		}
	}

	// Validate client_id
	if req.ClientID == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "client_id is required",
		}
	}

	// Find service by client_id
	service, err := s.serviceRepo.FindByClientID(req.ClientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ValidateAuthorizationRequestResponse{
				Valid:            false,
				Error:            "invalid_client",
				ErrorDescription: "Client not found",
			}
		}
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "server_error",
			ErrorDescription: "Failed to validate client",
		}
	}

	// Check if service is active
	if !service.Active {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "unauthorized_client",
			ErrorDescription: "Client is not active",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}

	// Validate redirect_uri
	if req.RedirectURI == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "redirect_uri is required",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}
	if !s.isValidRedirectURI(service.RedirectURIs, req.RedirectURI) {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_redirect_uri",
			ErrorDescription: "The redirect_uri is not allowed for this client",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}

	// Validate scopes
	if req.Scope == "" {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_request",
			ErrorDescription: "scope is required",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}
	requestedScopes := strings.Fields(req.Scope)
	if !s.isValidScopes(service.AllowedScopes, requestedScopes) {
		return &ValidateAuthorizationRequestResponse{
			Valid:            false,
			Error:            "invalid_scope",
			ErrorDescription: "One or more requested scopes are not allowed",
			Client: &ClientInfo{
				ID:            service.ID.String(),
				Name:          service.Name,
				RedirectURIs:  service.RedirectURIs,
				AllowedScopes: service.AllowedScopes,
				Active:        service.Active,
			},
		}
	}

	// Validate PKCE if provided
	if req.CodeChallenge != "" {
		if err := s.validatePKCE(req.CodeChallenge, req.CodeChallengeMethod); err != nil {
			var errorCode string
			var errorDesc string
			switch err {
			case ErrInvalidCodeChallenge:
				errorCode = "invalid_code_challenge"
				errorDesc = "Invalid code_challenge format"
			case ErrInvalidCodeChallengeMethod:
				errorCode = "unsupported_code_challenge_method"
				errorDesc = "Only 's256' or 'S256' code_challenge_method is supported"
			default:
				errorCode = "invalid_request"
				errorDesc = err.Error()
			}
			return &ValidateAuthorizationRequestResponse{
				Valid:            false,
				Error:            errorCode,
				ErrorDescription: errorDesc,
				Client: &ClientInfo{
					ID:            service.ID.String(),
					Name:          service.Name,
					RedirectURIs:  service.RedirectURIs,
					AllowedScopes: service.AllowedScopes,
					Active:        service.Active,
				},
			}
		}
	}

	// All validations passed
	return &ValidateAuthorizationRequestResponse{
		Valid: true,
		Client: &ClientInfo{
			ID:            service.ID.String(),
			Name:          service.Name,
			RedirectURIs:  service.RedirectURIs,
			AllowedScopes: service.AllowedScopes,
			Active:        service.Active,
		},
	}
}

// isValidRedirectURI checks if the redirect_uri is allowed for the service
func (s *Service) isValidRedirectURI(allowedURIs []string, redirectURI string) bool {
	return slices.Contains(allowedURIs, redirectURI)
}

// isValidScopes checks if all requested scopes are allowed
func (s *Service) isValidScopes(allowedScopes []string, requestedScopes []string) bool {
	allowedMap := make(map[string]bool)
	for _, scope := range allowedScopes {
		allowedMap[scope] = true
	}

	for _, scope := range requestedScopes {
		if !allowedMap[scope] {
			return false
		}
	}
	return true
}

// validatePKCE validates the PKCE parameters
func (s *Service) validatePKCE(codeChallenge, codeChallengeMethod string) error {
	if codeChallenge == "" {
		return ErrInvalidCodeChallenge
	}

	if codeChallengeMethod != "s256" && codeChallengeMethod != "S256" && codeChallengeMethod != "" {
		return ErrInvalidCodeChallengeMethod
	}

	// Validate code_challenge format (base64url encoded SHA256 hash)
	// Should be 43 characters (base64url encoded 32-byte hash)
	if len(codeChallenge) != 43 {
		return ErrInvalidCodeChallenge
	}

	_, err := base64.RawURLEncoding.DecodeString(codeChallenge)
	if err != nil {
		return ErrInvalidCodeChallenge
	}

	return nil
}
