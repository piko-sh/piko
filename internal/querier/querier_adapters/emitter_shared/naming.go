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
	"strings"
	"unicode"
)

// commonInitialisms maps lowercase initialisations to their canonical
// upper-case forms per Go naming conventions.
var commonInitialisms = map[string]string{
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

// SnakeToPascalCase converts a snake_case SQL identifier to PascalCase Go
// identifier, applying Go initialism conventions.
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

		runes := []rune(lower)
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}

	return builder.String()
}

// SnakeToCamelCase converts a snake_case SQL identifier to camelCase Go
// identifier, applying Go initialism conventions for non-leading segments.
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

	return builder.String()
}
