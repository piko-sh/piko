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

package premailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkipLink(t *testing.T) {
	testCases := []struct {
		name     string
		href     string
		expected bool
	}{
		{name: "Empty href", href: "", expected: true},
		{name: "Anchor only", href: "#section", expected: true},
		{name: "Anchor with ID", href: "#top", expected: true},
		{name: "javascript: protocol", href: "javascript:void(0)", expected: true},
		{name: "javascript: with code", href: "javascript:alert('test')", expected: true},
		{name: "JavaScript mixed case", href: "JavaScript:alert(1)", expected: true},
		{name: "mailto: link", href: "mailto:test@example.com", expected: true},
		{name: "Mailto mixed case", href: "MailTo:test@example.com", expected: true},
		{name: "tel: link", href: "tel:+1234567890", expected: true},
		{name: "Tel mixed case", href: "TEL:+1234567890", expected: true},
		{name: "sms: link", href: "sms:+1234567890", expected: true},
		{name: "SMS mixed case", href: "SMS:+1234567890", expected: true},
		{name: "data: URI", href: "data:image/png;base64,iVBORw0KG", expected: true},
		{name: "Data URI mixed case", href: "Data:text/html,<h1>Hi</h1>", expected: true},
		{name: "HTTP URL", href: "http://example.com", expected: false},
		{name: "HTTPS URL", href: "https://example.com", expected: false},
		{name: "HTTP mixed case", href: "HTTP://example.com", expected: false},
		{name: "HTTPS mixed case", href: "HTTPS://example.com", expected: false},
		{name: "HTTPS with path", href: "https://example.com/path", expected: false},
		{name: "HTTPS with query", href: "https://example.com?key=value", expected: false},
		{name: "Relative URL with path", href: "/products/item", expected: false},
		{name: "Relative URL with query", href: "/search?q=test", expected: false},
		{name: "Relative URL dot notation", href: "./page.html", expected: false},
		{name: "Relative URL parent", href: "../index.html", expected: false},
		{name: "Relative URL no prefix", href: "page.html", expected: false},
		{name: "ftp: protocol", href: "ftp://files.example.com", expected: true},
		{name: "FTP mixed case", href: "FTP://files.example.com", expected: true},
		{name: "file: protocol", href: "file:///path/to/file", expected: true},
		{name: "File mixed case", href: "FILE:///path/to/file", expected: true},
		{name: "Custom protocol", href: "custom://resource", expected: true},
		{name: "WebSocket protocol", href: "ws://example.com", expected: true},
		{name: "WebSocket Secure protocol", href: "wss://example.com", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := shouldSkipLink(tc.href)
			assert.Equal(t, tc.expected, actual, "shouldSkipLink(%q) should return %v", tc.href, tc.expected)
		})
	}
}
