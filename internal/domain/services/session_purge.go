package services

import (
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"gorm.io/gorm"
)

// PurgeSessions deletes for good the sessions revoked or expired before cutoff.
// It returns how many it deleted.
func PurgeSessions(db *gorm.DB, cutoff time.Time) (int64, error) {
	res := db.Unscoped().
		Where("deleted_at < ? OR expires_at < ?", cutoff, cutoff).
		Delete(&entities.Session{})
	return res.RowsAffected, res.Error
}
