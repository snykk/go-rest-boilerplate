package httpserver

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// ClientIP returns the address of the client that sent r. It is the peer's address unless the peer is
// inside a trusted prefix. Behind a trusted proxy it is the rightmost X-Forwarded-For entry outside the
// trusted prefixes, or the leftmost entry when every entry is trusted. An empty header, or an unparsable
// entry met while walking from the right, gives the peer's address. Entries left of the client's are
// written by the client and are never read.
func ClientIP(r *http.Request, trusted []netip.Prefix) string {
	peer, ok := parseRemote(r.RemoteAddr)
	if !ok {
		return r.RemoteAddr
	}
	if !inAny(peer, trusted) {
		return peer.String()
	}

	var hops []string
	for _, value := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(value, ",")...)
	}
	if len(hops) == 0 {
		return peer.String()
	}

	var hop netip.Addr
	for _, entry := range slices.Backward(hops) {
		address, err := netip.ParseAddr(strings.TrimSpace(entry))
		if err != nil {
			return peer.String()
		}
		hop = address.Unmap()
		if !inAny(hop, trusted) {
			return hop.String()
		}
	}
	return hop.String()
}

func parseRemote(remote string) (netip.Addr, bool) {
	if addressPort, err := netip.ParseAddrPort(remote); err == nil {
		return addressPort.Addr().Unmap(), true
	}
	if address, err := netip.ParseAddr(remote); err == nil {
		return address.Unmap(), true
	}
	return netip.Addr{}, false
}

func inAny(address netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
