package handlers

import (
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/domain/models"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/http/request"
	"github.com/monkeydioude/goauth/v2/pkg/plugins"

	"gorm.io/gorm"
)

// Layout is the context (in a setting sort of way) of a handler.
// It mostly holds dependencies and settings that need to get passed on
// up the execution tree.
type Layout struct {
	DB                  *gorm.DB
	SigningMethod       crypt.JWTSigningMethod
	UserParams          *models.UsersParams
	AccessTokenFactory  *services.JWTFactory
	RefreshTokenFactory *services.JWTFactory
	Plugins             *plugins.PluginsRecord
	// TrustedProxies may set X-Forwarded-For
	TrustedProxies []netip.Prefix
	// MaxActiveSessions a user may have; a login beyond it revokes the least recently used one
	MaxActiveSessions int
	// SessionReuseGrace lets a just-rotated refresh token still get an access token
	SessionReuseGrace time.Duration
	// AccessKeyMaxActive is the live access keys an account may hold; a realm may cap lower
	AccessKeyMaxActive int
}

// ClientInfo reads the end user's IP and user agent from req.
func (l *Layout) ClientInfo(req *http.Request) services.ClientInfo {
	return services.NewClientInfo(request.ClientIP(req, l.TrustedProxies), req.UserAgent())
}

// Handler our basic generic route handler
type Handler func(*Layout, http.ResponseWriter, *http.Request)

// Methods vector of available HTTP MEthods
var Methods = [5]string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// WithMethod is a geeneric wrapper around a generic handler, forcing the a HTTP verb
func (l *Layout) WithMethod(method string, handler Handler) func(http.ResponseWriter, *http.Request) {
	// #StephenCurrying
	return func(w http.ResponseWriter, req *http.Request) {
		for _, m := range Methods {
			// a method matches
			if m == method {
				handler(l, w, req)
				return
			}
		}
		// no method matched the one provided over the array of available methods
		w.WriteHeader(405)
		w.Write([]byte(fmt.Sprintf("Method %s not allowd", req.Method)))
	}
}

// Get is a wrapper around a generic handler, forcing the GET HTTP verb
func (l *Layout) Get(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("GET", handler)
}

// Post is a wrapper around a generic handler, forcing the POST HTTP verb
func (l *Layout) Post(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("POST", handler)
}

// Put is a wrapper around a generic handler, forcing the PUT HTTP verb
func (l *Layout) Put(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("PUT", handler)
}

// Patch is a wrapper around a generic handler, forcing the PATCH HTTP verb
func (l *Layout) Patch(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("PATCH", handler)
}

// Delete is a wrapper around a generic handler, forcing the DELETE HTTP verb
func (l *Layout) Delete(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("DELETE", handler)
}
