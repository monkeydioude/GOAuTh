package request

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTrustedProxies(t *testing.T) {
	trial, err := ParseTrustedProxies(" 10.0.0.0/8, 192.168.1.10 ,2001:db8::/32,, 10.1.2.3/16")
	assert.NoError(t, err)
	assert.Equal(t, []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.168.1.10/32"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("10.1.0.0/16"),
	}, trial)

	trial, err = ParseTrustedProxies("")
	assert.NoError(t, err)
	assert.Empty(t, trial)

	for _, invalid := range []string{"not-an-ip", "10.0.0.0/33", "10.0.0.0/8, nope"} {
		_, err = ParseTrustedProxies(invalid)
		assert.Error(t, err, invalid)
	}
}

func TestClientIP(t *testing.T) {
	proxies := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	tests := []struct {
		name         string
		remoteAddr   string
		forwardedFor []string
		trusted      []netip.Prefix
		want         string
	}{
		{"direct client", "203.0.113.7:51234", nil, proxies, "203.0.113.7"},
		{"forwarded-for ignored from an untrusted peer", "203.0.113.7:51234", []string{"198.51.100.1"}, proxies, "203.0.113.7"},
		{"forwarded-for ignored when no proxy is trusted", "10.0.0.2:8080", []string{"198.51.100.1"}, nil, "10.0.0.2"},
		{"behind a trusted proxy", "10.0.0.2:8080", []string{"198.51.100.1"}, proxies, "198.51.100.1"},
		{"a hop prepended by the client is never reached", "10.0.0.2:8080", []string{"1.2.3.4, 198.51.100.1, 10.0.0.3"}, proxies, "198.51.100.1"},
		{"several header lines", "10.0.0.2:8080", []string{"1.2.3.4", "198.51.100.1"}, proxies, "198.51.100.1"},
		{"trusted proxy without forwarded-for", "10.0.0.2:8080", nil, proxies, "10.0.0.2"},
		{"every hop trusted", "10.0.0.2:8080", []string{"10.0.0.5, 10.0.0.3"}, proxies, "10.0.0.5"},
		{"malformed hop", "10.0.0.2:8080", []string{"198.51.100.1, garbage"}, proxies, "10.0.0.2"},
		{"ipv6 client", "[2001:db8::1]:443", nil, proxies, "2001:db8::1"},
		{"ipv4-mapped proxy", "[::ffff:10.0.0.2]:8080", []string{"198.51.100.1"}, proxies, "198.51.100.1"},
		{"remote addr without port", "203.0.113.7", nil, proxies, "203.0.113.7"},
		{"unusable remote addr", "", nil, proxies, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/identity/v1/auth/login", nil)
			req.RemoteAddr = tt.remoteAddr
			for _, line := range tt.forwardedFor {
				req.Header.Add("X-Forwarded-For", line)
			}
			assert.Equal(t, tt.want, ClientIP(req, tt.trusted))
		})
	}
}
