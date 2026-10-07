package functional

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"

	"github.com/stretchr/testify/assert"
)

// deviceLogin is one login of a user: its tokens and the session it created.
type deviceLogin struct {
	accessToken  string
	refreshToken string
	session      entities.Session
}

// loginDevice logs the user in from a known client.
func loginDevice(t *testing.T, layout *handlers.Layout, login string, client services.ClientInfo) deviceLogin {
	accessCookie, refreshCookie, err := services.AuthLogin(entities.NewUser(login, "test", login), client, layout.DB, layout.UserParams, layout.AccessTokenFactory, layout.RefreshTokenFactory, layout.MaxActiveSessions)
	assert.NoError(t, err)
	accessToken := strings.TrimPrefix(accessCookie.Value, "Bearer ")
	decoded, err := layout.AccessTokenFactory.DecodeToken(accessToken)
	assert.NoError(t, err)
	return deviceLogin{
		accessToken:  accessToken,
		refreshToken: refreshCookie.Value,
		session:      findSessionRow(t, layout.DB, decoded.Claims.SID),
	}
}

// withAccessToken makes a call as the user the access token belongs to.
func withAccessToken(accessToken string) context.Context {
	return rpc.SetOutgoingCookie(context.Background(), http.Cookie{Name: consts.AuthorizationCookie, Value: "Bearer " + accessToken})
}

func setLastConnection(t *testing.T, layout *handlers.Layout, sid string, at time.Time) {
	assert.NoError(t, layout.DB.Model(&entities.Session{}).Where("id = ?", sid).Update("last_connection", at).Error)
}

