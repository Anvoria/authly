package auth

import (
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func (ks *KeyStore) Verify(tokenString string) (*AccessTokenClaims, error) {
	// Verify and parse token using the key set
	// The library will automatically match the kid from the token header
	verifiedToken, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithKeySet(ks.KeySet, jws.WithInferAlgorithmFromKey(true)),
	)
	if err != nil {
		return nil, err
	}

	// Extract sid claim
	var sidStr string
	var sid interface{}
	if verifiedToken.Get("sid", &sid) == nil {
		if s, ok := sid.(string); ok {
			sidStr = s
		}
	}

	claims := &AccessTokenClaims{
		Sid:   sidStr,
		Token: verifiedToken,
	}

	return claims, nil
}
