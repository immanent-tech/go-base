// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package client

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/immanent-tech/go-base/config"
)

// New creates a new [resty.Client] with some default headers, redirects enabled (up to 3) and logging to the default
// [slog.Logger].
func New() *resty.Client {
	userAgent := "Go-Base/Unknown"
	if baseCfg, err := config.LoadAppConfig(); err == nil {
		userAgent = baseCfg.AppName + "/" + baseCfg.Version
	}
	return resty.New().
		SetHeader("User-Agent", userAgent).
		SetHeader("Accept", "*/*").
		SetHeader("Accept-Encoding", "gzip, deflate").
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(3)).
		SetLogger(&logger{Logger: slog.Default()})
}

type logger struct {
	*slog.Logger
}

func (l *logger) Errorf(format string, v ...any) {
	l.Error(fmt.Sprintf(format, v...))
}

func (l *logger) Warnf(format string, v ...any) {
	l.Warn(fmt.Sprintf(format, v...))
}

func (l *logger) Debugf(format string, v ...any) {
	l.Debug(fmt.Sprintf(format, v...))
}

var extraBlocked = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),   // CGNAT
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved
	netip.MustParsePrefix("2001:db8::/32"),   // IPv6 documentation
	netip.MustParsePrefix("64:ff9b::/96"),    // NAT64
}

func isPublicIP(ip netip.Addr) bool {
	ip = ip.Unmap() // handle ::ffff:10.0.0.1
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for p := range slices.Values(extraBlocked) {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

// IsPublicURL performs checks on the given [url.URL] to ensure that it is a public address.
func IsPublicURL(ctx context.Context, u *url.URL) (bool, error) {
	host := u.Hostname() // strips port and IPv6 brackets
	if host == "" {
		return false, fmt.Errorf("no host")
	}

	// Literal IP.
	if ip, err := netip.ParseAddr(host); err == nil {
		return isPublicIP(ip), nil
	}

	// Obvious non-public names.
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") ||
		strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".internal") ||
		!strings.Contains(h, ".") {
		return false, nil
	}

	// Resolve, and require ALL addresses to be public.
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", h)
	if err != nil {
		return false, err
	}
	if len(ips) == 0 {
		return false, fmt.Errorf("no addresses")
	}
	for ip := range slices.Values(ips) {
		if !isPublicIP(ip) {
			return false, nil
		}
	}
	return true, nil
}

// SecureDialer is a [net.Dialer] that performs some additional checks of an address before performing connection logic.
// It will ensure that the address is actually a public address. This also covers redirects, since every connection the
// client makes goes through the hook. Note that if you use a proxy, the dial goes to the proxy, so the check won't
// apply to the final destination.
var SecureDialer = &net.Dialer{
	Control: func(network, address string, c syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip, err := netip.ParseAddr(host)
		if err != nil || !isPublicIP(ip) {
			return fmt.Errorf("blocked address %s", host)
		}
		return nil
	},
	Timeout:   5 * time.Second,
	KeepAlive: 30 * time.Second,
}
