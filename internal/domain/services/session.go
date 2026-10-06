package services

import (
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// createSession stores a new session and records the login on its user.
// When the user already has maxActive active sessions, the least recently
// used ones are revoked to make room. maxActive < 1 means no limit.
func createSession(tx *gorm.DB, session *entities.Session, maxActive int, now time.Time) error {
	// lock the user, so concurrent logins of one user count its sessions one at a time
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&entities.User{}, session.UserID).Error; err != nil {
		return err
	}
	if maxActive > 0 {
		var active []entities.Session
		err := tx.Select("id").
			Where("user_id = ? AND expires_at > ?", session.UserID, now).
			Order("last_connection DESC, created_at DESC").
			Find(&active).Error
		if err != nil {
			return err
		}
		if excess := len(active) - maxActive + 1; excess > 0 {
			ids := make([]uuid.UUID, 0, excess)
			for _, leastRecent := range active[len(active)-excess:] {
				ids = append(ids, leastRecent.ID)
			}
			err := tx.Model(&entities.Session{}).
				Where("id IN ?", ids).
				Updates(map[string]any{"deleted_at": now, "revoked_reason": entities.SessionRevokedLimitExceeded}).Error
			if err != nil {
				return err
			}
		}
	}
	if err := tx.Create(session).Error; err != nil {
		return err
	}
	return tx.Model(&entities.User{}).Where("id = ?", session.UserID).Update("last_logged_at", now).Error
}
