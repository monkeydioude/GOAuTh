package entities

import (
	"testing"

	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func TestHumanRealmsHavePasswordFlows(t *testing.T) {
	assert.NoError(t, Realm{Kind: RealmKindHuman}.Strategy().AssertPasswordFlows())
	// a realm from before kinds existed
	assert.NoError(t, Realm{}.Strategy().AssertPasswordFlows())
}

func TestServiceRealmsRefusePasswordFlows(t *testing.T) {
	err := Realm{Kind: RealmKindService}.Strategy().AssertPasswordFlows()
	assert.Error(t, err)
	assert.Equal(t, 403, err.(errors.Err).Code())
}
