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

package textlimit

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		text  string
		want  string
		limit int
	}{
		{name: "within limit", text: "short", limit: 5, want: "short"},
		{name: "ascii cut", text: "abcdef", limit: 3, want: "abc" + Marker},
		{name: "multibyte boundary", text: "aéé", limit: 2, want: "a" + Marker},
		{name: "astral rune", text: "\U0001F600x", limit: 3, want: Marker},
		{name: "negative limit", text: "abc", limit: -1, want: Marker},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := Truncate(testCase.text, testCase.limit)
			assert.Equal(t, testCase.want, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
	long := strings.Repeat("é", 400)
	assert.LessOrEqual(t, len(Truncate(long, 320)), 320+len(Marker))
}

func TestCompletePrefix(t *testing.T) {
	for _, testCase := range []struct {
		name string
		data string
		want string
	}{
		{name: "empty", data: "", want: ""},
		{name: "ascii", data: "abc", want: "abc"},
		{name: "complete rune", data: "aé", want: "aé"},
		{name: "cut rune", data: "a\xc3", want: "a"},
		{name: "cut astral rune", data: "a\xf0\x9f\x98", want: "a"},
		{name: "invalid trailing bytes", data: "a\x80\x80\x80\x80\x80", want: "a\x80\x80\x80\x80\x80"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, string(CompletePrefix([]byte(testCase.data))))
		})
	}
}

func TestCompleteSuffix(t *testing.T) {
	for _, testCase := range []struct {
		name string
		data string
		want string
	}{
		{name: "empty", data: "", want: ""},
		{name: "ascii", data: "abc", want: "abc"},
		{name: "cut rune", data: "\xa9b", want: "b"},
		{name: "cut astral rune", data: "\x9f\x98\x80b", want: "b"},
		{name: "only continuation bytes", data: "\x80\x80\x80\x80\x80", want: "\x80\x80\x80\x80\x80"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, string(CompleteSuffix([]byte(testCase.data))))
		})
	}
}
