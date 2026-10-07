package functional

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/jwt"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// loginAccessToken logs the user in and returns the access token and the session it belongs to.
func loginAccessToken(t *testing.T, layout *handlers.Layout, login string) (string, entities.Session) {
	accessCookie, _, err := services.AuthLogin(entities.NewUser(login, "test", login), services.ClientInfo{}, layout.DB, layout.UserParams, layout.AccessTokenFactory, layout.RefreshTokenFactory, layout.MaxActiveSessions)
	assert.NoError(t, err)
	accessToken := strings.TrimPrefix(accessCookie.Value, "Bearer ")
	decoded, err := layout.AccessTokenFactory.DecodeToken(accessToken)
	assert.NoError(t, err)
	return accessToken, findSessionRow(t, layout.DB, decoded.Claims.SID)
}

// statusOverHTTP checks an access token through the JSON API.
func statusOverHTTP(t *testing.T, layout *handlers.Layout, accessToken string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/jwt/status", layout.Get(jwt.Status))
	req, err := http.NewRequest("GET", "/v1/jwt/status", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: consts.AuthorizationCookie, Value: "Bearer " + accessToken})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestStatusRefusesARevokedSessionAtOnce(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestStatusRefusesARevokedSessionAtOnce@test.com"
	newLoginUser(t, gormDB, login)
	revokedToken, revokedSession := loginAccessToken(t, layout, login)
	otherToken, _ := loginAccessToken(t, layout, login)
	conn := setupRPC(t, layout)
	defer conn.Close()
	assert.Equal(t, 200, statusOverHTTP(t, layout, revokedToken).Code)

	assert.NoError(t, services.RevokeSessions(gormDB, entities.SessionRevokedByUser, layout.RefreshTokenFactory.TimeFn(), "id = ?", revokedSession.ID))

	// over HTTP, the very next check refuses the revoked session's access token
	rec := statusOverHTTP(t, layout, revokedToken)
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, rec))

	// over gRPC too, with the code sb_back maps to unauthorized
	_, err := v1.NewJWTClient(conn).Status(context.Background(), &v1.StatusIn{AccessToken: revokedToken})
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Code(401), st.Code())
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, st.Message())

	// the user's other session still passes
	assert.Equal(t, 200, statusOverHTTP(t, layout, otherToken).Code)
	_, err = v1.NewJWTClient(conn).Status(context.Background(), &v1.StatusIn{AccessToken: otherToken})
	assert.NoError(t, err)
}

func TestStatusRefusesADeactivatedUser(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestStatusRefusesADeactivatedUser@test.com"
	user := newLoginUser(t, gormDB, login)
	accessToken, _ := loginAccessToken(t, layout, login)

	assert.NoError(t, services.AuthDeactivate(user.ID, gormDB, layout.RefreshTokenFactory.TimeFn()))

	rec := statusOverHTTP(t, layout, accessToken)
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, rec))
}

func TestStatusRefusesAnExpiredSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestStatusRefusesAnExpiredSession@test.com"
	newLoginUser(t, gormDB, login)
	accessToken, session := loginAccessToken(t, layout, login)

	// the access token is still valid, its session is not
	assert.NoError(t, gormDB.Model(&session).Update("expires_at", layout.AccessTokenFactory.TimeFn().Add(-time.Second)).Error)

	rec := statusOverHTTP(t, layout, accessToken)
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, rec))
}

func TestStatusRefusesTokensThatAreNotSessionAccessTokens(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestStatusRefusesTokensThatAreNotSessionAccessTokens@test.com"
	newLoginUser(t, gormDB, login)
	refreshToken, session := loginSession(t, layout, login)

	// a refresh token is not an access token
	rec := statusOverHTTP(t, layout, refreshToken)
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_WRONG_TOKEN_TYPE, errorMessage(t, rec))

	// an access token naming no session, as issued before sessions existed
	sessionless, err := layout.AccessTokenFactory.GenerateToken(crypt.JWTDefaultClaims{UID: session.UserID, Realm: login})
	assert.NoError(t, err)
	rec = statusOverHTTP(t, layout, sessionless.Token)
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_MISSING_PARAMS, errorMessage(t, rec))

	// an access token naming a session that does not exist
	unknown, err := layout.AccessTokenFactory.GenerateToken(crypt.JWTDefaultClaims{UID: session.UserID, Realm: login, SID: uuid.NewString()})
	assert.NoError(t, err)
	rec = statusOverHTTP(t, layout, unknown.Token)
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, rec))
}
