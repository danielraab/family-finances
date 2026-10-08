// Package clientip resolves the IP address of the client behind a request,
// believing X-Forwarded-For only from configured trusted reverse proxies.
// Both rate limiting and the IP recorded on a new session use it.
package clientip

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Resolver resolves client IPs. The zero value trusts no proxy: the direct
// peer is always the client.
type Resolver struct {
	Trusted []netip.Prefix
}

// Resolve returns the request's client IP. When the direct peer is not a
// trusted proxy it is the client, whatever the headers say. Otherwise
// X-Forwarded-For is walked right to left — each hop was appended by the
// proxy that received it — and the first address outside the trusted ranges
// is the client; entries further left were written by the client itself and
// are never believed. If every hop is trusted the left-most one is taken; an
// absent or unparsable header leaves the peer. The zero Addr means the peer
// address itself was unparsable.
func (r Resolver) Resolve(req *http.Request) netip.Addr {
	peer := parseHost(req.RemoteAddr)
	if !peer.IsValid() || !r.trusted(peer) {
		return peer
	}
	hops := forwardedHops(req.Header.Values("X-Forwarded-For"))
	if len(hops) == 0 {
		return peer
	}
	for i := len(hops) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(hops[i])
		if err != nil {
			// A malformed hop breaks the chain: nothing to its left can be
			// attributed to a trusted proxy.
			return peer
		}
		addr = addr.Unmap()
		if !r.trusted(addr) {
			return addr
		}
		if i == 0 {
			return addr
		}
	}
	return peer
}

// String is Resolve formatted for logging and storage, or "" when the peer
// address could not be parsed.
func (r Resolver) String(req *http.Request) string {
	if a := r.Resolve(req); a.IsValid() {
		return a.String()
	}
	return ""
}

func (r Resolver) trusted(a netip.Addr) bool {
	for _, p := range r.Trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func parseHost(remoteAddr string) netip.Addr {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	addr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}

// forwardedHops flattens every X-Forwarded-For header (a proxy may add its
// own header line rather than append to an existing one) into one ordered
// list.
func forwardedHops(values []string) []string {
	var hops []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				hops = append(hops, part)
			}
		}
	}
	return hops
}
