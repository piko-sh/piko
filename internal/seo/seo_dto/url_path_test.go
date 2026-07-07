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

package seo_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapePathSegments_EncodesEachSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty path returns empty",
			path: "",
			want: "",
		},
		{
			name: "plain ASCII slug is unchanged",
			path: "first-post",
			want: "first-post",
		},
		{
			name: "space becomes percent-encoded",
			path: "a b",
			want: "a%20b",
		},
		{
			name: "non-ASCII characters are percent-encoded",
			path: "café",
			want: "caf%C3%A9",
		},
		{
			name: "slash-separated segments are preserved",
			path: "a/b c",
			want: "a/b%20c",
		},
		{
			name: "leading slash is preserved",
			path: "/docs/x y",
			want: "/docs/x%20y",
		},
		{
			name: "embedded empty segment from double slash is preserved",
			path: "a//b",
			want: "a//b",
		},
		{
			name: "already-encoded input is double-encoded because callers pass raw values",
			path: "already%20encoded",
			want: "already%2520encoded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, EscapePathSegments(tt.path))
		})
	}
}
