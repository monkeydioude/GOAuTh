package services

import "net/netip"

// ClientInfo describes the end user's client behind a login or a refresh.
type ClientInfo struct {
	IP        string
	UserAgent string
}

// NewClientInfo normalizes ip (IPv4-mapped IPv6 unmapped, zone dropped)
// and leaves it empty when it is not a valid address.
func NewClientInfo(ip, userAgent string) ClientInfo {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ClientInfo{UserAgent: userAgent}
	}
	return ClientInfo{
		IP:        addr.Unmap().WithZone("").String(),
		UserAgent: userAgent,
	}
}
