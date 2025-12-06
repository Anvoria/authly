package auth

import "github.com/golang-jwt/jwt/v5"

func (ks *KeyStore) Sign(claims *AccessTokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = ks.ActiveKid
	return token.SignedString(ks.GetActiveKey().PrivateKey)
}
