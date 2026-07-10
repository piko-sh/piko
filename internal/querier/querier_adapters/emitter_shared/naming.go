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

package emitter_shared

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// commonInitialisms maps lowercase initialisations to their canonical upper-case forms
	// per Go naming conventions.
	commonInitialisms = map[string]string{
		"id":    "ID",
		"ids":   "IDs",
		"url":   "URL",
		"uri":   "URI",
		"api":   "API",
		"sql":   "SQL",
		"http":  "HTTP",
		"https": "HTTPS",
		"ip":    "IP",
		"css":   "CSS",
		"html":  "HTML",
		"json":  "JSON",
		"xml":   "XML",
		"ssh":   "SSH",
		"tls":   "TLS",
		"tcp":   "TCP",
		"udp":   "UDP",
		"cpu":   "CPU",
		"gpu":   "GPU",
		"ram":   "RAM",
		"uuid":  "UUID",
		"uid":   "UID",
		"ascii": "ASCII",
		"utf8":  "UTF8",
		"eof":   "EOF",
		"ttl":   "TTL",
		"acl":   "ACL",
		"pk":    "PK",
		"fk":    "FK",
	}
)

// SnakeToPascalCase converts a snake_case SQL identifier to PascalCase Go identifier,
// applying Go initialism conventions.
//
// A SQL name need not be a legal Go identifier stem: quoted identifiers and aliases such
// as "2fa_enabled" are valid SQL but their naive PascalCase form ("2faEnabled") starts
// with a digit and would not compile. The result is therefore passed through
// sanitiseGoIdentifier so a leading-digit (or otherwise empty) stem is prefixed with an
// underscore.
//
// Takes name (string) which is the snake_case identifier to convert.
//
// Returns string which is the PascalCase Go identifier.
func SnakeToPascalCase(name string) string {
	segments := strings.Split(name, "_")
	var builder strings.Builder
	builder.Grow(len(name))

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		lower := strings.ToLower(segment)
		if canonical, exists := commonInitialisms[lower]; exists {
			builder.WriteString(canonical)
			continue
		}

		if isMixedCaseSegment(segment) {
			runes := []rune(segment)
			runes[0] = unicode.ToUpper(runes[0])
			builder.WriteString(string(runes))
			continue
		}

		runes := []rune(lower)
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}

	return sanitiseGoIdentifier(builder.String())
}

// SnakeToCamelCase converts a snake_case SQL identifier to camelCase Go identifier,
// applying Go initialism conventions for non-leading segments.
//
// As with SnakeToPascalCase, the result is passed through sanitiseGoIdentifier so a SQL
// name whose camelCase form begins with a digit (or is empty) becomes a legal Go
// identifier rather than non-compiling source.
//
// Takes name (string) which is the snake_case identifier to convert.
//
// Returns string which is the camelCase Go identifier.
func SnakeToCamelCase(name string) string {
	segments := strings.Split(name, "_")
	var builder strings.Builder
	builder.Grow(len(name))

	for index, segment := range segments {
		if segment == "" {
			continue
		}

		lower := strings.ToLower(segment)

		if index == 0 {
			builder.WriteString(lower)
			continue
		}

		if canonical, exists := commonInitialisms[lower]; exists {
			builder.WriteString(canonical)
			continue
		}

		runes := []rune(lower)
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}

	return sanitiseGoIdentifier(builder.String())
}

// isMixedCaseSegment reports whether a segment already carries both an upper-case and a
// lower-case letter, marking it as a camelCase token (such as "jobCount") whose interior
// capitalisation must be preserved rather than folded away.
//
// Takes segment (string) which is a single underscore-delimited identifier segment.
//
// Returns bool which is true when the segment mixes upper- and lower-case letters.
func isMixedCaseSegment(segment string) bool {
	var hasLower, hasUpper bool
	for _, character := range segment {
		switch {
		case unicode.IsLower(character):
			hasLower = true
		case unicode.IsUpper(character):
			hasUpper = true
		}
	}
	return hasLower && hasUpper
}

// sanitiseGoIdentifier ensures a converted name is a legal Go identifier stem.
//
// A Go identifier may not start with a digit, yet a quoted SQL name such as "2fa_enabled"
// or "123" produces exactly such a stem. The name is prefixed with an underscore when it
// begins with a digit, leaving every already-valid name untouched. An empty name is left
// empty so callers that intentionally pass an empty string, for example an anonymous
// field or result, are not given a spurious underscore.
//
// Takes name (string) which is the candidate Go identifier.
//
// Returns string which is a legal Go identifier stem.
func sanitiseGoIdentifier(name string) string {
	if name == "" {
		return ""
	}
	if first, _ := utf8.DecodeRuneInString(name); unicode.IsDigit(first) {
		return "_" + name
	}
	return name
}

// DisambiguateGoFieldNames converts an ordered list of snake_case SQL names into the
// matching ordered list of PascalCase Go field identifiers, suffixing collisions.
//
// Takes names ([]string) which are the source SQL names in declaration order.
//
// Returns []string which are the disambiguated PascalCase identifiers in the same order.
func DisambiguateGoFieldNames(names []string) []string {
	return disambiguateGoFieldNames(names, SnakeToPascalCase)
}

// DisambiguateGoFieldNamesCamelCase converts an ordered list of snake_case SQL names into
// the matching ordered list of camelCase Go identifiers, suffixing collisions.
//
// Takes names ([]string) which are the source SQL names in declaration order.
//
// Returns []string which are the disambiguated camelCase identifiers in the same order.
func DisambiguateGoFieldNamesCamelCase(names []string) []string {
	return disambiguateGoFieldNames(names, SnakeToCamelCase)
}

// disambiguateGoFieldNames converts an ordered list of snake_case SQL names into the
// matching ordered list of Go identifiers using the supplied case converter.
//
// A numeric suffix is appended to any name that would otherwise collide with an earlier
// one. Distinct SQL names can fold onto the same Go identifier, for example "foo_bar" and
// "foo__bar" both become "FooBar", which would be a duplicate-field compile error. The
// suffix keeps each field unique while preserving source order so the struct fields and
// the scan targets derived from the same ordered list stay in lockstep.
//
// The suffix search continues past a suffixed candidate that itself collides, so a
// literal column whose converted form equals an already-emitted suffix (for example
// "foo2" alongside a second "foo" that became "Foo2") still receives a distinct
// identifier.
//
// Takes names ([]string) which are the source SQL names in declaration order.
// Takes convert (func(string) string) which maps a snake_case name to its Go identifier.
//
// Returns []string which are the disambiguated Go identifiers in the same order.
func disambiguateGoFieldNames(names []string, convert func(string) string) []string {
	result := make([]string, len(names))
	seen := make(map[string]struct{}, len(names))
	for index, name := range names {
		converted := convert(name)
		candidate := converted
		for suffix := 2; ; suffix++ {
			if _, exists := seen[candidate]; !exists {
				break
			}
			candidate = converted + strconv.Itoa(suffix)
		}
		seen[candidate] = struct{}{}
		result[index] = candidate
	}
	return result
}
