package oidc

import (
	"time"

	"github.com/Anvoria/authly/internal/database"
	"github.com/google/uuid"
)

// AuthorizeRequest represents the OAuth2/OIDC authorization request
type AuthorizeRequest struct {
	ResponseType        string `query:"response_type" validate:"required,oneof=code"`
	ClientID            string `query:"client_id" validate:"required"`
	RedirectURI         string `query:"redirect_uri" validate:"required,url"`
	Scope               string `query:"scope" validate:"required"`
	State               string `query:"state"`
	Nonce               string `query:"nonce"`
	CodeChallenge       string `query:"code_challenge"`
	CodeChallengeMethod string `query:"code_challenge_method" validate:"omitempty,oneof=S256"`
}

// AuthorizeResponse represents the response from authorization
type AuthorizeResponse struct {
	Code  string `json:"code"`
	State string `json:"state,omitempty"`
}

// TokenRequest represents the OAuth2 token request
type TokenRequest struct {
	GrantType    string `form:"grant_type" validate:"required,oneof=authorization_code refresh_token password client_credentials"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id" validate:"required"`
	ClientSecret string `form:"client_secret"`
	CodeVerifier string `form:"code_verifier"`
	RefreshToken string `form:"refresh_token"`
	Scope        string `form:"scope"`
	Username     string `form:"username"`
	Password     string `form:"password"`
	UserAgent    string `form:"-"` // Populated from request header
	IPAddress    string `form:"-"` // Populated from request remote address
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// ConfirmAuthorizationRequest represents the request to confirm authorization
type ConfirmAuthorizationRequest struct {
	ClientID            string `json:"client_id" validate:"required"`
	RedirectURI         string `json:"redirect_uri" validate:"required,url"`
	ResponseType        string `json:"response_type" validate:"required,oneof=code"`
	Scope               string `json:"scope" validate:"required"`
	State               string `json:"state"`
	Nonce               string `json:"nonce"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method" validate:"omitempty,oneof=s256 S256"`
}

// ConfirmAuthorizationResponse represents the response from authorization confirmation
type ConfirmAuthorizationResponse struct {
	Success          bool   `json:"success"`
	RedirectURI      string `json:"redirect_uri,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// AuthorizationCode represents an OAuth2 authorization code
type AuthorizationCode struct {
	database.BaseModel

	Code          string    `gorm:"column:code;type:varchar(255);uniqueIndex;not null"`
	ClientID      string    `gorm:"column:client_id;type:varchar(255);not null;index"`
	UserID        uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`
	RedirectURI   string    `gorm:"column:redirect_uri;type:text;not null"`
	Scopes        string    `gorm:"column:scopes;type:text;not null"` // space-separated
	Nonce         string    `gorm:"column:nonce;type:text"`
	CodeChallenge string    `gorm:"column:code_challenge;type:varchar(255)"`
	ChallengeMeth string    `gorm:"column:challenge_meth;type:varchar(10)"`
	ExpiresAt     time.Time `gorm:"column:expires_at;not null;index"`
	Used          bool      `gorm:"column:used;default:false;index"`
}

func (AuthorizationCode) TableName() string {
	return "authorization_codes"
}
