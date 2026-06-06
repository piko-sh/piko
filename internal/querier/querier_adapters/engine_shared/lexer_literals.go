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

package engine_shared

import (
	"strings"
)

// ScanDoubledDelimiter scans a literal opened by delimiter at position, where the
// delimiter is escaped within the literal by doubling it (the SQL-standard convention for
// string literals and quoted identifiers).
//
// The opening delimiter is expected at position. The returned body has the surrounding
// delimiters removed and each doubled delimiter collapsed.
//
// Takes input (string) which is the source being scanned.
// Takes position (int) which indexes the opening delimiter.
// Takes delimiter (byte) which opens and closes the literal and which is doubled to
// escape itself.
//
// Returns string which is the unescaped literal body without its delimiters.
// Returns int which is the offset after the closing delimiter, or the length of input
// when the literal was left unterminated.
// Returns bool which is true when the literal was closed, and false when input ended
// first.
func ScanDoubledDelimiter(input string, position int, delimiter byte) (string, int, bool) {
	position++

	var builder strings.Builder
	for position < len(input) {
		character := input[position]
		if character == delimiter {
			position++
			if position < len(input) && input[position] == delimiter {
				builder.WriteByte(delimiter)
				position++
				continue
			}
			return builder.String(), position, true
		}
		builder.WriteByte(character)
		position++
	}

	return "", position, false
}
