package functional

import (
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"github.com/stretchr/testify/assert"
)

func TestBootDropsTheLegacyRefreshTokenColumn(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// a database from before sessions
	assert.NoError(t, gormDB.Exec("ALTER TABLE users ADD COLUMN refresh_token text").Error)
	assert.True(t, gormDB.Migrator().HasColumn(&entities.User{}, "refresh_token"))

	rebooted, rebootedDB, _ := setup()
	defer cleanup(rebooted)
	assert.False(t, rebootedDB.Migrator().HasColumn(&entities.User{}, "refresh_token"))

	// booting again is a no-op
	again, againDB, _ := setup()
	defer cleanup(again)
	assert.False(t, againDB.Migrator().HasColumn(&entities.User{}, "refresh_token"))
	newLoginUser(t, againDB, "TestBootDropsTheLegacyRefreshTokenColumn@test.com")
}
