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

package route_pattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTrailing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want TrailingSegment
	}{
		{
			name: "empty pattern",
			in:   "",
			want: TrailingSegment{},
		},
		{
			name: "static pattern",
			in:   "/about",
			want: TrailingSegment{},
		},
		{
			name: "root pattern",
			in:   "/",
			want: TrailingSegment{},
		},
		{
			name: "bare named param",
			in:   "/blog/{slug}",
			want: TrailingSegment{Prefix: "/blog/", Name: "slug", Found: true},
		},
		{
			name: "regex catch-all",
			in:   "/docs/{slug:.+}",
			want: TrailingSegment{Prefix: "/docs/", Name: "slug", Regex: ".+", HasRegex: true, Found: true},
		},
		{
			name: "regex with star",
			in:   "/docs/{slug:.*}",
			want: TrailingSegment{Prefix: "/docs/", Name: "slug", Regex: ".*", HasRegex: true, Found: true},
		},
		{
			name: "non-greedy regex",
			in:   "/docs/{slug:.+?}",
			want: TrailingSegment{Prefix: "/docs/", Name: "slug", Regex: ".+?", HasRegex: true, Found: true},
		},
		{
			name: "character-class regex",
			in:   "/files/{path:[a-zA-Z0-9/_-]+}",
			want: TrailingSegment{Prefix: "/files/", Name: "path", Regex: "[a-zA-Z0-9/_-]+", HasRegex: true, Found: true},
		},
		{
			name: "regex with nested braces is parsed correctly",
			in:   "/docs/{slug:a{2,4}}",
			want: TrailingSegment{Prefix: "/docs/", Name: "slug", Regex: "a{2,4}", HasRegex: true, Found: true},
		},
		{
			name: "regex with deeply nested braces is parsed correctly",
			in:   "/x/{name:[{}]+}",
			want: TrailingSegment{Prefix: "/x/", Name: "name", Regex: "[{}]+", HasRegex: true, Found: true},
		},
		{
			name: "empty regex still classifies as HasRegex",
			in:   "/x/{name:}",
			want: TrailingSegment{Prefix: "/x/", Name: "name", Regex: "", HasRegex: true, Found: true},
		},
		{
			name: "regex without name rejected",
			in:   "/blog/{:.+}",
			want: TrailingSegment{},
		},
		{
			name: "empty braces rejected",
			in:   "/blog/{}",
			want: TrailingSegment{},
		},
		{
			name: "trailing brace without opener rejected",
			in:   "stray}",
			want: TrailingSegment{},
		},
		{
			name: "non-trailing param rejected",
			in:   "/docs/{slug}/index",
			want: TrailingSegment{},
		},
		{
			name: "unbalanced extra closing brace rejected",
			in:   "/x/{name}}",
			want: TrailingSegment{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseTrailing(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
