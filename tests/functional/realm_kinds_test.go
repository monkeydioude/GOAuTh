package functional

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/auth"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/user"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// newServiceRealmUser seeds a service realm named after login and an account in
// it. The account gets the password "test", hashed, which no service account
// should have: the refusals must not depend on what password is stored.
func newServiceRealmUser(t *testing.T, gormDB *gorm.DB, login string) entities.User {
	realm := entities.Realm{ID: uuid.New(), Name: login, AllowNewUser: true, Kind: entities.RealmKindService}
	assert.NoError(t, gormDB.Create(&realm).Error)
	account := entities.User{Login: login, Password: "test", RealmID: realm.ID, RealmName: realm.Name}
	assert.NoError(t, gormDB.Create(&account).Error)
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&account)
		gormDB.Unscoped().Delete(&realm)
	})
	return account
}

// forgedAccessToken mints the access token of a new active session of account,
// for accounts that cannot log in.
func forgedAccessToken(t *testing.T, layout *handlers.Layout, account entities.User) string {
	sid := uuid.New()
	now := layout.AccessTokenFactory.TimeFn()
	session := entities.Session{ID: sid, UserID: account.ID, TokenHash: "forged-" + sid.String(), LastConnection: now, ExpiresAt: now.Add(time.Hour)}
	assert.NoError(t, layout.DB.Create(&session).Error)
	claims := account.IntoClaims()
	claims.SID = sid.String()
	jwt, err := layout.AccessTokenFactory.GenerateToken(claims)
	assert.NoError(t, err)
	return jwt.Token
}

func httpLogin(t *testing.T, layout *handlers.Layout, login string, password string, realm string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", layout.Put(auth.Login))
	body, err := json.Marshal(auth.LoginIn{Login: login, Password: password, RealmName: realm})
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("PUT", "/v1/auth/login", bytes.NewReader(body)))
	return rec
}

func httpSignup(t *testing.T, layout *handlers.Layout, login string, password string, realm string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/signup", layout.Post(auth.Signup))
	body, err := json.Marshal(map[string]string{"login": login, "password": password, "realm_name": realm})
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewReader(body)))
	return rec
}

