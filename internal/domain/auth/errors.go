package auth

import "errors"

// Authentication errors
var (
	// ErrInvalidCredentials is returned when email or password is incorrect
	// during authentication attempts.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Key store errors
var (
	// ErrUnknownKey is returned when a JWT token contains a key ID (kid) that
	// is not found in the key store during token verification.
	ErrUnknownKey = errors.New("unknown key")
)

// Key loading errors
// These errors are returned by LoadKeys when there are issues loading
// cryptographic keys from the filesystem.

// ErrKeysDirectoryNotAccessible is returned when the keys directory path
// does not exist or cannot be accessed.
type ErrKeysDirectoryNotAccessible struct {
	Path string
	Err  error
}

func (e *ErrKeysDirectoryNotAccessible) Error() string {
	return "keys directory does not exist or is not accessible: " + e.Err.Error()
}

func (e *ErrKeysDirectoryNotAccessible) Unwrap() error {
	return e.Err
}

// ErrKeysPathNotDirectory is returned when the provided keys path exists
// but is not a directory.
type ErrKeysPathNotDirectory struct {
	Path string
}

func (e *ErrKeysPathNotDirectory) Error() string {
	return "keys path is not a directory: " + e.Path
}

// ErrFailedToReadKeysDirectory is returned when the keys directory cannot
// be read (e.g., permission denied).
type ErrFailedToReadKeysDirectory struct {
	Err error
}

func (e *ErrFailedToReadKeysDirectory) Error() string {
	return "failed to read keys directory: " + e.Err.Error()
}

func (e *ErrFailedToReadKeysDirectory) Unwrap() error {
	return e.Err
}

// ErrFailedToReadPrivateKeyFile is returned when a private key file cannot
// be read from the filesystem.
type ErrFailedToReadPrivateKeyFile struct {
	FileName string
	Err      error
}

func (e *ErrFailedToReadPrivateKeyFile) Error() string {
	return "failed to read private key file " + e.FileName + ": " + e.Err.Error()
}

func (e *ErrFailedToReadPrivateKeyFile) Unwrap() error {
	return e.Err
}

// ErrFailedToDecodePrivateKeyPEM is returned when a private key file's PEM
// block cannot be decoded.
type ErrFailedToDecodePrivateKeyPEM struct {
	FileName string
}

func (e *ErrFailedToDecodePrivateKeyPEM) Error() string {
	return "failed to decode PEM block from private key file: " + e.FileName
}

// ErrFailedToParsePrivateKey is returned when a private key cannot be parsed
// from PEM data, after attempting both PKCS1 and PKCS8 formats.
type ErrFailedToParsePrivateKey struct {
	FileName string
	Err      error
}

func (e *ErrFailedToParsePrivateKey) Error() string {
	return "failed to parse private key from " + e.FileName + " (tried PKCS1 and PKCS8): " + e.Err.Error()
}

func (e *ErrFailedToParsePrivateKey) Unwrap() error {
	return e.Err
}

// ErrPrivateKeyNotRSA is returned when a private key file contains a key
// that is not an RSA key.
type ErrPrivateKeyNotRSA struct {
	FileName string
}

func (e *ErrPrivateKeyNotRSA) Error() string {
	return "private key in " + e.FileName + " is not an RSA key"
}

// ErrFailedToReadPublicKeyFile is returned when a public key file cannot
// be read from the filesystem.
type ErrFailedToReadPublicKeyFile struct {
	FileName string
	Err      error
}

func (e *ErrFailedToReadPublicKeyFile) Error() string {
	return "failed to read public key file " + e.FileName + ": " + e.Err.Error()
}

func (e *ErrFailedToReadPublicKeyFile) Unwrap() error {
	return e.Err
}

// ErrFailedToDecodePublicKeyPEM is returned when a public key file's PEM
// block cannot be decoded.
type ErrFailedToDecodePublicKeyPEM struct {
	FileName string
}

func (e *ErrFailedToDecodePublicKeyPEM) Error() string {
	return "failed to decode PEM block from public key file: " + e.FileName
}

// ErrFailedToParsePublicKey is returned when a public key cannot be parsed
// from PEM data.
type ErrFailedToParsePublicKey struct {
	FileName string
	Err      error
}

func (e *ErrFailedToParsePublicKey) Error() string {
	return "failed to parse public key from " + e.FileName + ": " + e.Err.Error()
}

func (e *ErrFailedToParsePublicKey) Unwrap() error {
	return e.Err
}

// ErrPublicKeyNotRSA is returned when a public key file contains a key
// that is not an RSA key.
type ErrPublicKeyNotRSA struct {
	FileName string
}

func (e *ErrPublicKeyNotRSA) Error() string {
	return "public key in " + e.FileName + " is not an RSA key"
}

// Middleware errors
var (
	// ErrMissingAuthorizationHeader is returned when the Authorization header is missing
	ErrMissingAuthorizationHeader = errors.New("missing authorization header")

	// ErrInvalidAuthorizationHeader is returned when the Authorization header format is invalid
	ErrInvalidAuthorizationHeader = errors.New("invalid authorization header")

	// ErrMissingToken is returned when the token is missing from the Authorization header
	ErrMissingToken = errors.New("missing token")

	// ErrInvalidToken is returned when the token cannot be verified
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpiredOrInvalid is returned when the token is expired or invalid
	ErrTokenExpiredOrInvalid = errors.New("token expired or invalid")

	// ErrTokenValidationError is returned when there's an error validating the token
	ErrTokenValidationError = errors.New("token validation error")

	// ErrTokenRevoked is returned when the token has been revoked
	ErrTokenRevoked = errors.New("token revoked")

	// ErrUnauthorized is returned when the user is not authorized
	ErrUnauthorized = errors.New("unauthorized")
)

// Handler errors
var (
	// ErrInvalidBody is returned when the request body is invalid
	ErrInvalidBody = errors.New("invalid body")
)
