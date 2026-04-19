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

package route_pattern

import "strings"

// TrailingSegment describes the structure of a parsed trailing parameter.
type TrailingSegment struct {
	// Prefix is the substring of the route preceding the open brace (`{`).
	Prefix string

	// Name is the parameter name; never empty when Found is true.
	Name string

	// Regex is the segment after the colon (e.g. ".+", "[a-z]+").
	//
	// Empty when the original pattern had no colon or the regex itself was
	// empty.
	Regex string

	// HasRegex is true when the original pattern contained a colon.
	HasRegex bool

	// Found is true when the route pattern ends with a `{...}` segment whose
	// brace structure is well-formed and whose name is non-empty.
	Found bool
}

// ParseTrailing parses a route pattern's trailing named-parameter segment.
//
// Patterns such as "/blog/{slug}", "/docs/{slug:.+}" and
// "/files/{path:[a-zA-Z0-9/_-]+}" are recognised. Patterns without a trailing
// `{...}` segment, or whose brace structure is malformed (unbalanced, nested,
// empty name) are returned with Found=false.
//
// Brace nesting is handled by scanning right-to-left and counting closing
// braces relative to opens, so a regex such as "{name:a{2,4}}" is parsed
// correctly. Empty-regex segments such as "{name:}" are still treated as
// having a regex (HasRegex=true, Regex="") so the chi translator can decide
// whether to accept them.
//
// Takes pattern (string) which is the route pattern to inspect.
//
// Returns TrailingSegment which describes the parsed segment.
func ParseTrailing(pattern string) TrailingSegment {
	if !strings.HasSuffix(pattern, "}") {
		return TrailingSegment{}
	}

	depth := 0
	openIndex := -1
	for index := len(pattern) - 1; index >= 0; index-- {
		switch pattern[index] {
		case '}':
			depth++
		case '{':
			depth--
			if depth == 0 {
				openIndex = index
			}
		}
		if openIndex >= 0 {
			break
		}
	}
	if openIndex < 0 {
		return TrailingSegment{}
	}

	inside := pattern[openIndex+1 : len(pattern)-1]
	prefix := pattern[:openIndex]

	name, regex, hasRegex := strings.Cut(inside, ":")
	if name == "" {
		return TrailingSegment{}
	}
	return TrailingSegment{
		Prefix:   prefix,
		Name:     name,
		Regex:    regex,
		HasRegex: hasRegex,
		Found:    true,
	}
}
