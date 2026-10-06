package services

import (
	"errors"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"

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
			if err := RevokeSessions(tx, entities.SessionRevokedLimitExceeded, now, "id IN ?", ids); err != nil {
				return err
			}
		}
	}
	if err := tx.Create(session).Error; err != nil {
		return err
	}
	return tx.Model(&entities.User{}).Where("id = ?", session.UserID).Update("last_logged_at", now).Error
}

// RevokeSessions soft-deletes the active sessions matching query, recording why.
func RevokeSessions(tx *gorm.DB, reason string, now time.Time, query string, args ...any) error {
	// the soft-delete scope only matches sessions not revoked yet
	return tx.Model(&entities.Session{}).
		Where(query, args...).
		Updates(map[string]any{"deleted_at": now, "revoked_reason": reason}).Error
}

// isUserActive tells whether the user still exists and is not deactivated.
func isUserActive(db *gorm.DB, uid uint) (bool, error) {
	err := db.Select("id").First(&entities.User{}, uid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ListSessions lists the user's active sessions, most recently used first.
// includeRevoked adds the revoked and expired ones.
func ListSessions(db *gorm.DB, uid uint, includeRevoked bool, now time.Time) ([]entities.Session, error) {
	query := db.Where("user_id = ?", uid)
	if includeRevoked {
		query = query.Unscoped()
	} else {
		query = query.Where("expires_at > ?", now)
	}
	var sessions []entities.Session
	err := query.Order("last_connection DESC, created_at DESC").Find(&sessions).Error
	return sessions, err
}

// RevokeSession revokes one active session of the user.
// It is false when the user has no such session.
func RevokeSession(db *gorm.DB, uid uint, sid string, reason string, now time.Time) (bool, error) {
	id, err := uuid.Parse(sid)
	if err != nil {
		return false, nil
	}
	// the soft-delete scope only matches a session not revoked yet
	res := db.Model(&entities.Session{}).
		Where("id = ? AND user_id = ? AND expires_at > ?", id, uid, now).
		Updates(map[string]any{"deleted_at": now, "revoked_reason": reason})
	return res.RowsAffected == 1, res.Error
}

// RevokeAllSessions revokes the user's active sessions, except keepSID when not empty.
func RevokeAllSessions(db *gorm.DB, uid uint, keepSID string, now time.Time) error {
	if keepSID == "" {
		return RevokeSessions(db, entities.SessionRevokedLogoutAll, now, "user_id = ? AND expires_at > ?", uid, now)
	}
	return RevokeSessions(db, entities.SessionRevokedLogoutAll, now, "user_id = ? AND expires_at > ? AND id <> ?", uid, now, keepSID)
}

// LogoutSession revokes the session a token's claims name. A session already
// revoked or expired is left as is.
func LogoutSession(db *gorm.DB, claims crypt.JWTDefaultClaims, now time.Time) error {
	_, err := RevokeSession(db, claims.UID, claims.SID, entities.SessionRevokedLogout, now)
	return err
}

// IsSessionRevoked tells whether the session a token's sid names can no longer be used:
// gone, revoked, expired, or its user deactivated. It is a single query.
func IsSessionRevoked(db *gorm.DB, claims crypt.JWTDefaultClaims, now time.Time) (bool, error) {
	sid, err := uuid.Parse(claims.SID)
	if err != nil {
		return true, nil
	}
	var active int64
	// the soft-delete scope also requires the session not to be revoked
	err = db.Model(&entities.Session{}).
		Joins("JOIN users ON users.id = sessions.user_id AND users.deleted_at IS NULL").
		Where("sessions.id = ? AND sessions.user_id = ? AND sessions.expires_at > ?", sid, claims.UID, now).
		Count(&active).Error
	return active == 0, err
}

// refreshAttempt is a refresh token presented for the session its sid names.
type refreshAttempt struct {
	sessionID uuid.UUID
	userID    uint
	tokenHash string
	client    ClientInfo
	now       time.Time
}

// rotateSession replaces the session's refresh token hash with the new one. It is a
// single conditional update that only matches while the presented token is the
// current one, so two concurrent refreshes of the same token cannot both win.
func rotateSession(db *gorm.DB, attempt refreshAttempt, newTokenHash string, expiresAt time.Time) (bool, error) {
	columns := touchColumns(attempt.client, attempt.now)
	columns["token_hash"] = newTokenHash
	columns["previous_token_hash"] = attempt.tokenHash
	columns["rotated_at"] = attempt.now
	columns["expires_at"] = expiresAt
	// the soft-delete scope also requires the session not to be revoked
	res := db.Model(&entities.Session{}).
		Where("id = ? AND user_id = ? AND token_hash = ? AND expires_at > ?", attempt.sessionID, attempt.userID, attempt.tokenHash, attempt.now).
		Updates(columns)
	return res.RowsAffected == 1, res.Error
}

// findSession reads the attempt's session, revoked or not. It is nil when the session is gone.
func findSession(db *gorm.DB, attempt refreshAttempt) (*entities.Session, error) {
	var session entities.Session
	err := db.Unscoped().Where("id = ? AND user_id = ?", attempt.sessionID, attempt.userID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// touchSession records a use of the session without rotating its token.
// It is false when the session is not active anymore.
func touchSession(db *gorm.DB, attempt refreshAttempt) (bool, error) {
	// the soft-delete scope only matches an active session
	res := db.Model(&entities.Session{}).
		Where("id = ?", attempt.sessionID).
		Updates(touchColumns(attempt.client, attempt.now))
	return res.RowsAffected == 1, res.Error
}

// touchColumns records when a session was last used and, when known, from where.
func touchColumns(client ClientInfo, now time.Time) map[string]any {
	columns := map[string]any{"last_connection": now}
	if client.IP != "" {
		columns["last_ip"] = client.IP
	}
	if client.UserAgent != "" {
		columns["user_agent"] = client.UserAgent
	}
	return columns
}

type refreshOutcome int

const (
	refreshRevoked refreshOutcome = iota
	refreshExpired
	refreshInGrace
	refreshReused
)

// refreshOutcomeOf tells why a refresh token is not its session's current one;
// session is nil when the session is gone. For reuseGrace after a rotation, the
// previous token still gets an access token, so refreshes racing each other don't
// log the user out. Any other old token has leaked: its session must be revoked.
func refreshOutcomeOf(session *entities.Session, tokenHash string, now time.Time, reuseGrace time.Duration) refreshOutcome {
	switch {
	case session == nil || session.DeletedAt.Valid:
		return refreshRevoked
	case !session.ExpiresAt.After(now):
		return refreshExpired
	case isPreviousToken(session, tokenHash) && now.Sub(*session.RotatedAt) <= reuseGrace:
		return refreshInGrace
	default:
		return refreshReused
	}
}

func isPreviousToken(session *entities.Session, tokenHash string) bool {
	return session.RotatedAt != nil && session.PreviousTokenHash != nil && *session.PreviousTokenHash == tokenHash
}
