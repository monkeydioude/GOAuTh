package functional

import (
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"

	"github.com/stretchr/testify/assert"
)

// A consumer that never stores the rotated refresh token keeps presenting its login token.
// With a grace window as long as the session, it keeps getting access tokens until the
// session expires 30 days after its first refresh, and never gets logged out before that.
func TestJsonAPIClientKeepingItsLoginTokenLivesThroughAGraceAsLongAsTheSession(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestJsonAPIClientKeepingItsLoginToken@test.com"
	newLoginUser(t, gormDB, login)
	// the production values: a 30-day session, and a grace window as long
	ttl := 30 * 24 * time.Hour
	layout.RefreshTokenFactory.ExpiresIn = ttl
	layout.SessionReuseGrace = ttl
	loginToken, session := loginSession(t, layout, login)
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}

	// first refresh, 15 minutes after login: the login token is rotated out
	clock = timeRef.Add(15 * time.Minute)
	first := refreshOverHTTP(t, layout, loginToken)
	assert.Equal(t, 200, first.Code)
	rotated := refreshedToken(first)
	assert.NotEmpty(t, rotated)
	firstRefresh := clock
	expiresAt := findSessionRow(t, gormDB, session.ID.String()).ExpiresAt
	assert.True(t, expiresAt.Equal(firstRefresh.Add(ttl)))

	// the client discards the rotated token and keeps presenting the login token
	for _, later := range []time.Duration{30 * time.Minute, 24 * time.Hour, 15 * 24 * time.Hour, ttl - time.Minute} {
		clock = firstRefresh.Add(later)
		rec := refreshOverHTTP(t, layout, loginToken)
		assert.Equal(t, 200, rec.Code, "refresh %s after the first one", later)
		findCookie(t, rec.Result().Cookies(), consts.AuthorizationCookie)
		assert.Equal(t, "", refreshedToken(rec), "no new refresh token %s after the first refresh", later)
	}

	// the session was used, but not extended: it still ends 30 days after the first refresh
	stored := findSessionRow(t, gormDB, session.ID.String())
	assert.False(t, stored.DeletedAt.Valid)
	assert.Equal(t, crypt.HashToken(rotated), stored.TokenHash)
	assert.True(t, stored.LastConnection.Equal(clock))
	assert.True(t, stored.ExpiresAt.Equal(expiresAt))

	// once the session has expired, the client logs in again
	clock = expiresAt.Add(time.Second)
	expired := refreshOverHTTP(t, layout, loginToken)
	assert.Equal(t, 401, expired.Code)
	assert.Equal(t, consts.ERR_TOKEN_EXPIRED, errorMessage(t, expired))
}
