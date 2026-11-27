package httpserver_test

import (
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// ClientIP trusts X-Forwarded-For only from a trusted proxy, and then takes its rightmost untrusted hop.
func TestClientIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	tCases := []struct {
		name, remote, want string
		xff                []string // one entry per X-Forwarded-For header line
	}{
		{
			name:   "an untrusted peer's header is ignored",
			remote: "203.0.113.7:1234",
			xff:    []string{"10.1.1.1"},
			want:   "203.0.113.7",
		},
		{
			name:   "a trusted proxy's rightmost untrusted hop is the client",
			remote: "10.0.0.5:443",
			xff:    []string{"198.51.100.9, 10.0.0.7"},
			want:   "198.51.100.9",
		},
		{
			name:   "entries left of the client, written by the client, are not read",
			remote: "10.0.0.5:443",
			xff:    []string{"1.1.1.1, 198.51.100.9, 10.0.0.7"},
			want:   "198.51.100.9",
		},
		{
			name:   "several header lines read as one list, in order",
			remote: "10.0.0.5:443",
			xff:    []string{"1.1.1.1", "198.51.100.9, 10.0.0.7"},
			want:   "198.51.100.9",
		},
		{name: "a trusted proxy without the header is the client", remote: "10.0.0.5:443", want: "10.0.0.5"},
		{
			name:   "when every hop is trusted the leftmost one is the client",
			remote: "10.0.0.5:443",
			xff:    []string{"10.0.0.6, 10.0.0.7"},
			want:   "10.0.0.6",
		},
		{
			name:   "an unparsable hop falls back to the peer",
			remote: "10.0.0.5:443",
			xff:    []string{"not-an-ip"},
			want:   "10.0.0.5",
		},
		{
			name:   "IPv4-mapped IPv6 addresses are read as IPv4",
			remote: "[::ffff:10.0.0.5]:443",
			xff:    []string{"::ffff:198.51.100.9"},
			want:   "198.51.100.9",
		},
		{name: "an IPv6 peer is the client", remote: "[2001:db8::1]:443", want: "2001:db8::1"},
		{name: "a peer without a port is still read", remote: "203.0.113.7", want: "203.0.113.7"},
		{name: "a peer that is not an address is returned as it is", remote: "@unix", want: "@unix"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			request.RemoteAddr = tc.remote
			for _, value := range tc.xff {
				request.Header.Add("X-Forwarded-For", value)
			}
			if got := httpserver.ClientIP(request, trusted); got != tc.want {
				t.Errorf("ClientIP = %q, want %q", got, tc.want)
			}
		})
	}
}
