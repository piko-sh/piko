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
	"encoding/binary"
	"net"
	"net/http"
	"net/netip"
	"strconv"

	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

var _ security_domain.RequestContextBinderAdapter = (*ipBinderAdapter)(nil)

const (
	// decimalBase is the number base used to format numbers as decimal strings.
	decimalBase = 10
)

// ipBinderAdapter binds CSRF tokens to client IP addresses.
// It implements RequestContextBinderAdapter by reading the already-resolved
// client IP from the request context (set by RealIPMiddleware), ensuring
// consistency with the trusted proxy rules.
type ipBinderAdapter struct{}

// GetBindingIdentifier extracts and normalises the client IP address from the
// request context. It uses the IP resolved by RealIPMiddleware to ensure
// consistency with rate limiting and other downstream consumers.
//
// Takes r (*http.Request) which provides the incoming HTTP request to extract
// the client IP from.
//
// Returns string which is the normalised client IP address, or
// "context_no_request" when r is nil.
func (*ipBinderAdapter) GetBindingIdentifier(r *http.Request) string {
	if r == nil {
		return "context_no_request"
	}

	clientIP := security_dto.ClientIPFromRequest(r)
	if clientIP == "" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return normaliseIP(r.RemoteAddr)
		}
		return normaliseIP(ip)
	}

	return normaliseIP(clientIP)
}

// NewIPBinderAdapter creates a new adapter that binds request context based on
// IP address.
//
// Returns security_domain.RequestContextBinderAdapter which binds request
// context using the client IP address.
func NewIPBinderAdapter() security_domain.RequestContextBinderAdapter {
	return &ipBinderAdapter{}
}

// normaliseIP converts an IP address string to a consistent, compact
// identifier.
//
// When the input is an IPv4 address, returns the uint32 representation
// as a string (e.g., "2035327534"). When the input is an IPv6 address,
// returns the canonical string representation (e.g., "2001:db8::1").
// If the input is not a valid IP, returns the original string.
//
// Takes ipString (string) which is the IP address to normalise.
//
// Returns string which is the normalised IP identifier.
func normaliseIP(ipString string) string {
	addr, err := netip.ParseAddr(ipString)
	if err != nil {
		return ipString
	}

	if addr.Is4() {
		b := addr.As4()
		ipAsUint32 := binary.BigEndian.Uint32(b[:])
		return strconv.FormatUint(uint64(ipAsUint32), decimalBase)
	}

	return addr.String()
}
