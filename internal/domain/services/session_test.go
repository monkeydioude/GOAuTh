package services

import (
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestRefreshOutcomeOf(t *testing.T) {
	now := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	grace := 30 * time.Second
	rotatedSession := func(rotatedAgo time.Duration) *entities.Session {
		return &entities.Session{
			TokenHash:         "current",
			PreviousTokenHash: ptr.Ptr("previous"),
			RotatedAt:         ptr.Ptr(now.Add(-rotatedAgo)),
			ExpiresAt:         now.Add(time.Hour),
		}
	}

	t.Run("a session that is gone is revoked", func(t *testing.T) {
		assert.Equal(t, refreshRevoked, refreshOutcomeOf(nil, "current", now, grace))
	})

	t.Run("a soft-deleted session is revoked", func(t *testing.T) {
		session := rotatedSession(time.Minute)
		session.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
		assert.Equal(t, refreshRevoked, refreshOutcomeOf(session, "current", now, grace))
	})

	t.Run("an expired session is expired, whatever the token", func(t *testing.T) {
		session := rotatedSession(time.Minute)
		session.ExpiresAt = now
		assert.Equal(t, refreshExpired, refreshOutcomeOf(session, "previous", now, grace))
	})

	t.Run("the previous token within the grace window", func(t *testing.T) {
		assert.Equal(t, refreshInGrace, refreshOutcomeOf(rotatedSession(10*time.Second), "previous", now, grace))
	})

	t.Run("the previous token at the end of the grace window", func(t *testing.T) {
		assert.Equal(t, refreshInGrace, refreshOutcomeOf(rotatedSession(grace), "previous", now, grace))
	})

	t.Run("the previous token after the grace window is reused", func(t *testing.T) {
		assert.Equal(t, refreshReused, refreshOutcomeOf(rotatedSession(31*time.Second), "previous", now, grace))
	})

	t.Run("an older token is reused", func(t *testing.T) {
		assert.Equal(t, refreshReused, refreshOutcomeOf(rotatedSession(time.Second), "older", now, grace))
	})

	t.Run("a session never rotated has no previous token", func(t *testing.T) {
		session := &entities.Session{TokenHash: "current", ExpiresAt: now.Add(time.Hour)}
		assert.Equal(t, refreshReused, refreshOutcomeOf(session, "previous", now, grace))
	})
}

func TestTouchColumns(t *testing.T) {
	now := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)

	t.Run("records the client when it is known", func(t *testing.T) {
		assert.Equal(t, map[string]any{
			"last_connection": now,
			"last_ip":         "203.0.113.7",
			"user_agent":      "Mozilla/5.0",
		}, touchColumns(ClientInfo{IP: "203.0.113.7", UserAgent: "Mozilla/5.0"}, now))
	})

	t.Run("keeps the last known client otherwise", func(t *testing.T) {
		assert.Equal(t, map[string]any{"last_connection": now}, touchColumns(ClientInfo{}, now))
	})
}
