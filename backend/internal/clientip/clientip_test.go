package clientip_test

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"at.draab/familyfinances/internal/clientip"
)

func TestResolve(t *testing.T) {
	tenSlash8 := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	for _, tc := range []struct {
		name    string
		trusted []netip.Prefix
		remote  string
		xff     []string
		want    string
	}{
		{"no trusted proxies ignores the header", nil, "203.0.113.7:1234", []string{"198.51.100.1"}, "203.0.113.7"},
		{"trusted proxy forwards the client", tenSlash8, "10.0.0.2:1234", []string{"198.51.100.1"}, "198.51.100.1"},
		{"spoofed left-most entry is ignored", tenSlash8, "10.0.0.2:1234", []string{"1.2.3.4, 198.51.100.1"}, "198.51.100.1"},
		{"untrusted peer cannot forge", tenSlash8, "203.0.113.7:1234", []string{"198.51.100.1"}, "203.0.113.7"},
		{"chain of trusted proxies", tenSlash8, "10.0.0.2:1234", []string{"198.51.100.1, 10.1.1.1, 10.2.2.2"}, "198.51.100.1"},
		{"all hops trusted takes the left-most", tenSlash8, "10.0.0.2:1234", []string{"10.9.9.9, 10.1.1.1"}, "10.9.9.9"},
		{"several header lines are one list", tenSlash8, "10.0.0.2:1234", []string{"1.2.3.4", "198.51.100.1"}, "198.51.100.1"},
		{"no header from a trusted proxy", tenSlash8, "10.0.0.2:1234", nil, "10.0.0.2"},
		{"malformed hop falls back to the peer", tenSlash8, "10.0.0.2:1234", []string{"198.51.100.1, garbage"}, "10.0.0.2"},
		{"remote address without a port", nil, "203.0.113.7", nil, "203.0.113.7"},
		{"ipv6 peer", nil, "[2001:db8::1]:443", nil, "2001:db8::1"},
		{"ipv4-mapped ipv6 is unmapped", tenSlash8, "[::ffff:10.0.0.2]:1234", []string{"::ffff:198.51.100.1"}, "198.51.100.1"},
		{"unparsable peer", nil, "nonsense", nil, "invalid IP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			got := clientip.Resolver{Trusted: tc.trusted}.Resolve(req)
			if got.String() != tc.want {
				t.Errorf("Resolve = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestStringOfUnparsablePeerIsEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "nonsense"
	if got := (clientip.Resolver{}).String(req); got != "" {
		t.Errorf("String = %q, want empty", got)
	}
}
