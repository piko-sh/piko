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

// Package textlimit bounds diagnostic text on UTF-8 rune boundaries.
package textlimit

import (
	"unicode/utf8"
)

const (
	// Marker marks text shortened to fit a bound.
	Marker = "...[truncated]"
)

// Truncate shortens text to at most limit bytes on a rune boundary and marks the cut.
//
// Takes text (string) which is the text to bound.
// Takes limit (int) which is the largest kept byte length before the marker.
//
// Returns string which is the unchanged text when it fits, or its longest rune-aligned
// prefix within limit followed by Marker.
func Truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	end := max(limit, 0)
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + Marker
}

// CompletePrefix drops a trailing rune that was cut part-way through.
//
// Takes data ([]byte) which is the start of a longer UTF-8 text.
//
// Returns []byte which is data without its final incomplete rune.
func CompletePrefix(data []byte) []byte {
	for index := len(data) - 1; index >= 0 && index >= len(data)-utf8.UTFMax; index-- {
		if utf8.RuneStart(data[index]) {
			if !utf8.FullRune(data[index:]) {
				return data[:index]
			}
			break
		}
	}
	return data
}

// CompleteSuffix drops the continuation bytes of a leading rune that was cut part-way
// through.
//
// Takes data ([]byte) which is the end of a longer UTF-8 text.
//
// Returns []byte which is data from its first rune start.
func CompleteSuffix(data []byte) []byte {
	for index := range min(len(data), utf8.UTFMax) {
		if utf8.RuneStart(data[index]) {
			return data[index:]
		}
	}
	return data
}
