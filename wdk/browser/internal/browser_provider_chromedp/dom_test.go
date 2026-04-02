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

package browser_provider_chromedp

import (
	"testing"
)

func TestNormaliseDOM(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		opts     NormaliseOptions
	}{
		{
			name:  "replaces UUID v4 with placeholder",
			input: `<div id="550e8400-e29b-41d4-a716-446655440000">content</div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   false,
			},
			expected: `<div id="[UUID]">content</div>`,
		},
		{
			name:  "replaces multiple UUIDs",
			input: `<div data-a="550e8400-e29b-41d4-a716-446655440000" data-b="6ba7b810-9dad-11d1-80b4-00c04fd430c8">x</div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   false,
			},
			expected: `<div data-a="[UUID]" data-b="[UUID]">x</div>`,
		},
		{
			name:  "preserves UUIDs when disabled",
			input: `<div id="550e8400-e29b-41d4-a716-446655440000">content</div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: false,
				FormatHTML:   false,
			},
			expected: `<div id="550e8400-e29b-41d4-a716-446655440000">content</div>`,
		},
		{
			name:  "trims whitespace",
			input: `   <div>content</div>   `,
			opts: NormaliseOptions{
				ReplaceUUIDs: false,
				FormatHTML:   false,
			},
			expected: `<div>content</div>`,
		},
		{
			name:  "handles empty string",
			input: "",
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   false,
			},
			expected: "",
		},
		{
			name:  "handles string with no UUIDs",
			input: `<p>Hello world</p>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   false,
			},
			expected: `<p>Hello world</p>`,
		},
		{
			name:  "UUID in text content",
			input: `<span>ID: 123e4567-e89b-12d3-a456-426614174000</span>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   false,
			},
			expected: `<span>ID: [UUID]</span>`,
		},
		{
			name:  "case insensitive UUID matching",
			input: `<div id="550E8400-E29B-41D4-A716-446655440000">content</div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   false,
			},
			expected: `<div id="[UUID]">content</div>`,
		},
		{
			name:  "formats HTML when enabled with block elements",
			input: `<div><div>nested</div></div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: false,
				FormatHTML:   true,
			},
			expected: `<div>
  <div>nested</div>
</div>`,
		},
		{
			name:  "formats and replaces UUIDs together",
			input: `<div id="550e8400-e29b-41d4-a716-446655440000"><div>nested</div></div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: true,
				FormatHTML:   true,
			},
			expected: `<div id="[UUID]">
  <div>nested</div>
</div>`,
		},
		{
			name:  "inline elements stay on same line",
			input: `<div><span>text</span></div>`,
			opts: NormaliseOptions{
				ReplaceUUIDs: false,
				FormatHTML:   true,
			},
			expected: `<div><span>text</span></div>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormaliseDOM(tc.input, tc.opts)
			if result != tc.expected {
				t.Errorf("NormaliseDOM() mismatch:\n  input:    %q\n  expected: %q\n  got:      %q",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestDefaultNormaliseOptions(t *testing.T) {
	opts := DefaultNormaliseOptions()

	if !opts.ReplaceUUIDs {
		t.Error("DefaultNormaliseOptions().ReplaceUUIDs should be true")
	}
	if !opts.FormatHTML {
		t.Error("DefaultNormaliseOptions().FormatHTML should be true")
	}
}
