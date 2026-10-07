package functional

import (
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const retention = 90 * 24 * time.Hour

func sessionExists(t *testing.T, gormDB *gorm.DB, device deviceLogin) bool {
	var count int64
	assert.NoError(t, gormDB.Unscoped().Model(&entities.Session{}).Where("id = ?", device.session.ID).Count(&count).Error)
	return count == 1
}

func endSession(t *testing.T, gormDB *gorm.DB, device deviceLogin, column string, at time.Time) {
	assert.NoError(t, gormDB.Unscoped().Model(&entities.Session{}).Where("id = ?", device.session.ID).Update(column, at).Error)
}

func TestPurgeDeletesSessionsEndedBeforeTheRetention(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestPurgeDeletesSessionsEndedBeforeTheRetention@test.com"
	newLoginUser(t, gormDB, login)
	now := layout.RefreshTokenFactory.TimeFn()
	cutoff := now.Add(-retention)
	active := loginDevice(t, layout, login, services.ClientInfo{})
	oldRevoked := loginDevice(t, layout, login, services.ClientInfo{})
	endSession(t, gormDB, oldRevoked, "deleted_at", cutoff.Add(-time.Hour))
	recentRevoked := loginDevice(t, layout, login, services.ClientInfo{})
	endSession(t, gormDB, recentRevoked, "deleted_at", cutoff.Add(time.Hour))
	oldExpired := loginDevice(t, layout, login, services.ClientInfo{})
	endSession(t, gormDB, oldExpired, "expires_at", cutoff.Add(-time.Hour))
	recentExpired := loginDevice(t, layout, login, services.ClientInfo{})
	endSession(t, gormDB, recentExpired, "expires_at", cutoff.Add(time.Hour))

	purged, err := services.PurgeSessions(gormDB, cutoff)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), purged)

	assert.False(t, sessionExists(t, gormDB, oldRevoked))
	assert.False(t, sessionExists(t, gormDB, oldExpired))
	assert.True(t, sessionExists(t, gormDB, recentRevoked))
	assert.True(t, sessionExists(t, gormDB, recentExpired))
	assert.True(t, sessionExists(t, gormDB, active))
	assert.Equal(t, 200, refreshOverHTTP(t, layout, active.refreshToken).Code)
}
