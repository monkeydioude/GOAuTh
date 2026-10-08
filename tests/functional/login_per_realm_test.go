package functional

import (
	"context"
	"strings"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// newRealmUser creates a realm and a user in it with the password "test".
func newRealmUser(t *testing.T, gormDB *gorm.DB, realmName string, login string) entities.User {
	realm := entities.Realm{ID: uuid.New(), Name: realmName, AllowNewUser: true}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{Login: login, Password: "test", RealmID: realm.ID, RealmName: realm.Name}
	assert.NoError(t, gormDB.Create(&user).Error)
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&user)
		gormDB.Unscoped().Delete(&realm)
	})
	return user
}

// loginIn logs a user of realm in and returns their access token.
func loginIn(t *testing.T, layout *handlers.Layout, realm string, login string, password string) (string, error) {
	cookie, _, err := services.AuthLogin(entities.NewUser(login, password, realm), services.ClientInfo{}, layout.DB, layout.UserParams, layout.AccessTokenFactory, layout.RefreshTokenFactory, layout.MaxActiveSessions)
	return strings.TrimPrefix(cookie.Value, "Bearer "), err
}

func TestBootDropsTheGlobalLoginIndex(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// a database from before logins were unique per realm
	assert.NoError(t, gormDB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_login_active ON users (login) WHERE deleted_at IS NULL").Error)
	assert.True(t, gormDB.Migrator().HasIndex(&entities.User{}, "idx_login_active"))

	rebooted, rebootedDB, _ := setup()
	defer cleanup(rebooted)
	assert.False(t, rebootedDB.Migrator().HasIndex(&entities.User{}, "idx_login_active"))
	assert.True(t, rebootedDB.Migrator().HasIndex(&entities.User{}, "idx_realm_login_active"))

	// booting again is a no-op
	again, againDB, _ := setup()
	defer cleanup(again)
	assert.False(t, againDB.Migrator().HasIndex(&entities.User{}, "idx_login_active"))
	assert.True(t, againDB.Migrator().HasIndex(&entities.User{}, "idx_realm_login_active"))
}

func TestSignupAllowsTheSameLoginInAnotherRealm(t *testing.T) {
	layout, gormDB, _ := setup()
	login := "TestSignupAllowsTheSameLoginInAnotherRealm@test.com"
	newRealmUser(t, gormDB, "realm-a", login)
	realmB := entities.Realm{ID: uuid.New(), Name: "realm-b", AllowNewUser: true}
	assert.NoError(t, gormDB.Create(&realmB).Error)
	conn := setupRPC(t, layout)
	t.Cleanup(func() {
		conn.Close()
		cleanup(layout)
		gormDB.Unscoped().Delete(&entities.User{}, "login = ?", login)
		gormDB.Unscoped().Delete(&realmB)
	})
	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	res, err := client.Signup(ctx, &v1.UserRequest{Login: login, Password: "test", Realm: realmB.Name})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	// twice in the same realm is still refused
	res, err = client.Signup(ctx, &v1.UserRequest{Login: login, Password: "test", Realm: realmB.Name})
	assert.NoError(t, err)
	assert.Equal(t, int32(400), res.Code)
	assert.Contains(t, res.Message, consts.ERR_USER_ALREADY_EXIST)

	var count int64
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("login = ?", login).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestUserActionActsOnTheUserOfItsRealm(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestUserActionActsOnTheUserOfItsRealm@test.com"
	newRealmUser(t, gormDB, "realm-a", login)
	inB := newRealmUser(t, gormDB, "realm-b", login)

	created, err := services.UserActionCreate(gormDB, services.UserActionCreateIn{
		Login:  login,
		Realm:  "realm-b",
		Action: entities.UserActionTypePassword,
	}, uuid.NewString)
	assert.NoError(t, err)
	var action entities.UserAction
	assert.NoError(t, gormDB.First(&action, "data = ?", created.Data).Error)
	assert.Equal(t, inB.ID, action.UserID)

	_, err = services.UserActionValidate(gormDB, layout.UserParams, services.UserActionValidateIn{
		Login:   login,
		Realm:   "realm-b",
		Data:    created.Data,
		Against: "newpassword",
	})
	assert.NoError(t, err)

	// only realm-b's user got the new password
	_, err = loginIn(t, layout, "realm-b", login, "newpassword")
	assert.NoError(t, err)
	_, err = loginIn(t, layout, "realm-b", login, "test")
	assert.Error(t, err)
	_, err = loginIn(t, layout, "realm-a", login, "test")
	assert.NoError(t, err)

	// and the action is listed under realm-b only
	actions, err := services.UserActionStatus(gormDB, services.UserActionStatusIn{Realm: "realm-b", Login: login, Action: entities.UserActionTypePassword})
	assert.NoError(t, err)
	assert.Len(t, actions, 1)
	actions, err = services.UserActionStatus(gormDB, services.UserActionStatusIn{Realm: "realm-a", Login: login, Action: entities.UserActionTypePassword})
	assert.NoError(t, err)
	assert.Len(t, actions, 0)
}

func TestLoginChangeAllowsALoginUsedInAnotherRealm(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestLoginChangeAllowsALoginUsedInAnotherRealm@test.com"
	taken := "taken-TestLoginChangeAllowsALoginUsedInAnotherRealm@test.com"
	mover := newRealmUser(t, gormDB, "realm-a", login)
	newRealmUser(t, gormDB, "realm-b", taken)

	token, err := loginIn(t, layout, "realm-a", login, "test")
	assert.NoError(t, err)
	assert.Equal(t, 200, editLoginOverHTTP(t, layout, token, "test", taken).Code)
	var moved entities.User
	assert.NoError(t, gormDB.First(&moved, mover.ID).Error)
	assert.Equal(t, taken, moved.Login)

	// taken in realm-a now, so nobody else there can move to it
	otherLogin := "other-TestLoginChangeAllowsALoginUsedInAnotherRealm@test.com"
	other := entities.User{Login: otherLogin, Password: "test", RealmID: mover.RealmID, RealmName: "realm-a"}
	assert.NoError(t, gormDB.Create(&other).Error)
	t.Cleanup(func() { gormDB.Unscoped().Delete(&other) })
	token, err = loginIn(t, layout, "realm-a", otherLogin, "test")
	assert.NoError(t, err)
	assert.NotEqual(t, 200, editLoginOverHTTP(t, layout, token, "test", taken).Code)
	var unchanged entities.User
	assert.NoError(t, gormDB.First(&unchanged, other.ID).Error)
	assert.Equal(t, otherLogin, unchanged.Login)
}
