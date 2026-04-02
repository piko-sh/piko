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

package htmllexer

import "testing"

func TestTokenTypeString(t *testing.T) {
	testCases := []struct {
		name     string
		token    TokenType
		expected string
	}{
		{name: "error token", token: ErrorToken, expected: "Error"},
		{name: "text token", token: TextToken, expected: "Text"},
		{name: "start tag token", token: StartTagToken, expected: "StartTag"},
		{name: "end tag token", token: EndTagToken, expected: "EndTag"},
		{name: "comment token", token: CommentToken, expected: "Comment"},
		{name: "svg token", token: SVGToken, expected: "SVG"},
		{name: "math token", token: MathToken, expected: "Math"},
		{name: "attribute token", token: AttributeToken, expected: "Attribute"},
		{name: "start tag close token", token: StartTagCloseToken, expected: "StartTagClose"},
		{name: "start tag void token", token: StartTagVoidToken, expected: "StartTagVoid"},
		{name: "unknown token", token: TokenType(255), expected: "Unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.token.String()
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}
