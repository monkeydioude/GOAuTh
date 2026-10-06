package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClientInfoNormalizesTheIP(t *testing.T) {
	assert.Equal(t, ClientInfo{IP: "203.0.113.7", UserAgent: "Mozilla/5.0"}, NewClientInfo("203.0.113.7", "Mozilla/5.0"))
	assert.Equal(t, "2001:db8::1", NewClientInfo("2001:db8::1", "").IP)
	assert.Equal(t, "10.0.0.2", NewClientInfo("::ffff:10.0.0.2", "").IP)
	assert.Equal(t, "fe80::1", NewClientInfo("fe80::1%en0", "").IP)
	// an invalid IP is dropped, the user agent is kept
	assert.Equal(t, ClientInfo{UserAgent: "curl/8.7.1"}, NewClientInfo("not-an-ip", "curl/8.7.1"))
	assert.Equal(t, ClientInfo{}, NewClientInfo("", ""))
}
