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

package generator_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveModulePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		path       string
		moduleName string
		want       string
	}{
		{
			name:       "non-alias path returned unchanged",
			path:       "partials/card.pk",
			moduleName: "mymodule",
			want:       "partials/card.pk",
		},
		{
			name:       "alias path with module resolved",
			path:       "@/partials/card.pk",
			moduleName: "test_resolve_1",
			want:       "test_resolve_1/partials/card.pk",
		},
		{
			name:       "alias path with empty module returned unchanged",
			path:       "@/partials/card.pk",
			moduleName: "",
			want:       "@/partials/card.pk",
		},
		{
			name:       "just the prefix",
			path:       "@/",
			moduleName: "test_resolve_2",
			want:       "test_resolve_2/",
		},
		{
			name:       "path without @ prefix",
			path:       "regular/path",
			moduleName: "test_resolve_3",
			want:       "regular/path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ResolveModulePath(tc.path, tc.moduleName)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("cache hit returns same result", func(t *testing.T) {
		t.Parallel()

		first := ResolveModulePath("@/components/button.pk", "test_cache_module")
		second := ResolveModulePath("@/components/button.pk", "test_cache_module")
		assert.Equal(t, first, second)
		assert.Equal(t, "test_cache_module/components/button.pk", first)
	})

	t.Run("different paths same module", func(t *testing.T) {
		t.Parallel()

		a := ResolveModulePath("@/a.pk", "test_multi_path")
		b := ResolveModulePath("@/b.pk", "test_multi_path")
		assert.Equal(t, "test_multi_path/a.pk", a)
		assert.Equal(t, "test_multi_path/b.pk", b)
	})
}
