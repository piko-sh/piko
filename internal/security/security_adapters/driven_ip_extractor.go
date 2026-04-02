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
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"piko.sh/piko/internal/security/security_domain"
)

const (
	// ipv4CIDRBits is the number of bits in an IPv4 CIDR mask (/32).
	ipv4CIDRBits = 32

	// ipv6CIDRBits is the number of bits in an IPv6 CIDR mask (/128).
	ipv6CIDRBits = 128
)

// trustedProxyIPExtractor implements security_domain.ClientIPExtractor.
// It extracts the real client IP address from HTTP requests, only trusting
// forwarding headers (X-Forwarded-For, X-Real-IP, CF-Connecting-IP) when
// the request comes from a trusted proxy CIDR range.
type trustedProxyIPExtractor struct {
	// trustedCIDRs holds the CIDR ranges that are treated as trusted proxies.
	// Uses netip.Prefix for zero-allocation IP containment checks.
	trustedCIDRs []netip.Prefix

	// cloudflareEnabled controls whether the CF-Connecting-IP header from
	// trusted proxies is accepted. When false, the header is ignored to
	// prevent IP spoofing in non-Cloudflare deployments.
	cloudflareEnabled bool
}

// ExtractClientIP returns the real client IP address from the request.
//
// The extraction logic follows this priority:
//  1. If the direct connection IP is NOT from a trusted proxy, return it
//     directly (do not trust any forwarding headers from untrusted sources).
//  2. If the direct connection IS from a trusted proxy, check headers in order:
//     CF-Connecting-IP (only when cloudflareEnabled), X-Real-IP (nginx default),
//     X-Forwarded-For (rightmost non-trusted IP).
//  3. Fall back to the direct connection IP if no headers are present.
//
// Takes r (*http.Request) which provides the incoming request to extract the
// client IP from.
//
// Returns string which is the resolved client IP address.
func (e *trustedProxyIPExtractor) ExtractClientIP(r *http.Request) string {
	remoteIP, parsedRemote := parseRemoteAddrWithIP(r.RemoteAddr)

	if !e.isTrustedAddr(parsedRemote) {
		return remoteIP
	}

	if e.cloudflareEnabled {
		if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
			trimmed := strings.TrimSpace(cfIP)
			if _, err := netip.ParseAddr(trimmed); err == nil {
				return trimmed
			}
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		trimmed := strings.TrimSpace(realIP)
		if _, err := netip.ParseAddr(trimmed); err == nil {
			return trimmed
		}
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := e.extractFromXFF(xff); ip != "" {
			return ip
		}
	}

	return remoteIP
}

// IsTrustedProxy checks if the given IP address is within any trusted proxy
// CIDR range.
//
// Takes ip (string) which is the IP address to check.
//
// Returns bool which is true if the IP is within a trusted CIDR range.
func (e *trustedProxyIPExtractor) IsTrustedProxy(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	return e.isTrustedAddr(addr)
}

// extractFromXFF walks X-Forwarded-For from right to left, skipping IPs that
// are within trusted proxy CIDR ranges, and returns the first non-trusted IP.
// This prevents clients from spoofing their IP by prepending entries to XFF
// when the proxy appends.
//
// The scan uses index arithmetic over the raw header string to avoid
// allocating a []string slice from strings.Split.
//
// Takes xff (string) which is the raw X-Forwarded-For header value.
//
// Returns string which is the first non-trusted IP found scanning right to
// left, or empty string if all IPs are trusted or invalid.
func (e *trustedProxyIPExtractor) extractFromXFF(xff string) string {
	end := len(xff)
	for end > 0 {
		start := end
		for start > 0 && xff[start-1] != ',' {
			start--
		}

		if ip, ok := e.parseXFFSegment(xff, start, end); ok {
			return ip
		}

		end = start
		if end > 0 {
			end--
		}
	}

	return ""
}

// parseXFFSegment trims whitespace from a single segment of the
// X-Forwarded-For header and checks whether it contains a non-trusted IP
// address.
//
// Takes xff (string) which is the full header value.
// Takes start (int) which is the inclusive start index of the segment.
// Takes end (int) which is the exclusive end index of the segment.
//
// Returns string which is the candidate IP if it is non-trusted.
// Returns bool which is true when a non-trusted IP was found.
func (e *trustedProxyIPExtractor) parseXFFSegment(xff string, start, end int) (string, bool) {
	lo := start
	hi := end
	for lo < hi && xff[lo] == ' ' {
		lo++
	}
	for hi > lo && xff[hi-1] == ' ' {
		hi--
	}

	if lo >= hi {
		return "", false
	}

	candidate := xff[lo:hi]
	addr, err := netip.ParseAddr(candidate)
	if err != nil {
		return "", false
	}
	if e.isTrustedAddr(addr) {
		return "", false
	}
	return candidate, true
}

// isTrustedAddr checks if a pre-parsed IP is within any trusted proxy
// CIDR range. Uses netip.Prefix.Contains for zero-allocation containment
// checks.
//
// Takes addr (netip.Addr) which is the pre-parsed IP address to check.
//
// Returns bool which is true if the IP is within a trusted CIDR range.
func (e *trustedProxyIPExtractor) isTrustedAddr(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}

	for _, prefix := range e.trustedCIDRs {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// NewTrustedProxyIPExtractor creates a new IP extractor with the given trusted
// proxy CIDR ranges.
//
// Takes trustedProxies ([]string) which specifies CIDR ranges or IP addresses
// for proxies that are trusted. Each entry must be a valid CIDR (e.g.
// "10.0.0.0/8") or a valid IP address (e.g. "10.0.0.1", auto-wrapped to /32
// or /128).
// Takes cloudflareEnabled (bool) which controls whether CF-Connecting-IP is
// trusted from trusted proxies.
//
// Returns security_domain.ClientIPExtractor which extracts client IPs from
// requests, taking the trusted proxy settings into account.
// Returns error when any trusted proxy entry is not a valid CIDR or IP.
func NewTrustedProxyIPExtractor(trustedProxies []string, cloudflareEnabled bool) (security_domain.ClientIPExtractor, error) {
	prefixes := make([]netip.Prefix, 0, len(trustedProxies))

	for _, entry := range trustedProxies {
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			prefixes = append(prefixes, prefix.Masked())
			continue
		}

		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy entry %q: not a valid CIDR or IP address", entry)
		}

		bits := ipv4CIDRBits
		if addr.Is6() {
			bits = ipv6CIDRBits
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr, bits))
	}

	return &trustedProxyIPExtractor{
		trustedCIDRs:      prefixes,
		cloudflareEnabled: cloudflareEnabled,
	}, nil
}

// parseRemoteAddrWithIP extracts the IP address from an address string,
// returning both string and parsed forms to avoid re-parsing for trusted
// proxy checks.
//
// Takes remoteAddr (string) which is the address to parse.
//
// Returns string which is the IP address portion.
// Returns netip.Addr which is the parsed IP, or zero Addr if parsing fails.
func parseRemoteAddrWithIP(remoteAddr string) (string, netip.Addr) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		if addr, parseErr := netip.ParseAddr(remoteAddr); parseErr == nil {
			return remoteAddr, addr
		}
		return remoteAddr, netip.Addr{}
	}

	addr, _ := netip.ParseAddr(host)
	return host, addr
}
