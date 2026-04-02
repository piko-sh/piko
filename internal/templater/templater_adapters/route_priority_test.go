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

package templater_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestClassifyRoutePattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pattern  string
		expected routeType
	}{
		{name: "root path", pattern: "/", expected: routeTypeStatic},
		{name: "simple static path", pattern: "/docs", expected: routeTypeStatic},
		{name: "nested static path", pattern: "/docs/api/v1", expected: routeTypeStatic},
		{name: "static with trailing slash", pattern: "/docs/", expected: routeTypeStatic},
		{name: "empty pattern", pattern: "", expected: routeTypeStatic},
		{name: "single dynamic segment", pattern: "/{slug}", expected: routeTypeDynamic},
		{name: "dynamic with prefix", pattern: "/docs/{id}", expected: routeTypeDynamic},
		{name: "dynamic in middle", pattern: "/docs/{id}/edit", expected: routeTypeDynamic},
		{name: "multiple dynamic segments", pattern: "/{category}/{id}", expected: routeTypeDynamic},
		{name: "catch-all at root", pattern: "/{path}*", expected: routeTypeCatchAll},
		{name: "catch-all with prefix", pattern: "/docs/{path}*", expected: routeTypeCatchAll},
		{name: "wildcard only", pattern: "/*", expected: routeTypeCatchAll},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := classifyRoutePattern(tc.pattern)
			assert.Equal(t, tc.expected, result, "pattern %q should be classified as %v", tc.pattern, tc.expected)
		})
	}
}

func TestCountPathSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pattern  string
		expected int
	}{
		{name: "empty string", pattern: "", expected: 0},
		{name: "root only", pattern: "/", expected: 0},
		{name: "single segment", pattern: "/docs", expected: 1},
		{name: "two segments", pattern: "/docs/api", expected: 2},
		{name: "three segments", pattern: "/docs/api/v1", expected: 3},
		{name: "with trailing slash", pattern: "/docs/", expected: 1},
		{name: "dynamic segment", pattern: "/{slug}", expected: 1},
		{name: "mixed static and dynamic", pattern: "/docs/{id}/edit", expected: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := countPathSegments(tc.pattern)
			assert.Equal(t, tc.expected, result, "pattern %q should have %d segments", tc.pattern, tc.expected)
		})
	}
}

func TestCountStaticSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pattern  string
		expected int
	}{
		{name: "empty string", pattern: "", expected: 0},
		{name: "root only", pattern: "/", expected: 0},
		{name: "single static", pattern: "/docs", expected: 1},
		{name: "two static", pattern: "/docs/api", expected: 2},
		{name: "single dynamic", pattern: "/{slug}", expected: 0},
		{name: "static before dynamic", pattern: "/docs/{id}", expected: 1},
		{name: "static around dynamic", pattern: "/docs/{id}/edit", expected: 2},
		{name: "all dynamic", pattern: "/{category}/{id}", expected: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := countStaticSegments(tc.pattern)
			assert.Equal(t, tc.expected, result, "pattern %q should have %d static segments", tc.pattern, tc.expected)
		})
	}
}

func TestSortKeysByRouteSpecificity_StaticBeforeDynamic(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/{slug}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{slug}"},
				},
			},
			"pages/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/{slug}.pk", "pages/index.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 2)
	assert.Equal(t, "pages/index.pk", store.keys[0], "static route should be first")
	assert.Equal(t, "pages/{slug}.pk", store.keys[1], "dynamic route should be second")
}

func TestSortKeysByRouteSpecificity_NestedStaticBeforeDynamic(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/docs/{id}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/{id}"},
				},
			},
			"pages/docs/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/docs/{id}.pk", "pages/docs/index.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 2)
	assert.Equal(t, "pages/docs/index.pk", store.keys[0], "static route should be first")
	assert.Equal(t, "pages/docs/{id}.pk", store.keys[1], "dynamic route should be second")
}

func TestSortKeysByRouteSpecificity_DynamicBeforeCatchAll(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/docs/{path}*.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/{path}*"},
				},
			},
			"pages/docs/{id}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/{id}"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/docs/{path}*.pk", "pages/docs/{id}.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 2)
	assert.Equal(t, "pages/docs/{id}.pk", store.keys[0], "dynamic route should be first")
	assert.Equal(t, "pages/docs/{path}*.pk", store.keys[1], "catch-all route should be second")
}

