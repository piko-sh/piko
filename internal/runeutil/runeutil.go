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

package runeutil

import (
	"strings"
	"unicode/utf8"
)

const (
	// ellipsis marks text that TruncateWithEllipsis shortened.
	ellipsis = "..."

	// ellipsisRunes is the length of ellipsis, in runes.
	ellipsisRunes = len(ellipsis)
)

// Truncate shortens s to at most maxRunes runes.
//
// Truncate replaces each invalid UTF-8 byte with U+FFFD, counting each replacement as one
// rune. The result is always valid UTF-8.
//
// Takes s (string) which is the text to shorten.
// Takes maxRunes (int) which is the most runes the result may hold. Zero or less gives an
// empty string.
//
// Returns string which is s unchanged when it is valid and already fits, or its first
// maxRunes runes.
func Truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if !utf8.ValidString(s) {
		return truncateReplacingInvalid(s, maxRunes)
	}
	if len(s) <= maxRunes {
		return s
	}

	count := 0
	for index := range s {
		if count == maxRunes {
			return s[:index]
		}
		count++
	}
	return s
}

// TruncateWithEllipsis shortens s to at most maxRunes runes and marks a shortened result
// with "...".
//
// When maxRunes is too small to hold the ellipsis, the result is cut without one. Like
// Truncate, it always returns valid UTF-8.
//
// Takes s (string) which is the text to shorten.
// Takes maxRunes (int) which is the most runes the result may hold, ellipsis included.
// Zero or less gives an empty string.
//
// Returns string which is s unchanged when it is valid and already fits, or a shortened
// copy.
func TruncateWithEllipsis(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes || maxRunes <= ellipsisRunes {
		return Truncate(s, maxRunes)
	}
	return Truncate(s, maxRunes-ellipsisRunes) + ellipsis
}

// TruncateBytes shortens s to at most maxBytes bytes without cutting through a multi-byte
// UTF-8 sequence. A rune that would not fit whole is dropped.
//
// TruncateBytes preserves invalid UTF-8 bytes, so invalid input may produce an invalid
// result. It is suitable for byte-limited stores and protocols.
//
// Takes s (string) which is the text to shorten.
// Takes maxBytes (int) which is the most bytes the result may hold. Zero or less gives an
// empty string.
//
// Returns string which is s unchanged when it already fits, or its longest prefix that
// fits and ends on a rune boundary.
func TruncateBytes(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}

	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// truncateReplacingInvalid copies the first maxRunes runes of s, writing U+FFFD for each
// byte that is not part of a valid UTF-8 sequence.
//
// Takes s (string) which holds invalid UTF-8.
// Takes maxRunes (int) which is the most runes to copy, greater than zero.
//
// Returns string which is valid UTF-8.
func truncateReplacingInvalid(s string, maxRunes int) string {
	var builder strings.Builder
	builder.Grow(min(len(s), maxRunes) * utf8.UTFMax)

	count := 0
	for _, r := range s {
		if count == maxRunes {
			break
		}
		_, _ = builder.WriteRune(r)
		count++
	}
	return builder.String()
}