func TestRPCSessionListShowsTheCallersSessions(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCSessionListShowsTheCallersSessions@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.NewClientInfo("203.0.113.7", "Mozilla/5.0 (laptop)"))
	phone := loginDevice(t, layout, login, services.NewClientInfo("198.51.100.4", "Mozilla/5.0 (phone)"))
	revoked := loginDevice(t, layout, login, services.ClientInfo{})
	expired := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestRPCSessionListLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	loginDevice(t, layout, otherLogin, services.ClientInfo{})
	now := layout.AccessTokenFactory.TimeFn()
	setLastConnection(t, layout, laptop.session.ID.String(), now.Add(-2*time.Hour))
	setLastConnection(t, layout, phone.session.ID.String(), now.Add(-1*time.Hour))
	assert.NoError(t, services.RevokeSessions(gormDB, entities.SessionRevokedByUser, now.Add(-30*time.Minute), "id = ?", revoked.session.ID))
	assert.NoError(t, gormDB.Model(&entities.Session{}).Where("id = ?", expired.session.ID).Update("expires_at", now.Add(-time.Second)).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewSessionClient(conn)

	// active sessions only, most recently used first
	res, err := client.List(withAccessToken(laptop.accessToken), &v1.ListSessionsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	assert.Len(t, res.Sessions, 2)
	assert.Equal(t, phone.session.ID.String(), res.Sessions[0].Id)
	assert.False(t, res.Sessions[0].Current)
	assert.Equal(t, "198.51.100.4", res.Sessions[0].LoginIp)
	assert.Equal(t, "Mozilla/5.0 (phone)", res.Sessions[0].UserAgent)
	assert.Equal(t, laptop.session.ID.String(), res.Sessions[1].Id)
	assert.True(t, res.Sessions[1].Current)
	assert.Equal(t, "203.0.113.7", res.Sessions[1].LoginIp)
	assert.Equal(t, "203.0.113.7", res.Sessions[1].LastIp)
	assert.Equal(t, now.Add(-2*time.Hour).Unix(), res.Sessions[1].LastConnection.AsTime().Unix())
	assert.Equal(t, laptop.session.ExpiresAt.Unix(), res.Sessions[1].ExpiresAt.AsTime().Unix())
	assert.Nil(t, res.Sessions[1].RevokedAt)
	assert.Empty(t, res.Sessions[1].RevokedReason)

	// include_revoked adds the revoked and the expired session
	res, err = client.List(withAccessToken(laptop.accessToken), &v1.ListSessionsRequest{IncludeRevoked: true})
	assert.NoError(t, err)
	assert.Len(t, res.Sessions, 4)
	byID := map[string]*v1.SessionInfo{}
	for _, info := range res.Sessions {
		byID[info.Id] = info
	}
	assert.Equal(t, now.Add(-30*time.Minute).Unix(), byID[revoked.session.ID.String()].RevokedAt.AsTime().Unix())
	assert.Equal(t, entities.SessionRevokedByUser, byID[revoked.session.ID.String()].RevokedReason)
	assert.Equal(t, now.Add(-time.Second).Unix(), byID[expired.session.ID.String()].ExpiresAt.AsTime().Unix())
	assert.Nil(t, byID[expired.session.ID.String()].RevokedAt)
}

func TestRPCSessionCallsNeedAnActiveAccessToken(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCSessionCallsNeedAnActiveAccessToken@test.com"
	newLoginUser(t, gormDB, login)
	device := loginDevice(t, layout, login, services.ClientInfo{})
	revoked := loginDevice(t, layout, login, services.ClientInfo{})
	assert.NoError(t, services.RevokeSessions(gormDB, entities.SessionRevokedByUser, layout.AccessTokenFactory.TimeFn(), "id = ?", revoked.session.ID))
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewSessionClient(conn)

	// no access token
	res, err := client.List(context.Background(), &v1.ListSessionsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	assert.Empty(t, res.Sessions)

	// a refresh token is not an access token
	res, err = client.List(withAccessToken(device.refreshToken), &v1.ListSessionsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	assert.Equal(t, consts.ERR_WRONG_TOKEN_TYPE, res.Message)

	// the access token of a revoked session
	res, err = client.List(withAccessToken(revoked.accessToken), &v1.ListSessionsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, res.Message)

	// not a token at all
	revokeRes, err := client.Revoke(withAccessToken("not-a-token"), &v1.RevokeSessionRequest{SessionId: device.session.ID.String()})
	assert.NoError(t, err)
	assert.Equal(t, int32(400), revokeRes.Code)
	revokeRes, err = client.RevokeAll(withAccessToken(revoked.accessToken), &v1.RevokeAllSessionsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), revokeRes.Code)
	assert.False(t, findSessionRow(t, gormDB, device.session.ID.String()).DeletedAt.Valid)
}

func TestRPCSessionRevokeEndsOneSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCSessionRevokeEndsOneSession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestRPCSessionRevokeLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewSessionClient(conn)

	res, err := client.Revoke(withAccessToken(laptop.accessToken), &v1.RevokeSessionRequest{SessionId: phone.session.ID.String()})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	stored := findSessionRow(t, gormDB, phone.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedByUser), stored.RevokedReason)
	assert.Equal(t, 401, statusOverHTTP(t, layout, phone.accessToken).Code)
	assert.Equal(t, 401, refreshOverHTTP(t, layout, phone.refreshToken).Code)
	// the caller's session goes on
	assert.Equal(t, 200, statusOverHTTP(t, layout, laptop.accessToken).Code)

	// already revoked
	res, err = client.Revoke(withAccessToken(laptop.accessToken), &v1.RevokeSessionRequest{SessionId: phone.session.ID.String()})
	assert.NoError(t, err)
	assert.Equal(t, int32(404), res.Code)
	assert.Equal(t, consts.ERR_SESSION_NOT_FOUND, res.Message)

	// another user's session is not found, and left alone
	res, err = client.Revoke(withAccessToken(laptop.accessToken), &v1.RevokeSessionRequest{SessionId: other.session.ID.String()})
	assert.NoError(t, err)
	assert.Equal(t, int32(404), res.Code)
	assert.False(t, findSessionRow(t, gormDB, other.session.ID.String()).DeletedAt.Valid)

	res, err = client.Revoke(withAccessToken(laptop.accessToken), &v1.RevokeSessionRequest{SessionId: "not-a-session-id"})
	assert.NoError(t, err)
	assert.Equal(t, int32(404), res.Code)
}

