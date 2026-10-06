package functional

import (
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionsTableIsMigrated(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	m := gormDB.Migrator()

	assert.True(t, m.HasTable(&entities.Session{}))
	for _, field := range []string{"TokenHash", "UserID", "PreviousTokenHash", "DeletedAt"} {
		assert.True(t, m.HasIndex(&entities.Session{}, field), "missing index on %s", field)
	}
	assert.True(t, m.HasConstraint(&entities.Session{}, "User"))
}

func TestSessionsTableRules(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestSessionsTableRules@test.com"
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         login,
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{
		Login:     login,
		Password:  "test",
		RealmID:   realm.ID,
		RealmName: realm.Name,
	}
	assert.NoError(t, gormDB.Create(&user).Error)
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})

	session := entities.Session{UserID: user.ID, TokenHash: "hash-1", LastConnection: timeRef, ExpiresAt: timeRef.Add(time.Hour)}
	assert.NoError(t, gormDB.Create(&session).Error)
	assert.NotEqual(t, uuid.Nil, session.ID)

	// a token hash belongs to one session only
	duplicate := entities.Session{UserID: user.ID, TokenHash: "hash-1", LastConnection: timeRef, ExpiresAt: timeRef.Add(time.Hour)}
	assert.Error(t, gormDB.Create(&duplicate).Error)

	// deleting the user deletes its sessions
	assert.NoError(t, gormDB.Unscoped().Delete(&user).Error)
	var count int64
	assert.NoError(t, gormDB.Unscoped().Model(&entities.Session{}).Where("user_id = ?", user.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
