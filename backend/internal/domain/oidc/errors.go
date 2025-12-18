package oidc

import "errors"

var (
	// ErrInvalidClientID is returned when client_id is invalid or not found
	ErrInvalidClientID = errors.New("invalid_client_id")

	// ErrInvalidRedirectURI is returned when redirect_uri is not allowed for the client
	ErrInvalidRedirectURI = errors.New("invalid_redirect_uri")

	// ErrInvalidScope is returned when requested scopes are not allowed
	ErrInvalidScope = errors.New("invalid_scope")

	// ErrInvalidResponseType is returned when response_type is not supported
	ErrInvalidResponseType = errors.New("unsupported_response_type")

	// ErrInvalidCodeChallenge is returned when code_challenge is invalid
	ErrInvalidCodeChallenge = errors.New("invalid_code_challenge")

	// ErrInvalidCodeChallengeMethod is returned when code_challenge_method is not supported
	ErrInvalidCodeChallengeMethod = errors.New("unsupported_code_challenge_method")

	// ErrClientNotActive is returned when client is not active
	ErrClientNotActive = errors.New("client_not_active")

	// ErrUnauthorizedClient is returned when client is not authorized
	ErrUnauthorizedClient = errors.New("unauthorized_client")

	// ErrInvalidGrant is returned when grant_type is invalid or authorization code is invalid
	ErrInvalidGrant = errors.New("invalid_grant")

	// ErrInvalidCode is returned when authorization code is invalid, expired, or already used
	ErrInvalidCode = errors.New("invalid_code")

	// ErrInvalidCodeVerifier is returned when code_verifier doesn't match code_challenge
	ErrInvalidCodeVerifier = errors.New("invalid_code_verifier")

	// ErrInvalidClientSecret is returned when client_secret is invalid
	ErrInvalidClientSecret = errors.New("invalid_client_secret")
)
