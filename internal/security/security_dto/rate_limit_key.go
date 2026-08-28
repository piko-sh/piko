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
	"net/netip"
	"strings"
)

const (
	// RateLimitKeyMaxLength caps a single segment of a rate limit key. A caller-supplied
	// identity is otherwise unbounded, so a client choosing a megabyte of key material would
	// make every stored bucket that size.
	RateLimitKeyMaxLength = 64
)

// SanitiseRateLimitKey normalises a value for use as one segment of a rate limit key.
//
// Takes value (string) which is the raw identity candidate.
//
// Returns string which is the sanitised segment.
func SanitiseRateLimitKey(value string) string {
	trimmed := strings.TrimSpace(value)

	if address, err := netip.ParseAddr(trimmed); err == nil {
		return truncateRateLimitSegment(address.Unmap().WithZone("").String())
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
