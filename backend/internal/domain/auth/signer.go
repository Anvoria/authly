package auth

import (
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func (ks *KeyStore) Sign(claims *AccessTokenClaims) (string, error) {
	key, err := ks.GetActiveKey()
	if err != nil {
		return "", err
	}

	// Sign with RS256 using the token from claims
	// The key ID is already set on the key, so it will be included in the header
	signed, err := jwt.Sign(claims.Token, jwt.WithKey(jwa.RS256(), key))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}
