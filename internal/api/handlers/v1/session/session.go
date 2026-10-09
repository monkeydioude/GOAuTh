package session

import (
	stdErr "errors"
	"net/http"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
)

// SessionOut is one session of the user: a login on a device, kept alive by refreshes.
type SessionOut struct {
	ID             string     `json:"id"`
	UserAgent      string     `json:"user_agent"`
	LoginIP        string     `json:"login_ip"`
	LastIP         string     `json:"last_ip"`
	CreatedAt      time.Time  `json:"created_at"`
	LastConnection time.Time  `json:"last_connection"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	RevokedReason  string     `json:"revoked_reason,omitempty"`
	// Current is the session of the access token the request was made with
	Current bool `json:"current"`
}

type ListOut struct {
	Sessions []SessionOut `json:"sessions"`
}

// caller reads the claims of the access token in the Authorization cookie.
// The token must be valid and its session active.
func caller(h *handlers.Layout, req *http.Request) (crypt.JWTDefaultClaims, error) {
	cookie, err := req.Cookie(consts.AuthorizationCookie)
	if err != nil {
		return crypt.JWTDefaultClaims{}, errors.Unauthorized(stdErr.New("No JWT provided in the request"))
	}
	token, err := services.GetTokenFromBearer(cookie.Value)
	if err != nil {
		return crypt.JWTDefaultClaims{}, err
	}
	jwt, err := services.AuthenticateAccessToken(req.Context(), token, *h.AccessTokenFactory)
	if err != nil {
		return crypt.JWTDefaultClaims{}, err
	}
	return jwt.Claims, nil
}

// List lists the caller's active sessions, most recently used first.
// ?include_revoked=true adds the revoked and expired ones.
func List(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	claims, err := caller(h, req)
	if err != nil {
		errors.HTTPError(err, w)
		return
	}
	includeRevoked := req.URL.Query().Get("include_revoked") == "true"
	sessions, err := services.ListSessions(h.DB.WithContext(req.Context()), claims.UID, includeRevoked, h.AccessTokenFactory.TimeFn())
	if err != nil {
		errors.HTTPError(errors.DBError(err), w)
		return
	}
	out := ListOut{Sessions: make([]SessionOut, 0, len(sessions))}
	for _, session := range sessions {
		out.Sessions = append(out.Sessions, intoSessionOut(session, claims.SID))
	}
	response.Json(out, w)
}

// Revoke revokes one of the caller's sessions, named by the {id} path value.
func Revoke(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	claims, err := caller(h, req)
	if err != nil {
		errors.HTTPError(err, w)
		return
	}
	revoked, err := services.RevokeSession(h.DB.WithContext(req.Context()), claims.UID, req.PathValue("id"), entities.SessionRevokedByUser, h.AccessTokenFactory.TimeFn())
	if err != nil {
		errors.HTTPError(errors.DBError(err), w)
		return
	}
	// another user's session is not found either
	if !revoked {
		response.NotFound(consts.ERR_SESSION_NOT_FOUND, w)
		return
	}
	response.JsonOk(w)
}

// RevokeAll revokes all the caller's sessions. ?keep_current=true keeps the caller's.
func RevokeAll(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	claims, err := caller(h, req)
	if err != nil {
		errors.HTTPError(err, w)
		return
	}
	keepSID := ""
	if req.URL.Query().Get("keep_current") == "true" {
		keepSID = claims.SID
	}
	if err := services.RevokeAllSessions(h.DB.WithContext(req.Context()), claims.UID, keepSID, h.AccessTokenFactory.TimeFn()); err != nil {
		errors.HTTPError(errors.DBError(err), w)
		return
	}
	response.JsonOk(w)
}

func intoSessionOut(session entities.Session, currentSID string) SessionOut {
	out := SessionOut{
		ID:             session.ID.String(),
		UserAgent:      session.UserAgent,
		LoginIP:        session.LoginIP,
		LastIP:         session.LastIP,
		CreatedAt:      session.CreatedAt.UTC(),
		LastConnection: session.LastConnection.UTC(),
		ExpiresAt:      session.ExpiresAt.UTC(),
		Current:        session.ID.String() == currentSID,
	}
	if session.DeletedAt.Valid {
		revokedAt := session.DeletedAt.Time.UTC()
		out.RevokedAt = &revokedAt
	}
	if session.RevokedReason != nil {
		out.RevokedReason = *session.RevokedReason
	}
	return out
}
