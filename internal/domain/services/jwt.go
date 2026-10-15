package services

import (
	"context"
	stdErr "errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetTokenFromBearer(tokenWithBearer string) (string, error) {
	parts := strings.Split(tokenWithBearer, " ")
	partsLen := len(parts)
	if partsLen == 0 || partsLen != 2 || parts[0] != "Bearer" {
		return "", errors.Unauthorized(stdErr.New(consts.ERR_WRONG_TOKEN_SCHEMA))
	}
	return parts[1], nil
}

// AuthenticateBearer checks a "Bearer {token}" value with AuthenticateAccessToken.
func AuthenticateBearer(ctx context.Context, tokenWithBearer string, factory JWTFactory) (entities.JWT[crypt.JWTDefaultClaims], error) {
	token, err := GetTokenFromBearer(tokenWithBearer)
	if err != nil {
		return entities.JWT[crypt.JWTDefaultClaims]{}, err
	}
	return AuthenticateAccessToken(ctx, token, factory)
}

// SessionOfToken reads which session a token belongs to: signed, of the factory's
// type, naming a session. Its expiry is not checked.
func SessionOfToken(token string, factory JWTFactory) (entities.JWT[crypt.JWTDefaultClaims], error) {
	var none entities.JWT[crypt.JWTDefaultClaims]
	jwt, err := factory.DecodeToken(token)
	if err != nil {
		return none, err
	}
	if jwt.Claims.Type != factory.Type {
		return none, errors.Unauthorized(stdErr.New(consts.ERR_WRONG_TOKEN_TYPE))
	}
	if _, err := uuid.Parse(jwt.Claims.SID); err != nil || !JWTClaimsValidation(jwt.Claims) {
		return none, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_MISSING_PARAMS))
	}
	return jwt, nil
}

// AuthenticateAccessToken validates an access token: signed, of the access type,
// not expired, and belonging to a session that is still active.
func AuthenticateAccessToken(ctx context.Context, token string, factory JWTFactory) (entities.JWT[crypt.JWTDefaultClaims], error) {
	var none entities.JWT[crypt.JWTDefaultClaims]
	jwt, err := SessionOfToken(token, factory)
	if err != nil {
		return none, err
	}
	claims := jwt.Claims
	now := factory.TimeFn()
	if claims.Expire < now.Unix() {
		slog.InfoContext(ctx, "access token rejected: expired", "uid", claims.UID, "realm", claims.Realm, "sid", claims.SID, "expired_since", now.Sub(time.Unix(claims.Expire, 0)).String())
		return none, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_EXPIRED))
	}
	// no checker means no way to know: refuse
	if factory.RevocationCheckerFn == nil {
		slog.ErrorContext(ctx, "access token rejected: no revocation checker", "uid", claims.UID, "realm", claims.Realm, "sid", claims.SID)
		return none, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_REVOKED))
	}
	revoked, err := factory.RevocationCheckerFn(claims, now)
	if err != nil {
		slog.ErrorContext(ctx, "access token rejected: revocation check failed", "uid", claims.UID, "realm", claims.Realm, "sid", claims.SID, "error", err.Error())
		return none, errors.DBError(err)
	}
	if revoked {
		slog.InfoContext(ctx, "access token rejected: session revoked", "uid", claims.UID, "realm", claims.Realm, "sid", claims.SID)
		return none, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_REVOKED))
	}
	return jwt, nil
}

// JWTStatus checks an access token (AuthenticateAccessToken) and hands it back as a cookie.
func JWTStatus(ctx context.Context, token string, factory JWTFactory) (http.Cookie, error) {
	jwt, err := AuthenticateAccessToken(ctx, token, factory)
	if err != nil {
		return http.Cookie{}, err
	}
	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  "Bearer " + jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}

// RefreshResult is what a refresh hands back. RefreshToken is nil when a parallel
// refresh already rotated the token: the client keeps the refresh token it got from it.
type RefreshResult struct {
	AccessToken    http.Cookie
	RefreshToken   *http.Cookie
	SessionExpires time.Time
}

