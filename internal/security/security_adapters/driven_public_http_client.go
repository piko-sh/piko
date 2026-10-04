// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package security_adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

const (
	// defaultPublicClientTimeout bounds a whole outbound request, including redirects and
	// reading the body, when no timeout option is given.
	defaultPublicClientTimeout = 30 * time.Second

	// defaultPublicClientMaxRedirects is the number of redirects followed when no redirect
	// option is given.
	defaultPublicClientMaxRedirects = 5

	// publicClientDialTimeout bounds establishing a single TCP connection.
	publicClientDialTimeout = 10 * time.Second

	// publicClientKeepAlive is the TCP keep-alive period for outbound connections.
	publicClientKeepAlive = 30 * time.Second

	// publicClientTLSHandshakeTimeout bounds the TLS handshake of an outbound connection.
	publicClientTLSHandshakeTimeout = 10 * time.Second

	// publicClientIdleConnTimeout is how long an idle pooled connection is kept open.
	publicClientIdleConnTimeout = 90 * time.Second

	// publicClientExpectContinueTimeout bounds the wait for a 100-continue response.
	publicClientExpectContinueTimeout = time.Second

	// publicClientMaxIdleConnections caps the pooled idle connections across all hosts.
	publicClientMaxIdleConnections = 16

	// publicClientMaxIdleConnectionsPerHost caps the pooled idle connections per host.
	publicClientMaxIdleConnectionsPerHost = 2

	// publicClientMaxResponseHeaderBytes caps the size of response headers (1 MiB).
	publicClientMaxResponseHeaderBytes = 1 << 20

	// sixToFourEmbeddedOffset is the byte offset of the IPv4 address embedded in a 6to4
	// (2002::/16) address.
	sixToFourEmbeddedOffset = 2

	// nat64EmbeddedOffset is the byte offset of the IPv4 address embedded in a NAT64
	// (64:ff9b::/96) address.
	nat64EmbeddedOffset = 12
)

var (
	// ErrNonPublicDestination is returned when an outbound request would connect to, or be
	// redirected to, an address that is not publicly routable.
	ErrNonPublicDestination = errors.New("outbound destination is not a public address")

	// errTooManyRedirects is returned when a request exceeds its redirect budget.
	errTooManyRedirects = errors.New("too many redirects")

	// errUnsupportedRedirectScheme is returned when a redirect leaves http or https.
	errUnsupportedRedirectScheme = errors.New("redirect uses an unsupported scheme")

	// globalUnicastIPv6 is the IANA-allocated IPv6 global unicast space; any IPv6 address
	// outside it is treated as non-public.
	globalUnicastIPv6 = netip.MustParsePrefix("2000::/3")

	// nat64Prefix is the well-known NAT64 prefix, whose addresses embed an IPv4 address.
	nat64Prefix = netip.MustParsePrefix("64:ff9b::/96")

	// sixToFourPrefix is the 6to4 prefix, whose addresses embed an IPv4 address.
	sixToFourPrefix = netip.MustParsePrefix("2002::/16")

	// nonPublicPrefixes lists special-purpose ranges that are not covered by the netip
	// classification helpers and must never be reached by outbound requests.
	nonPublicPrefixes = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("2001::/23"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("3fff::/20"),
	}
)

// PublicHTTPClientOption configures a client built by NewPublicHTTPClient.
type PublicHTTPClientOption func(*publicHTTPClientConfig)

// publicHTTPClientConfig holds the settings for a public-only HTTP client.
type publicHTTPClientConfig struct {
	// isPermitted reports whether a resolved destination address may be dialled.
	isPermitted func(netip.Addr) bool

	// timeout bounds a whole request, including redirects and reading the body.
	timeout time.Duration

	// maxRedirects is the number of redirects followed before a request fails.
	maxRedirects int
}

// WithPublicClientTimeout sets the overall timeout of each request made by the client.
//
// Non-positive values are ignored and the default of 30 seconds is kept.
//
// Takes timeout (time.Duration) which bounds a whole request, including redirects and
// reading the response body.
//
// Returns PublicHTTPClientOption which applies the timeout.
func WithPublicClientTimeout(timeout time.Duration) PublicHTTPClientOption {
	return func(config *publicHTTPClientConfig) {
		if timeout > 0 {
			config.timeout = timeout
		}
	}
}

// WithPublicClientMaxRedirects sets how many redirects the client follows.
//
// Zero refuses every redirect. Negative values are ignored and the default of 5 is kept.
//
// Takes count (int) which is the number of redirects followed before a request fails.
//
// Returns PublicHTTPClientOption which applies the redirect limit.
func WithPublicClientMaxRedirects(count int) PublicHTTPClientOption {
	return func(config *publicHTTPClientConfig) {
		if count >= 0 {
			config.maxRedirects = count
		}
	}
}

// NewPublicHTTPClient builds an HTTP client that only connects to publicly routable
// addresses.
//
// Takes options (...PublicHTTPClientOption) which override the default timeout and
// redirect limit.
//
// Returns *http.Client which refuses non-public destinations with an error wrapping
// ErrNonPublicDestination.
func NewPublicHTTPClient(options ...PublicHTTPClientOption) *http.Client {
	config := publicHTTPClientConfig{
		isPermitted:  isPublicAddress,
		timeout:      defaultPublicClientTimeout,
		maxRedirects: defaultPublicClientMaxRedirects,
	}
	for _, option := range options {
		option(&config)
	}
	return newGuardedHTTPClient(&config)
}

