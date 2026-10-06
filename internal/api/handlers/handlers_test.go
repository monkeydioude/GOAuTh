package handlers

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/services"

	"github.com/stretchr/testify/assert"
)

func TestLayoutClientInfo(t *testing.T) {
	layout := &Layout{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	req := httptest.NewRequest("PUT", "/identity/v1/auth/login", nil)
	req.RemoteAddr = "10.0.0.2:8080"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	assert.Equal(t, services.ClientInfo{IP: "198.51.100.1", UserAgent: "Mozilla/5.0"}, layout.ClientInfo(req))
}
