package services

import "GOAuTh/pkg/crypt"

func JWTClaimsValidation(claims crypt.JWTDefaultClaims) bool {
	return claims.Realm != "" && claims.UID != 0 && claims.Expire > 0
}
