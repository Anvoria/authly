package oidc

import (
	"errors"
	"net/http"
)

// Standard OIDC error codes as defined in RFC 6749 and OIDC Core 1.0
const (
	ErrorCodeInvalidRequest          = "invalid_request"
	ErrorCodeInvalidClient           = "invalid_client"
	ErrorCodeInvalidGrant            = "invalid_grant"
	ErrorCodeUnauthorizedClient      = "unauthorized_client"
	ErrorCodeUnsupportedResponseType = "unsupported_response_type"
	ErrorCodeUnsupportedGrantType    = "unsupported_grant_type"
	ErrorCodeInvalidScope            = "invalid_scope"
	ErrorCodeServerError             = "server_error"
	ErrorCodeLoginRequired           = "login_required"
	ErrorCodeInvalidRequestURI       = "invalid_request_uri"
	ErrorCodeInteractionRequired     = "interaction_required"
)

var (
	// ErrInvalidClientID is returned when the client_id provided in the request is invalid, unknown, or malformed.
	ErrInvalidClientID = errors.New("invalid_client_id")

	// ErrInvalidRedirectURI is returned when the redirect_uri provided does not match the pre-registered value.
	ErrInvalidRedirectURI = errors.New("invalid_redirect_uri")

	// ErrInvalidScope is returned when the requested scope is invalid, unknown, or malformed.
	ErrInvalidScope = errors.New("invalid_scope")

	// ErrInvalidResponseType is returned when the response_type is not supported by the authorization server.
	ErrInvalidResponseType = errors.New("unsupported_response_type")

	// ErrUnsupportedGrantType is returned when the grant_type is not supported.
	ErrUnsupportedGrantType = errors.New("unsupported_grant_type")

	// ErrInvalidCodeChallenge is returned when the code_challenge is missing or invalid.
	ErrInvalidCodeChallenge = errors.New("invalid_code_challenge")

	// ErrInvalidCodeChallengeMethod is returned when the code_challenge_method is not supported (only S256 is supported).
	ErrInvalidCodeChallengeMethod = errors.New("unsupported_code_challenge_method")

	// ErrClientNotActive is returned when the client application has been disabled or suspended.
	ErrClientNotActive = errors.New("client_not_active")

	// ErrUnauthorizedClient is returned when the client is not authorized to use this authorization grant type.
	ErrUnauthorizedClient = errors.New("unauthorized_client")

	// ErrInvalidGrant is returned when the provided authorization grant (e.g., authorization code) or refresh token is invalid, expired, revoked, does not match the redirection URI used in the authorization request, or was issued to another client.
	ErrInvalidGrant = errors.New("invalid_grant")

	// ErrInvalidCode is returned specifically when the authorization code is invalid, expired, or already used.
	ErrInvalidCode = errors.New("invalid_code")

	// ErrInvalidCodeVerifier is returned when the PKCE code_verifier does not match the code_challenge associated with the authorization code.
	ErrInvalidCodeVerifier = errors.New("invalid_code_verifier")

	// ErrInvalidClientSecret is returned when the client authentication fails (e.g., invalid client_secret).
	ErrInvalidClientSecret = errors.New("invalid_client_secret")
)

// OIDCError represents a standardized OIDC protocol error.
type OIDCError struct {
	// Code is the error code to be returned to the client (e.g., "invalid_request").
	Code string
	// Description is a human-readable ASCII text providing additional information.
	Description string
	// StatusCode is the HTTP status code associated with this error.
	StatusCode int
}

// MapErrorToOIDC maps an internal domain error to a standardized OIDC protocol error.
// MapErrorToOIDC maps internal domain errors to their corresponding OIDC error responses.
// It returns an OIDCError containing the standard OIDC error code, a client-facing description, and an HTTP status code; unrecognized errors map to `server_error` with HTTP 500.
func MapErrorToOIDC(err error) OIDCError {
	switch err {
	case ErrInvalidClientID:
		return OIDCError{Code: ErrorCodeInvalidClient, Description: "Invalid client_id", StatusCode: http.StatusBadRequest}
	case ErrInvalidRedirectURI:
		return OIDCError{Code: ErrorCodeInvalidRequest, Description: "The redirect_uri is not allowed for this client", StatusCode: http.StatusBadRequest}
	case ErrInvalidScope:
		return OIDCError{Code: ErrorCodeInvalidScope, Description: "One or more requested scopes are not allowed", StatusCode: http.StatusBadRequest}
	case ErrInvalidResponseType:
		return OIDCError{Code: ErrorCodeUnsupportedResponseType, Description: "Only 'code' response_type is supported", StatusCode: http.StatusBadRequest}
	case ErrUnsupportedGrantType:
		return OIDCError{Code: ErrorCodeUnsupportedGrantType, Description: "The authorization grant type is not supported", StatusCode: http.StatusBadRequest}
	case ErrInvalidCodeChallenge:
		return OIDCError{Code: ErrorCodeInvalidRequest, Description: "Invalid code_challenge format", StatusCode: http.StatusBadRequest}
	case ErrInvalidCodeChallengeMethod:
		return OIDCError{Code: ErrorCodeInvalidRequest, Description: "Only 's256' code_challenge_method is supported", StatusCode: http.StatusBadRequest}
	case ErrClientNotActive:
		return OIDCError{Code: ErrorCodeUnauthorizedClient, Description: "Client is not active", StatusCode: http.StatusUnauthorized}
	case ErrUnauthorizedClient:
		return OIDCError{Code: ErrorCodeUnauthorizedClient, Description: "Client is not authorized", StatusCode: http.StatusUnauthorized}
	case ErrInvalidGrant:
		return OIDCError{Code: ErrorCodeInvalidGrant, Description: "The provided grant is invalid", StatusCode: http.StatusBadRequest}
	case ErrInvalidCode:
		return OIDCError{Code: ErrorCodeInvalidGrant, Description: "The provided authorization code is invalid, expired, or already used", StatusCode: http.StatusBadRequest}
	case ErrInvalidCodeVerifier:
		return OIDCError{Code: ErrorCodeInvalidGrant, Description: "code_verifier is invalid", StatusCode: http.StatusBadRequest}
	case ErrInvalidClientSecret:
		return OIDCError{Code: ErrorCodeInvalidClient, Description: "Invalid client_secret", StatusCode: http.StatusUnauthorized}
	default:
		return OIDCError{Code: ErrorCodeServerError, Description: "internal_server_error", StatusCode: http.StatusInternalServerError}
	}
}