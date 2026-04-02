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

package layouter_domain

import "testing"

func TestApplyTextTransform(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		transform TextTransformType
		expected  string
	}{
		{
			name:      "none leaves text unchanged",
			text:      "Hello World",
			transform: TextTransformNone,
			expected:  "Hello World",
		},
		{
			name:      "uppercase converts all to upper",
			text:      "Hello World",
			transform: TextTransformUppercase,
			expected:  "HELLO WORLD",
		},
		{
			name:      "lowercase converts all to lower",
			text:      "Hello World",
			transform: TextTransformLowercase,
			expected:  "hello world",
		},
		{
			name:      "capitalise title-cases each word",
			text:      "hello world",
			transform: TextTransformCapitalise,
			expected:  "Hello World",
		},
		{
			name:      "uppercase with accented characters",
			text:      "cafe resume",
			transform: TextTransformUppercase,
			expected:  "CAFE RESUME",
		},
		{
			name:      "empty string returns empty",
			text:      "",
			transform: TextTransformUppercase,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyTextTransform(tt.text, tt.transform)
			if result != tt.expected {
				t.Errorf("applyTextTransform(%q, %d) = %q, want %q",
					tt.text, tt.transform, result, tt.expected)
			}
		})
	}
}