func httpEditPassword(t *testing.T, layout *handlers.Layout, accessToken string, password string, newPassword string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/user/password", layout.Put(user.EditPassword))
	body, err := json.Marshal(entities.EditUserPayload{Password: password, NewPassword: &newPassword})
	assert.NoError(t, err)
	req := httptest.NewRequest("PUT", "/v1/user/password", bytes.NewReader(body))
	req.AddCookie(accessCookie(accessToken))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestRealmsAreHumanUnlessSaidOtherwise(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// setup inserts its realm without a kind, as every realm from before kinds existed
	var kind string
	assert.NoError(t, gormDB.Raw("SELECT kind FROM realms WHERE name = 'test'").Scan(&kind).Error)
	assert.Equal(t, entities.RealmKindHuman, kind)

	realm := entities.Realm{ID: uuid.New(), Name: "TestRealmsAreHumanUnlessSaidOtherwise"}
	assert.NoError(t, gormDB.Create(&realm).Error)
	t.Cleanup(func() { gormDB.Unscoped().Delete(&realm) })
	var stored entities.Realm
	assert.NoError(t, gormDB.First(&stored, "id = ?", realm.ID).Error)
	assert.Equal(t, entities.RealmKindHuman, stored.Kind)

	// only human and service exist
	assert.Error(t, gormDB.Exec("INSERT INTO realms (id, name, allow_new_user, kind) VALUES (gen_random_uuid(), 'robots', true, 'robot')").Error)
}

func TestAnEmptyPasswordMatchesNobody(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestAnEmptyPasswordMatchesNobody@test.com"
	account := newServiceRealmUser(t, gormDB, login)
	// a service account stores no password
	assert.NoError(t, gormDB.Exec("UPDATE users SET password = '' WHERE id = ?", account.ID).Error)
	human := newLoginUser(t, gormDB, "human-"+login)

	// the guard is in AssertAuth itself: without it, password = '' would match
	assert.Error(t, (&entities.User{Login: login, Password: "", RealmName: login}).AssertAuth(gormDB, layout.UserParams))
	assert.Equal(t, 401, httpLogin(t, layout, login, "", login).Code)
	assert.Equal(t, 401, httpLogin(t, layout, human.Login, "", human.RealmName).Code)
	conn := setupRPC(t, layout)
	defer conn.Close()
	res, err := v1.NewAuthClient(conn).Login(context.Background(), &v1.UserRequest{Login: login, Password: "", Realm: login})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
}

func TestServiceRealmsRefuseSignup(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	realm := entities.Realm{ID: uuid.New(), Name: "TestServiceRealmsRefuseSignup", AllowNewUser: true, Kind: entities.RealmKindService}
	assert.NoError(t, gormDB.Create(&realm).Error)
	t.Cleanup(func() { gormDB.Unscoped().Delete(&realm) })
	login := "TestServiceRealmsRefuseSignup@test.com"

	rec := httpSignup(t, layout, login, "test", realm.Name)
	assert.Equal(t, 403, rec.Code)
	assert.Contains(t, rec.Body.String(), consts.ERR_FORBIDDEN_BY_REALM_KIND)
	conn := setupRPC(t, layout)
	defer conn.Close()
	res, err := v1.NewAuthClient(conn).Signup(context.Background(), &v1.UserRequest{Login: login, Password: "test", Realm: realm.Name})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), res.Code)
	assert.Contains(t, res.Message, consts.ERR_FORBIDDEN_BY_REALM_KIND)

	var count int64
	assert.NoError(t, gormDB.Model(&entities.User{}).Where("login = ?", login).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestServiceRealmsRefuseLogin(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestServiceRealmsRefuseLogin@test.com"
	newServiceRealmUser(t, gormDB, login)

	// the same answer as a wrong password
	rec := httpLogin(t, layout, login, "test", login)
	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), consts.ERR_INVALID_CREDENTIALS)
	conn := setupRPC(t, layout)
	defer conn.Close()
	res, err := v1.NewAuthClient(conn).Login(context.Background(), &v1.UserRequest{Login: login, Password: "test", Realm: login})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	assert.Contains(t, res.Message, consts.ERR_INVALID_CREDENTIALS)
}

func TestServiceRealmsRefusePasswordAndLoginChanges(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestServiceRealmsRefusePasswordAndLoginChanges@test.com"
	account := newServiceRealmUser(t, gormDB, login)
	token := forgedAccessToken(t, layout, account)
	newLogin := "new-" + login

	assert.Equal(t, 403, httpEditPassword(t, layout, token, "test", "newpassword").Code)
	assert.Equal(t, 403, editLoginOverHTTP(t, layout, token, "test", newLogin).Code)

	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewUserClient(conn)
	res, err := client.EditUser(withAccessToken(token), &v1.EditUserRequest{Password: "test", NewPassword: "newpassword"})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), res.Code)
	res, err = client.EditUser(withAccessToken(token), &v1.EditUserRequest{Password: "test", NewLogin: newLogin})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), res.Code)

	stored := storedUser(t, gormDB, account.ID)
	assert.Equal(t, login, stored.Login)
	assert.Equal(t, account.Password, stored.Password)
}

func TestServiceRealmsRefuseUserActions(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestServiceRealmsRefuseUserActions@test.com"
	newServiceRealmUser(t, gormDB, login)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewUserActionClient(conn)
	ctx := context.Background()

	res, err := client.Create(ctx, &v1.UserActionRequest{Login: login, Realm: login, Action: entities.UserActionTypePassword})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), res.Code)
	res, err = client.Validate(ctx, &v1.UserActionValidation{Login: login, Realm: login, Data: "code", Against: "newpassword"})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), res.Code)
	status, err := client.Status(ctx, &v1.UserActionRequest{Login: login, Realm: login, Action: entities.UserActionTypePassword})
	assert.NoError(t, err)
	assert.Equal(t, int32(403), status.Code)

	var count int64
	assert.NoError(t, gormDB.Model(&entities.UserAction{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