func TestSortKeysByRouteSpecificity_FullPriority(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/[...path].pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{path}*"},
				},
			},
			"pages/{slug}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{slug}"},
				},
			},
			"pages/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/"},
				},
			},
			"pages/docs/{id}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/{id}"},
				},
			},
			"pages/docs/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/"},
				},
			},
			"pages/docs/api/v1.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/api/v1"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys: []string{
			"pages/[...path].pk",
			"pages/{slug}.pk",
			"pages/index.pk",
			"pages/docs/{id}.pk",
			"pages/docs/index.pk",
			"pages/docs/api/v1.pk",
		},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 6)
	assert.Equal(t, "pages/docs/api/v1.pk", store.keys[0], "3-segment static should be first")
	assert.Equal(t, "pages/docs/index.pk", store.keys[1], "1-segment static (/docs/) should be second")
	assert.Equal(t, "pages/index.pk", store.keys[2], "0-segment static (/) should be third")
	assert.Equal(t, "pages/docs/{id}.pk", store.keys[3], "2-segment dynamic should be fourth")
	assert.Equal(t, "pages/{slug}.pk", store.keys[4], "1-segment dynamic should be fifth")
	assert.Equal(t, "pages/[...path].pk", store.keys[5], "catch-all should be last")
}

func TestSortKeysByRouteSpecificity_MoreStaticSegmentsFirst(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/{category}/{id}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{category}/{id}"},
				},
			},
			"pages/docs/{id}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/docs/{id}"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/{category}/{id}.pk", "pages/docs/{id}.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 2)
	assert.Equal(t, "pages/docs/{id}.pk", store.keys[0], "route with 1 static segment should be first")
	assert.Equal(t, "pages/{category}/{id}.pk", store.keys[1], "route with 0 static segments should be second")
}

func TestSortKeysByRouteSpecificity_AlphabeticalTiebreaker(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/zebra.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/zebra"},
				},
			},
			"pages/alpha.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/alpha"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/zebra.pk", "pages/alpha.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 2)
	assert.Equal(t, "pages/alpha.pk", store.keys[0], "alphabetically first should be first")
	assert.Equal(t, "pages/zebra.pk", store.keys[1], "alphabetically second should be second")
}

func TestSortKeysByRouteSpecificity_WithPartials(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/{slug}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{slug}"},
				},
			},
			"pages/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/"},
				},
			},
		},
		partials: map[string]*PageEntry{
			"partials/card.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"": "/_piko/partial/card"},
				},
			},
		},
		emails: map[string]*PageEntry{},
		keys:   []string{"pages/{slug}.pk", "pages/index.pk", "partials/card.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 3)
	assert.Equal(t, "partials/card.pk", store.keys[0], "partial with 3 segments should be first")
	assert.Equal(t, "pages/index.pk", store.keys[1], "index page should be second")
	assert.Equal(t, "pages/{slug}.pk", store.keys[2], "dynamic route should be last")
}

func TestSortKeysByRouteSpecificity_EmptyStore(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages:    map[string]*PageEntry{},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{},
	}

	sortKeysByRouteSpecificity(store)

	assert.Empty(t, store.keys)
}

func TestSortKeysByRouteSpecificity_SingleKey(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/index.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 1)
	assert.Equal(t, "pages/index.pk", store.keys[0])
}

func TestGetPrimaryRoutePattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		store    *ManifestStore
		key      string
		expected string
	}{
		{
			name: "page entry",
			store: &ManifestStore{
				pages: map[string]*PageEntry{
					"pages/index.pk": {
						ManifestPageEntry: generator_dto.ManifestPageEntry{
							RoutePatterns: map[string]string{"en": "/"},
						},
					},
				},
				partials: map[string]*PageEntry{},
				emails:   map[string]*PageEntry{},
			},
			key:      "pages/index.pk",
			expected: "/",
		},
		{
			name: "partial entry",
			store: &ManifestStore{
				pages: map[string]*PageEntry{},
				partials: map[string]*PageEntry{
					"partials/card.pk": {
						ManifestPageEntry: generator_dto.ManifestPageEntry{
							RoutePatterns: map[string]string{"": "/_piko/partial/card"},
						},
					},
				},
				emails: map[string]*PageEntry{},
			},
			key:      "partials/card.pk",
			expected: "/_piko/partial/card",
		},
		{
			name: "email entry (no routes)",
			store: &ManifestStore{
				pages:    map[string]*PageEntry{},
				partials: map[string]*PageEntry{},
				emails: map[string]*PageEntry{
					"emails/welcome.pk": {
						ManifestPageEntry: generator_dto.ManifestPageEntry{
							RoutePatterns: nil,
						},
					},
				},
			},
			key:      "emails/welcome.pk",
			expected: "",
		},
		{
			name: "non-existent key",
			store: &ManifestStore{
				pages:    map[string]*PageEntry{},
				partials: map[string]*PageEntry{},
				emails:   map[string]*PageEntry{},
			},
			key:      "nonexistent.pk",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getPrimaryRoutePattern(tc.store, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSortKeysByRouteSpecificity_RealWorldBlogScenario(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/[...path].pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{path}*"},
				},
			},
			"pages/[page].pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{page}"},
				},
			},
			"pages/blog/{category}/{slug}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/blog/{category}/{slug}"},
				},
			},
			"pages/blog/{slug}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/blog/{slug}"},
				},
			},
			"pages/blog/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/blog"},
				},
			},
			"pages/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys: []string{
			"pages/[...path].pk",
			"pages/[page].pk",
			"pages/blog/{category}/{slug}.pk",
			"pages/blog/{slug}.pk",
			"pages/blog/index.pk",
			"pages/index.pk",
		},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 6)
	assert.Equal(t, "pages/blog/index.pk", store.keys[0], "/blog should be first")
	assert.Equal(t, "pages/index.pk", store.keys[1], "/ should be second")
	assert.Equal(t, "pages/blog/{category}/{slug}.pk", store.keys[2], "/blog/{category}/{slug} should be third")
	assert.Equal(t, "pages/blog/{slug}.pk", store.keys[3], "/blog/{slug} should be fourth")
	assert.Equal(t, "pages/[page].pk", store.keys[4], "/{page} should be fifth")
	assert.Equal(t, "pages/[...path].pk", store.keys[5], "catch-all should be last")
}

func TestSelectCanonicalRoutePattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		patterns map[string]string
		expected string
	}{
		{
			name:     "nil map returns empty string",
			patterns: nil,
			expected: "",
		},
		{
			name:     "empty map returns empty string",
			patterns: map[string]string{},
			expected: "",
		},
		{
			name:     "single entry returns that entry",
			patterns: map[string]string{"en": "/docs"},
			expected: "/docs",
		},
		{
			name:     "prefers empty key over en",
			patterns: map[string]string{"": "/default", "en": "/english"},
			expected: "/default",
		},
		{
			name:     "prefers en over other locales",
			patterns: map[string]string{"en": "/english", "fr": "/french", "de": "/german"},
			expected: "/english",
		},
		{
			name:     "falls back to alphabetically first locale",
			patterns: map[string]string{"fr": "/french", "de": "/german", "es": "/spanish"},
			expected: "/german",
		},
		{
			name:     "handles i18n path-based routing",
			patterns: map[string]string{"en": "/docs", "fr": "/fr/docs", "de": "/de/docs"},
			expected: "/docs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := selectCanonicalRoutePattern(tc.patterns)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSortKeysByRouteSpecificity_MultiLocaleIsDeterministic(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/docs.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{
						"en": "/docs",
						"fr": "/fr/docs",
						"de": "/de/docs",
					},
				},
			},
			"pages/{slug}.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/{slug}"},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/{slug}.pk", "pages/docs.pk"},
	}

	for i := range 20 {
		store.keys = []string{"pages/{slug}.pk", "pages/docs.pk"}
		sortKeysByRouteSpecificity(store)

		require.Len(t, store.keys, 2)
		assert.Equal(t, "pages/docs.pk", store.keys[0],
			"iteration %d: static route should always be first", i)
		assert.Equal(t, "pages/{slug}.pk", store.keys[1],
			"iteration %d: dynamic route should always be second", i)
	}
}

func TestSortKeysByRouteSpecificity_MultiLocaleWithMixedTypes(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/about.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{
						"en": "/about",
						"fr": "/fr/a-propos",
						"de": "/de/ueber-uns",
					},
				},
			},
			"pages/blog/index.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{
						"en": "/blog",
						"fr": "/fr/blog",
					},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
		keys:     []string{"pages/about.pk", "pages/blog/index.pk"},
	}

	sortKeysByRouteSpecificity(store)

	require.Len(t, store.keys, 2)
	assert.Equal(t, "pages/about.pk", store.keys[0], "alphabetically first key should win tie")
	assert.Equal(t, "pages/blog/index.pk", store.keys[1])
}

func TestGetPrimaryRoutePattern_MultiLocale(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		pages: map[string]*PageEntry{
			"pages/docs.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{
						"en": "/docs",
						"fr": "/fr/docs",
						"de": "/de/docs",
					},
				},
			},
		},
		partials: map[string]*PageEntry{},
		emails:   map[string]*PageEntry{},
	}

	for i := range 10 {
		result := getPrimaryRoutePattern(store, "pages/docs.pk")
		assert.Equal(t, "/docs", result, "iteration %d: should consistently return en pattern", i)
	}
}
