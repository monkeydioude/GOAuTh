package entities

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionBeforeCreateRequiresUserAndTokenHash(t *testing.T) {
	assert.Error(t, (&Session{TokenHash: "hash"}).BeforeCreate(nil))
	assert.Error(t, (&Session{UserID: 1}).BeforeCreate(nil))
}

func TestSessionBeforeCreateSetsAnIDOnlyWhenMissing(t *testing.T) {
	trial := Session{UserID: 1, TokenHash: "hash"}
	assert.NoError(t, trial.BeforeCreate(nil))
	assert.NotEqual(t, uuid.Nil, trial.ID)

	id := uuid.New()
	trial = Session{ID: id, UserID: 1, TokenHash: "hash"}
	assert.NoError(t, trial.BeforeCreate(nil))
	assert.Equal(t, id, trial.ID)
}
