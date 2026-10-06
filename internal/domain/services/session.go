package services

import (
	"errors"
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
func touchSession(db *gorm.DB, attempt refreshAttempt) error {
	return db.Model(&entities.Session{}).
		Where("id = ?", attempt.sessionID).
		Updates(touchColumns(attempt.client, attempt.now)).Error
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
