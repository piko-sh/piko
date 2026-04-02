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

package mem_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/mem"
)

func TestString(t *testing.T) {
	testCases := []struct {
		name  string
		want  string
		input []byte
	}{
		{name: "nil slice returns empty string", input: nil, want: ""},
		{name: "empty slice returns empty string", input: []byte{}, want: ""},
		{name: "single byte", input: []byte("a"), want: "a"},
		{name: "ascii string", input: []byte("hello world"), want: "hello world"},
		{name: "unicode content", input: []byte("caf\xc3\xa9"), want: "caf\u00e9"},
		{name: "binary content", input: []byte{0x00, 0xff, 0x01}, want: "\x00\xff\x01"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mem.String(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBytes(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string returns zero length", input: "", want: ""},
		{name: "single character", input: "a", want: "a"},
		{name: "ascii string", input: "hello world", want: "hello world"},
		{name: "unicode content", input: "caf\u00e9", want: "caf\u00e9"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mem.Bytes(tc.input)
			if tc.input == "" {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tc.want, string(got))
			}
		})
	}
}
