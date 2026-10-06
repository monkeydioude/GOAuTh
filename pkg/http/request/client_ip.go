package request

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ParseTrustedProxies parses a comma-separated list of CIDRs or single IPs.
// An empty list trusts no proxy.
func ParseTrustedProxies(list string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, entry := range strings.Split(list, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if addr, err := netip.ParseAddr(entry); err == nil {
			addr = normalize(addr)
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("ParseTrustedProxies: %q is neither an IP nor a CIDR", entry)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

// ClientIP returns the IP of the client that sent r, or "" when RemoteAddr is unusable.
//
// X-Forwarded-For is only read when the direct peer is a trusted proxy. It is then
// walked from the right, and the first address that is not a trusted proxy is the
// client, so addresses a client prepends itself are never reached. A malformed hop
// stops the walk at the last trusted address.
func ClientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	client, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	client = normalize(client)
	if !isTrusted(client, trustedProxies) {
		return client.String()
	}
	hops := forwardedFor(r.Header)
	for i := len(hops) - 1; i >= 0; i-- {
		hop, err := netip.ParseAddr(hops[i])
		if err != nil {
			break
		}
		client = normalize(hop)
		if !isTrusted(client, trustedProxies) {
			break
		}
	}
	return client.String()
}

func forwardedFor(header http.Header) []string {
	var hops []string
	for _, line := range header.Values("X-Forwarded-For") {
		for _, hop := range strings.Split(line, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}
	return hops
}

func isTrusted(addr netip.Addr, trustedProxies []netip.Prefix) bool {
	for _, prefix := range trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func normalize(addr netip.Addr) netip.Addr {
	return addr.Unmap().WithZone("")
}