func TestRPCSessionRevokeAllCanKeepTheCurrentSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCSessionRevokeAllCanKeepTheCurrentSession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	tablet := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestRPCSessionRevokeAllLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewSessionClient(conn)

	res, err := client.RevokeAll(withAccessToken(laptop.accessToken), &v1.RevokeAllSessionsRequest{KeepCurrent: true})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	for _, device := range []deviceLogin{phone, tablet} {
		stored := findSessionRow(t, gormDB, device.session.ID.String())
		assert.True(t, stored.DeletedAt.Valid)
		assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogoutAll), stored.RevokedReason)
		assert.Equal(t, 401, refreshOverHTTP(t, layout, device.refreshToken).Code)
	}
	assert.False(t, findSessionRow(t, gormDB, laptop.session.ID.String()).DeletedAt.Valid)
	assert.Equal(t, 200, statusOverHTTP(t, layout, laptop.accessToken).Code)
	assert.False(t, findSessionRow(t, gormDB, other.session.ID.String()).DeletedAt.Valid)
}

func TestRPCSessionRevokeAllEndsEverySession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCSessionRevokeAllEndsEverySession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewSessionClient(conn)

	res, err := client.RevokeAll(withAccessToken(laptop.accessToken), &v1.RevokeAllSessionsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	for _, device := range []deviceLogin{laptop, phone} {
		stored := findSessionRow(t, gormDB, device.session.ID.String())
		assert.True(t, stored.DeletedAt.Valid)
		assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogoutAll), stored.RevokedReason)
	}
	assert.Equal(t, 401, statusOverHTTP(t, layout, laptop.accessToken).Code)
}

func TestRPCLogoutWithAnAccessTokenEndsItsSessionOnly(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCLogoutWithAnAccessTokenEndsItsSessionOnly@test.com"
	user := newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAuthClient(conn)

	// uid and realm are ignored once a token names the session
	res, err := client.Logout(withAccessToken(laptop.accessToken), &v1.LogoutRequest{Uid: int32(user.ID), Realm: login})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	stored := findSessionRow(t, gormDB, laptop.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), stored.RevokedReason)
	assert.Equal(t, 401, refreshOverHTTP(t, layout, laptop.refreshToken).Code)
	assert.False(t, findSessionRow(t, gormDB, phone.session.ID.String()).DeletedAt.Valid)
	assert.Equal(t, 200, statusOverHTTP(t, layout, phone.accessToken).Code)

	// logging out of a session already ended is a no-op
	res, err = client.Logout(withAccessToken(laptop.accessToken), &v1.LogoutRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), findSessionRow(t, gormDB, laptop.session.ID.String()).RevokedReason)
}

func TestRPCLogoutWithARefreshTokenEndsItsSessionOnly(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCLogoutWithARefreshTokenEndsItsSessionOnly@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	expired := loginDevice(t, layout, login, services.ClientInfo{})
	assert.NoError(t, gormDB.Model(&entities.Session{}).Where("id = ?", expired.session.ID).Update("expires_at", layout.AccessTokenFactory.TimeFn().Add(-time.Second)).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAuthClient(conn)

	res, err := client.Logout(context.Background(), &v1.LogoutRequest{RefreshToken: phone.refreshToken})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	stored := findSessionRow(t, gormDB, phone.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), stored.RevokedReason)
	assert.False(t, findSessionRow(t, gormDB, laptop.session.ID.String()).DeletedAt.Valid)

	// an expired session is left as is
	res, err = client.Logout(context.Background(), &v1.LogoutRequest{RefreshToken: expired.refreshToken})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	assert.False(t, findSessionRow(t, gormDB, expired.session.ID.String()).DeletedAt.Valid)

	// an access token is not a refresh token
	res, err = client.Logout(context.Background(), &v1.LogoutRequest{RefreshToken: laptop.accessToken})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	assert.False(t, findSessionRow(t, gormDB, laptop.session.ID.String()).DeletedAt.Valid)
}
