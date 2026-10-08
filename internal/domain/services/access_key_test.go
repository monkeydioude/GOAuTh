package services

import (
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"github.com/stretchr/testify/assert"
)

func TestTheRealmCapNeverExceedsTheServers(t *testing.T) {
	two, five := 2, 5
	assert.Equal(t, 3, effectiveAccessKeyCap(entities.Realm{}, 3))
	assert.Equal(t, 2, effectiveAccessKeyCap(entities.Realm{AccessKeyMaxActive: &two}, 3))
	assert.Equal(t, 3, effectiveAccessKeyCap(entities.Realm{AccessKeyMaxActive: &five}, 3))
}
