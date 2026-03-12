package services

import "github.com/monkeydioude/goauth/v2/pkg/crypt"

func JWTClaimsValidation(claims crypt.JWTDefaultClaims) bool {
	return claims.Realm != "" && claims.UID != 0 && claims.Expire > 0
}
