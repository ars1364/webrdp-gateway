// Package netguard resolves user-supplied RDP targets and refuses addresses
// that would turn the gateway into a pivot into its own network (SSRF).
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"
)

var ErrForbiddenTarget = errors.New("target address is not allowed")

// blocked ranges beyond what netip's Is* helpers cover.
var extraBlocked = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),  // CGNAT
	netip.MustParsePrefix("192.0.0.0/24"),   // IETF protocol assignments
	netip.MustParsePrefix("198.18.0.0/15"),  // benchmarking
	netip.MustParsePrefix("169.254.0.0/16"), // link-local / cloud metadata
	netip.MustParsePrefix("fd00:ec2::/32"),  // AWS IMDS v6
	netip.MustParsePrefix("64:ff9b:1::/48"), // local NAT64
	netip.MustParsePrefix("240.0.0.0/4"),    // reserved
	netip.MustParsePrefix("255.255.255.255/32"),
}

func allowed(a netip.Addr, allowPrivate bool) bool {
	a = a.Unmap()
	if a.IsUnspecified() || a.IsLoopback() || a.IsMulticast() ||
		a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() || a.IsInterfaceLocalMulticast() {
		return false
	}
	for _, p := range extraBlocked {
		if p.Contains(a) {
			return false
		}
	}
	if a.IsPrivate() && !allowPrivate {
		return false
	}
	return true
}

// Resolve validates host (IP literal or DNS name) and returns one IP to dial.
// Every resolved address must be allowed, and the caller must connect to the
// returned IP (not the name) so DNS rebinding can't swap it afterwards.
func Resolve(ctx context.Context, host string, allowPrivate bool) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" || len(host) > 253 {
		return "", fmt.Errorf("invalid host")
	}
	if a, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		if !allowed(a, allowPrivate) {
			return "", ErrForbiddenTarget
		}
		return a.Unmap().String(), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addrs) == 0 {
		return "", fmt.Errorf("cannot resolve %q", host)
	}
	for _, a := range addrs {
		if !allowed(a, allowPrivate) {
			return "", ErrForbiddenTarget
		}
	}
	return addrs[0].Unmap().String(), nil
}
