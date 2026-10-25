package functional

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const verifyEmail = "verify-email"

// createCode makes login's verify-email code over gRPC, with these limits (nil: unset).
func createCode(t *testing.T, client v1.UserActionClient, login string, code string, ttlSeconds *int64, maxAttempts *int32) {
	t.Helper()
	res, err := client.Create(context.Background(), &v1.UserActionRequest{
		Login: login, Realm: login, Action: verifyEmail, Data: code,
		TtlSeconds: ttlSeconds, MaxAttempts: maxAttempts,
	})
	require.NoError(t, err)
	require.Equal(t, int32(201), res.Code, res.Message)
}

// tryCode validates code against login's pending verify-email action over gRPC.
func tryCode(t *testing.T, client v1.UserActionClient, login string, code string) *v1.Response {
	t.Helper()
	res, err := client.Validate(context.Background(), &v1.UserActionValidation{Login: login, Realm: login, Action: verifyEmail, Data: code})
	require.NoError(t, err)
	return res
}

// latestStatus is login's last verify-email action, as Status tells it.
func latestStatus(t *testing.T, client v1.UserActionClient, login string) *v1.UserActionStatus {
	t.Helper()
	res, err := client.Status(context.Background(), &v1.UserActionRequest{Login: login, Realm: login, Action: verifyEmail})
	require.NoError(t, err)
	require.NotEmpty(t, res.Statuses)
	return res.Statuses[0]
}

func TestUserActionExpires(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	client := v1.NewUserActionClient(setupRPC(t, layout))
	login := "TestUserActionExpires@test.com"
	newLoginUser(t, gormDB, login)
	createCode(t, client, login, "123456", ptr.Ptr(int64(1800)), nil)

	layout.AccessTokenFactory.TimeFn = clockAt(timeRef.Add(31 * time.Minute))
	res := tryCode(t, client, login, "123456")
	assert.Equal(t, int32(410), res.Code)
	assert.Equal(t, consts.ERR_USER_ACTION_EXPIRED, res.Message)

	layout.AccessTokenFactory.TimeFn = clockAt(timeRef.Add(29 * time.Minute))
	assert.Equal(t, int32(200), tryCode(t, client, login, "123456").Code, "still in time")
}

func TestSelfContainedCodesExpireButCountNoTries(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestSelfContainedCodesExpire@test.com"
	newLoginUser(t, gormDB, login)
	reset := func() string {
		out, err := services.UserActionCreate(gormDB, services.UserActionCreateIn{
			Login: login, Realm: login, Action: entities.UserActionTypePassword, TTL: time.Hour, MaxAttempts: 1,
		}, uuid.NewString, clockAt(timeRef))
		require.NoError(t, err)
		return out.Data
	}
	validate := func(token string, now time.Time) error {
		_, err := services.UserActionValidate(gormDB, layout.UserParams, services.UserActionValidateIn{
			Realm: login, Data: token, Against: "newpassword", SelfContainedCode: true,
		}, clockAt(now))
		return err
	}

	assertGone(t, validate(reset(), timeRef.Add(2*time.Hour)), consts.ERR_USER_ACTION_EXPIRED)
	token := reset()
	assert.NoError(t, gormDB.Model(&entities.UserAction{}).Where("data = ?", token).Update("attempts", 5).Error)
	assert.NoError(t, validate(token, timeRef.Add(time.Minute)), "no try limit on a self-contained code")
}

