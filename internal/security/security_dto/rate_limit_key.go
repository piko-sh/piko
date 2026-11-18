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

package security_dto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

const (
	// RateLimitKeyMaxLength caps a single segment of a rate limit key. A caller-supplied
	// identity is otherwise unbounded, so a client choosing a megabyte of key material would
	// make every stored bucket that size.
	RateLimitKeyMaxLength = 64

	// DefaultIPv6RateLimitPrefixBits is the IPv6 network prefix length that identifies one
	// caller. A single subscriber is usually handed a whole /64, so keying on the full
	// address would let one client rotate through billions of fresh buckets.
	DefaultIPv6RateLimitPrefixBits = 64

	// maxIPv6PrefixBits is the length of an IPv6 address in bits.
	maxIPv6PrefixBits = 128

	// ipv4CompatiblePrefixBytes is how many leading zero bytes mark an IPv4-compatible IPv6
	// address (::a.b.c.d).
	ipv4CompatiblePrefixBytes = 12

	// teredoClientOffset is where a Teredo address holds its client's IPv4 address.
	teredoClientOffset = 12

	// teredoClientMask is XORed with each byte of a Teredo client address, which Teredo
	// stores inverted.
	teredoClientMask = 0xff

	// ipv4AddressBytes is the length of an IPv4 address in bytes.
	ipv4AddressBytes = 4
)

var (
	// teredoPrefix is the Teredo range, 2001::/32, whose addresses carry their client's IPv4
	// address in their last 32 bits.
	teredoPrefix = netip.MustParsePrefix("2001::/32")
)

var (
	// ErrInvalidIPv6PrefixBits is returned when an IPv6 rate limit prefix length is outside
	// 1 to 128.
	ErrInvalidIPv6PrefixBits = errors.New("IPv6 rate limit prefix bits must be between 1 and 128")
)

// RateLimitKeyOption changes how SanitiseRateLimitKey builds a key segment.
type RateLimitKeyOption func(*rateLimitKeyOptions)

// rateLimitKeyOptions holds the settings SanitiseRateLimitKey applies.
type rateLimitKeyOptions struct {
	// ipv6PrefixBits is the number of leading IPv6 bits kept in the key.
	ipv6PrefixBits int
}

// WithIPv6PrefixBits sets how many leading bits of an IPv6 address form the key.
//
// Values outside 1 to 128 leave the current setting unchanged.
//
// Takes bits (int) which is the prefix length to keep.
//
// Returns RateLimitKeyOption which applies the prefix length.
func WithIPv6PrefixBits(bits int) RateLimitKeyOption {
	return func(options *rateLimitKeyOptions) {
		if ValidateIPv6PrefixBits(bits) == nil {
			options.ipv6PrefixBits = bits
		}
	}
}

// SanitiseRateLimitKey normalises a value for use as one segment of a rate limit key.
//
// An IP address is written in its canonical form. An IPv4 or IPv4-mapped address is kept
// whole, while an IPv6 address is cut down to its network prefix so every address in one
// allocation shares a single key. Any other value is replaced by its SHA-256 digest.
//
// Some IPv6 addresses do not fit the prefix rule. The loopback address ::1 is kept whole,
// since its /64 would be shared with the unspecified address. An IPv4-compatible address
// (::a.b.c.d) is keyed as its IPv4 address. A Teredo address (2001::/32) is keyed as the
// IPv4 address of its client, since its /64 is shared by every client of one Teredo
// server, while its full address changes with the client's port.
//
// Takes value (string) which is the raw identity candidate.
// Takes opts (...RateLimitKeyOption) which change the IPv6 prefix length.
//
// Returns string which is the sanitised segment.
func SanitiseRateLimitKey(value string, opts ...RateLimitKeyOption) string {
	trimmed := strings.TrimSpace(value)

	if address, err := netip.ParseAddr(trimmed); err == nil {
		options := rateLimitKeyOptions{ipv6PrefixBits: DefaultIPv6RateLimitPrefixBits}
		for _, opt := range opts {
			opt(&options)
		}

		return truncateRateLimitSegment(canonicalRateLimitAddress(address, options.ipv6PrefixBits))
	}

	digest := sha256.Sum256([]byte(trimmed))

	return truncateRateLimitSegment(hex.EncodeToString(digest[:]))
}

// truncateRateLimitSegment caps a segment at RateLimitKeyMaxLength.
//
// Takes segment (string) which is the segment to cap.
//
// Returns string which is at most RateLimitKeyMaxLength bytes.
func truncateRateLimitSegment(segment string) string {
	if len(segment) > RateLimitKeyMaxLength {
		return segment[:RateLimitKeyMaxLength]
	}

	return segment
}

// canonicalRateLimitAddress writes an address in the form used inside a key.
//
// Takes address (netip.Addr) which is the parsed caller address.
// Takes ipv6PrefixBits (int) which is the number of leading IPv6 bits to keep.
//
// Returns string which is the canonical address or IPv6 network.
func canonicalRateLimitAddress(address netip.Addr, ipv6PrefixBits int) string {
	address = embeddedIPv4Client(address.Unmap().WithZone(""))
	if address.Is4() || address.IsLoopback() {
		return address.String()
	}

	prefix, err := address.Prefix(ipv6PrefixBits)
	if err != nil {
		return address.String()
	}

	return prefix.Addr().String()
}

// embeddedIPv4Client returns the IPv4 address an IPv6 address stands for, when it is an
// IPv4-compatible or a Teredo address.
//
// Takes address (netip.Addr) which is an unmapped address without a zone.
//
// Returns netip.Addr which is the embedded IPv4 address, or address unchanged.
func embeddedIPv4Client(address netip.Addr) netip.Addr {
	if !address.Is6() {
		return address
	}

	raw := address.As16()
	if teredoPrefix.Contains(address) {
		client := [ipv4AddressBytes]byte(raw[teredoClientOffset:])
		for index := range client {
			client[index] ^= teredoClientMask
		}
		return netip.AddrFrom4(client)
	}

	if isIPv4Compatible(address) {
		return netip.AddrFrom4([ipv4AddressBytes]byte(raw[ipv4CompatiblePrefixBytes:]))
	}
	return address
}

// isIPv4Compatible reports whether an IPv6 address has the deprecated ::a.b.c.d form.
//
// The unspecified and loopback addresses share that form but are not IPv4 clients.
//
// Takes address (netip.Addr) which is the IPv6 address.
//
// Returns bool which is true for an IPv4-compatible address.
func isIPv4Compatible(address netip.Addr) bool {
	if address.IsUnspecified() || address.IsLoopback() {
		return false
	}
	raw := address.As16()
	for _, value := range raw[:ipv4CompatiblePrefixBytes] {
		if value != 0 {
			return false
		}
	}
	return true
}

// ValidateIPv6PrefixBits checks that an IPv6 rate limit prefix length is usable.
//
// Takes bits (int) which is the prefix length to check.
//
// Returns error wrapping ErrInvalidIPv6PrefixBits when bits is outside 1 to 128.
func ValidateIPv6PrefixBits(bits int) error {
	if bits < 1 || bits > maxIPv6PrefixBits {
		return fmt.Errorf("%w: got %d", ErrInvalidIPv6PrefixBits, bits)
	}
	return nil
}
