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

package browser

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestIsUnresponsivePageError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "context deadline exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "page unresponsive",
			err:  errors.New("page unresponsive"),
			want: true,
		},
		{
			name: "CDP stuck",
			err:  errors.New("CDP stuck"),
			want: true,
		},
		{
			name: "page not responsive",
			err:  errors.New("page not responsive"),
			want: true,
		},
		{
			name: "pattern embedded in longer message",
			err:  errors.New("action failed: context deadline exceeded after 5s"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("element not found: #my-button"),
			want: false,
		},
		{
			name: "empty error message",
			err:  errors.New(""),
			want: false,
		},
		{
			name: "similar but non-matching",
			err:  errors.New("context was cancelled"),
			want: false,
		},
		{
			name: "case-sensitive mismatch",
			err:  errors.New("Context Deadline Exceeded"),
			want: false,
		},
		{
			name: "wrapped error with matching pattern",
			err:  errors.New("navigate failed: page unresponsive: timeout"),
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isUnresponsivePageError(tc.err)
			if got != tc.want {
				t.Errorf("isUnresponsivePageError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		maxRunes int
		want     string
	}{
		{name: "ascii fits", input: "hello", maxRunes: 10, want: "hello"},
		{name: "ascii truncates", input: "hello world", maxRunes: 5, want: "hello..."},
		{name: "cjk truncates by runes", input: "你好世界你好世界", maxRunes: 4, want: "你好世界..."},
		{name: "accented latin truncates by runes", input: "élève élève", maxRunes: 5, want: "élève..."},
		{name: "exact length no suffix", input: "abcde", maxRunes: 5, want: "abcde"},
		{name: "zero produces empty", input: "hello", maxRunes: 0, want: ""},
		{name: "negative produces empty", input: "hello", maxRunes: -1, want: ""},
		{name: "empty input", input: "", maxRunes: 5, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateRunes(tc.input, tc.maxRunes)
			if got != tc.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tc.input, tc.maxRunes, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("truncateRunes(%q, %d) returned invalid UTF-8: %q", tc.input, tc.maxRunes, got)
			}
			if body, ok := strings.CutSuffix(got, fmtTruncatedText); ok {
				if utf8.RuneCountInString(body) > tc.maxRunes {
					t.Errorf("truncateRunes(%q, %d) body has %d runes (exceeds limit)", tc.input, tc.maxRunes, utf8.RuneCountInString(body))
				}
			}
		})
	}
}