func TestUserActionTryLimit(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	client := v1.NewUserActionClient(conn)

	t.Run("5 wrong codes burn the action", func(t *testing.T) {
		login := "TestUserActionTryLimit-burn@test.com"
		newLoginUser(t, gormDB, login)
		createCode(t, client, login, "123456", ptr.Ptr(int64(1800)), ptr.Ptr(int32(5)))
		for range 5 {
			assert.Equal(t, int32(404), tryCode(t, client, login, "000000").Code)
		}
		res := tryCode(t, client, login, "123456")
		assert.Equal(t, int32(410), res.Code)
		assert.Equal(t, consts.ERR_USER_ACTION_TOO_MANY_ATTEMPTS, res.Message)
	})

	t.Run("4 wrong codes, then the right one", func(t *testing.T) {
		login := "TestUserActionTryLimit-four@test.com"
		newLoginUser(t, gormDB, login)
		createCode(t, client, login, "123456", nil, ptr.Ptr(int32(5)))
		for range 4 {
			assert.Equal(t, int32(404), tryCode(t, client, login, "000000").Code)
		}
		assert.Equal(t, int32(200), tryCode(t, client, login, "123456").Code)
	})

	t.Run("10 wrong codes at once use 5 tries", func(t *testing.T) {
		login := "TestUserActionTryLimit-parallel@test.com"
		newLoginUser(t, gormDB, login)
		createCode(t, client, login, "123456", nil, ptr.Ptr(int32(5)))
		codes := make(chan int32, 10)
		var wg sync.WaitGroup
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codes <- tryCode(t, client, login, "000000").Code
			}()
		}
		wg.Wait()
		close(codes)
		answered := map[int32]int{}
		for code := range codes {
			answered[code]++
		}
		assert.Equal(t, map[int32]int{404: 5, 410: 5}, answered)
		assert.Equal(t, int32(5), latestStatus(t, client, login).Attempts)
	})

	t.Run("a new code starts with 0 tries, the old one stops working", func(t *testing.T) {
		login := "TestUserActionTryLimit-new@test.com"
		newLoginUser(t, gormDB, login)
		createCode(t, client, login, "111111", nil, ptr.Ptr(int32(5)))
		for range 3 {
			tryCode(t, client, login, "000000")
		}
		createCode(t, client, login, "222222", nil, ptr.Ptr(int32(5)))
		assert.Equal(t, int32(0), latestStatus(t, client, login).Attempts)
		assert.Equal(t, int32(404), tryCode(t, client, login, "111111").Code)
		assert.Equal(t, int32(200), tryCode(t, client, login, "222222").Code)
	})
}

func TestUserActionWithoutLimitsWorksAsBefore(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	client := v1.NewUserActionClient(conn)
	login := "TestUserActionWithoutLimits@test.com"
	newLoginUser(t, gormDB, login)
	createCode(t, client, login, "123456", nil, nil)

	status := latestStatus(t, client, login)
	assert.Nil(t, status.ExpiresAt)
	assert.Equal(t, int32(0), status.MaxAttempts)
	for range 20 {
		assert.Equal(t, int32(404), tryCode(t, client, login, "000000").Code)
	}
	// as sb_back validates: by its code, no action
	res, err := client.Validate(context.Background(), &v1.UserActionValidation{Login: login, Realm: login, Data: "123456"})
	require.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
}

func TestUserActionStatusTellsTheLimits(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	client := v1.NewUserActionClient(conn)
	login := "TestUserActionStatusTellsTheLimits@test.com"
	newLoginUser(t, gormDB, login)
	createCode(t, client, login, "123456", ptr.Ptr(int64(1800)), ptr.Ptr(int32(5)))
	tryCode(t, client, login, "000000")
	tryCode(t, client, login, "000001")

	status := latestStatus(t, client, login)

	assert.Equal(t, int32(2), status.Attempts)
	assert.Equal(t, int32(5), status.MaxAttempts)
	require.NotNil(t, status.ExpiresAt)
	assert.Equal(t, timeRef.Add(30*time.Minute), status.ExpiresAt.AsTime(), "30 minutes after its creation, by the layout's clock")
}

func TestUserActionRefusesNegativeLimits(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	client := v1.NewUserActionClient(conn)
	login := "TestUserActionRefusesNegativeLimits@test.com"
	newLoginUser(t, gormDB, login)

	res, err := client.Create(context.Background(), &v1.UserActionRequest{
		Login: login, Realm: login, Action: verifyEmail, Data: "123456", TtlSeconds: ptr.Ptr(int64(-1)),
	})

	require.NoError(t, err)
	assert.Equal(t, int32(422), res.Code)
	var count int64
	assert.NoError(t, gormDB.Model(&entities.UserAction{}).Where("data = ?", "123456").Count(&count).Error)
	assert.Zero(t, count)
}

// clockAt is a clock stopped at at.
func clockAt(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func assertGone(t *testing.T, err error, message string) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, 410, err.(interface{ Code() int }).Code())
	assert.Equal(t, message, err.Error())
}