// JWTRefresh rotates the refresh token of one session: the one its sid names.
// The user's other sessions are never read or written.
func JWTRefresh(
	token string,
	client ClientInfo,
	accessTokenFactory JWTFactory,
	refreshTokenFactory JWTFactory,
	reuseGrace time.Duration,
	db *gorm.DB,
) (RefreshResult, error) {
	jwt, err := refreshTokenFactory.DecodeToken(token)
	if err != nil {
		return RefreshResult{}, errors.Unauthorized(err)
	}
	// an access token must not rotate, nor revoke, the session
	if jwt.Claims.Type != refreshTokenFactory.Type {
		return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_WRONG_TOKEN_TYPE))
	}
	sid, err := uuid.Parse(jwt.Claims.SID)
	if err != nil || !JWTClaimsValidation(jwt.Claims) {
		return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_MISSING_PARAMS))
	}
	// a deactivated or deleted user can't refresh anymore
	active, err := isUserActive(db, jwt.Claims.UID)
	if err != nil {
		return RefreshResult{}, errors.DBError(err)
	}
	if !active {
		return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_REVOKED))
	}
	newAT, err := accessTokenFactory.GenerateToken(jwt.Claims)
	if err != nil {
		return RefreshResult{}, errors.Unauthorized(err)
	}
	newRT, err := refreshTokenFactory.TryRefresh(jwt)
	if err != nil {
		return RefreshResult{}, errors.Unauthorized(err)
	}
	attempt := refreshAttempt{
		sessionID: sid,
		userID:    jwt.Claims.UID,
		tokenHash: crypt.HashToken(token),
		client:    client,
		now:       refreshTokenFactory.TimeFn(),
	}
	expiresAt := time.Unix(newRT.Claims.Expire, 0)
	rotated, err := rotateSession(db, attempt, crypt.HashToken(newRT.GetToken()), expiresAt)
	if err != nil {
		return RefreshResult{}, errors.DBError(err)
	}
	if !rotated {
		return refreshWithoutRotation(db, attempt, newAT, reuseGrace)
	}
	refreshCookie := refreshTokenCookie(newRT)
	return RefreshResult{
		AccessToken:    accessTokenCookie(newAT),
		RefreshToken:   &refreshCookie,
		SessionExpires: expiresAt,
	}, nil
}

// refreshWithoutRotation answers a refresh whose token is not its session's current one.
func refreshWithoutRotation(
	db *gorm.DB,
	attempt refreshAttempt,
	newAT entities.JWT[crypt.JWTDefaultClaims],
	reuseGrace time.Duration,
) (RefreshResult, error) {
	session, err := findSession(db, attempt)
	if err != nil {
		return RefreshResult{}, errors.DBError(err)
	}
	switch refreshOutcomeOf(session, attempt.tokenHash, attempt.now, reuseGrace) {
	case refreshInGrace:
		touched, err := touchSession(db, attempt)
		if err != nil {
			return RefreshResult{}, errors.DBError(err)
		}
		// revoked between the read and the touch
		if !touched {
			return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_REVOKED))
		}
		return RefreshResult{AccessToken: accessTokenCookie(newAT), SessionExpires: session.ExpiresAt}, nil
	case refreshExpired:
		return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_EXPIRED))
	case refreshReused:
		if err := RevokeSessions(db, entities.SessionRevokedReuseDetected, attempt.now, "id = ?", attempt.sessionID); err != nil {
			return RefreshResult{}, errors.DBError(err)
		}
		slog.WarnContext(logs.DBContext(db), "refresh token reused: session revoked", "uid", attempt.userID, "sid", attempt.sessionID)
		return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_REUSED))
	default:
		return RefreshResult{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_REVOKED))
	}
}

func accessTokenCookie(jwt entities.JWT[crypt.JWTDefaultClaims]) http.Cookie {
	return http.Cookie{
		Name:    consts.AuthorizationCookie,
		Value:   "Bearer " + jwt.GetToken(),
		Expires: time.Now().Add(jwt.GetExpiresIn()),
		MaxAge:  int(jwt.GetExpiresIn().Seconds()),
		Path:    "/",
	}
}

func refreshTokenCookie(jwt entities.JWT[crypt.JWTDefaultClaims]) http.Cookie {
	return http.Cookie{
		Name:    consts.RefreshTokenCookie,
		Value:   jwt.GetToken(),
		Expires: time.Now().Add(jwt.GetExpiresIn()),
		MaxAge:  int(jwt.GetExpiresIn().Seconds()),
		Path:    "/",
	}
}
