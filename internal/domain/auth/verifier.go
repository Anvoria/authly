package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

func (ks *KeyStore) Verify(token string) (*AccessTokenClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	claims := &AccessTokenClaims{}
	_, err := parser.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		kid := token.Header["kid"].(string)
		keyPair, ok := ks.Keys[kid]
		if !ok {
			return nil, ErrUnknownKey
		}
		return keyPair.PublicKey, nil
	})

	if err != nil {
		return nil, err
	}
	return claims, nil
}
