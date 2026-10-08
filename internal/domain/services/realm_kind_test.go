package services

import (
	"strings"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func codeOf(t *testing.T, err error) int {
	typed, ok := err.(errors.Err)
	if !ok {
		t.Fatalf("not an errors.Err: %v", err)
	}
	return typed.Code()
}

func TestHumanRealmsHavePasswordFlowsAndNoManagedAccounts(t *testing.T) {
	human := entities.Realm{Kind: entities.RealmKindHuman}
	assert.NoError(t, KindOf(human).AssertPasswordFlows())
	_, err := KindOf(human).NewAccount(human, "bots", "human creation")
	assert.Equal(t, 403, codeOf(t, err))

	// a realm from before kinds existed
	assert.NoError(t, KindOf(entities.Realm{}).AssertPasswordFlows())
}

func TestServiceRealmsRefusePasswordFlows(t *testing.T) {
	err := KindOf(entities.Realm{Kind: entities.RealmKindService}).AssertPasswordFlows()
	assert.Equal(t, 403, codeOf(t, err))
}

func TestServiceAccountsHaveASlugAnActorAndNoPassword(t *testing.T) {
	realm := entities.Realm{ID: uuid.New(), Name: "bots", Kind: entities.RealmKindService}
	kind := KindOf(realm)

	account, err := kind.NewAccount(realm, "sb:org:42", "human creation")
	assert.NoError(t, err)
	assert.Equal(t, "", account.Password)
	assert.Equal(t, realm.ID, account.RealmID)
	assert.Equal(t, realm.Name, account.RealmName)
	assert.Equal(t, "human creation", *account.CreatedBy)

	for _, login := range []string{"sb@org.com", "Bots", ""} {
		_, err := kind.NewAccount(realm, login, "human creation")
		assert.Equal(t, 422, codeOf(t, err), login)
	}
	for _, actor := range []string{"", strings.Repeat("a", 256)} {
		_, err := kind.NewAccount(realm, "bots", actor)
		assert.Equal(t, 422, codeOf(t, err))
	}
}