// newGuardedHTTPClient builds the HTTP client for a public-only configuration.
//
// Takes config (*publicHTTPClientConfig) which supplies the address policy, timeout and
// redirect limit.
//
// Returns *http.Client which applies the address policy to every dial and redirect.
func newGuardedHTTPClient(config *publicHTTPClientConfig) *http.Client {
	dialer := &net.Dialer{
		Timeout:        publicClientDialTimeout,
		KeepAlive:      publicClientKeepAlive,
		ControlContext: newAddressControl(config.isPermitted),
	}

	transport := &http.Transport{
		Proxy:                  nil,
		DialContext:            dialer.DialContext,
		ForceAttemptHTTP2:      true,
		MaxIdleConns:           publicClientMaxIdleConnections,
		MaxIdleConnsPerHost:    publicClientMaxIdleConnectionsPerHost,
		IdleConnTimeout:        publicClientIdleConnTimeout,
		TLSHandshakeTimeout:    publicClientTLSHandshakeTimeout,
		ResponseHeaderTimeout:  config.timeout,
		ExpectContinueTimeout:  publicClientExpectContinueTimeout,
		MaxResponseHeaderBytes: publicClientMaxResponseHeaderBytes,
	}

	return &http.Client{
		Transport:     transport,
		CheckRedirect: newRedirectPolicy(config.maxRedirects, config.isPermitted),
		Timeout:       config.timeout,
	}
}

// newAddressControl returns a dialer control that refuses connections to addresses the
// policy does not permit.
//
// Takes isPermitted (func(netip.Addr) bool) which reports whether an address may be
// dialled.
//
// Returns func(context.Context, string, string, syscall.RawConn) error which is suitable
// for net.Dialer.ControlContext.
func newAddressControl(
	isPermitted func(netip.Addr) bool,
) func(context.Context, string, string, syscall.RawConn) error {
	return func(_ context.Context, network, address string, _ syscall.RawConn) error {
		switch network {
		case "tcp", "tcp4", "tcp6":
		default:
			return fmt.Errorf("%w: network %q is not permitted", ErrNonPublicDestination, network)
		}

		addressPort, err := netip.ParseAddrPort(address)
		if err != nil {
			return fmt.Errorf("%w: parsing dial address %q: %w", ErrNonPublicDestination, address, err)
		}
		if !isPermitted(addressPort.Addr()) {
			return fmt.Errorf("%w: %s", ErrNonPublicDestination, addressPort.Addr())
		}
		return nil
	}
}

// newRedirectPolicy returns a redirect policy that limits the number of hops, keeps them
// on http or https, and refuses literal non-public addresses before dialling.
//
// Hostnames are checked again when each hop is dialled.
//
// Takes maxRedirects (int) which is the number of redirects followed.
// Takes isPermitted (func(netip.Addr) bool) which reports whether an address may be
// reached.
//
// Returns func(*http.Request, []*http.Request) error which is suitable for
// http.Client.CheckRedirect.
func newRedirectPolicy(
	maxRedirects int,
	isPermitted func(netip.Addr) bool,
) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, via []*http.Request) error {
		if len(via) > maxRedirects {
			return fmt.Errorf("%w: stopped after %d", errTooManyRedirects, maxRedirects)
		}
		if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
			return fmt.Errorf("%w: %q", errUnsupportedRedirectScheme, request.URL.Scheme)
		}
		if addr, err := netip.ParseAddr(request.URL.Hostname()); err == nil && !isPermitted(addr) {
			return fmt.Errorf("%w: redirect to %s", ErrNonPublicDestination, addr)
		}
		return nil
	}
}

// isPublicAddress reports whether an address is publicly routable.
//
// IPv4-mapped IPv6 addresses are judged by their IPv4 form, and NAT64 and 6to4 addresses
// by the IPv4 address they embed. IPv6 addresses outside the global unicast space are
// refused.
//
// Takes addr (netip.Addr) which is the address to classify.
//
// Returns bool which is true when the address may be reached by outbound requests.
func isPublicAddress(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}

	addr = embeddedIPv4(addr.WithZone("").Unmap())

	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return false
	}
	if addr.Is6() && !globalUnicastIPv6.Contains(addr) {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

// embeddedIPv4 returns the IPv4 address carried inside a NAT64 or 6to4 address.
//
// Takes addr (netip.Addr) which is the address to inspect.
//
// Returns netip.Addr which is the embedded IPv4 address, or addr unchanged when it does
// not embed one.
func embeddedIPv4(addr netip.Addr) netip.Addr {
	switch {
	case nat64Prefix.Contains(addr):
		bytes := addr.As16()
		return netip.AddrFrom4([4]byte(bytes[nat64EmbeddedOffset:]))
	case sixToFourPrefix.Contains(addr):
		bytes := addr.As16()
		return netip.AddrFrom4([4]byte(bytes[sixToFourEmbeddedOffset : sixToFourEmbeddedOffset+net.IPv4len]))
	default:
		return addr
	}
}
