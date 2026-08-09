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

package runtime

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/markdown/markdown_dto"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestGetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{name: "found", m: map[string]any{"title": "Hello"}, key: "title", want: "Hello"},
		{name: "missing key", m: map[string]any{}, key: "title", want: ""},
		{name: "wrong type", m: map[string]any{"title": 42}, key: "title", want: ""},
		{name: "nil map", m: nil, key: "title", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, getString(tt.m, tt.key))
		})
	}
}

func TestGetInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want int
	}{
		{name: "int value", m: map[string]any{"level": 3}, key: "level", want: 3},
		{name: "int64 value", m: map[string]any{"level": int64(5)}, key: "level", want: 5},
		{name: "int32 value", m: map[string]any{"level": int32(4)}, key: "level", want: 4},
		{name: "int16 value", m: map[string]any{"level": int16(3)}, key: "level", want: 3},
		{name: "int8 value", m: map[string]any{"level": int8(2)}, key: "level", want: 2},
		{name: "uint value", m: map[string]any{"level": uint(6)}, key: "level", want: 6},
		{name: "uint64 value", m: map[string]any{"level": uint64(7)}, key: "level", want: 7},
		{name: "uint32 value", m: map[string]any{"level": uint32(8)}, key: "level", want: 8},
		{name: "uint16 value", m: map[string]any{"level": uint16(9)}, key: "level", want: 9},
		{name: "uint8 value", m: map[string]any{"level": uint8(10)}, key: "level", want: 10},
		{name: "float64 value", m: map[string]any{"level": float64(5)}, key: "level", want: 5},
		{name: "float32 value", m: map[string]any{"level": float32(3)}, key: "level", want: 3},
		{name: "missing key", m: map[string]any{}, key: "level", want: 0},
		{name: "wrong type", m: map[string]any{"level": "three"}, key: "level", want: 0},
		{name: "nil map", m: nil, key: "level", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, getInt(tt.m, tt.key))
		})
	}
}

func TestGetSections_NilCollectionData(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Nil(t, sections)
}

func TestGetSections_NonMapCollectionData(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData("not a map").
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Nil(t, sections)
}

func TestGetSections_MissingPage(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(map[string]any{"other": "data"}).
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Nil(t, sections)
}

func TestGetSections_PageNotMap(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(map[string]any{"page": "not a map"}).
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Nil(t, sections)
}

func TestGetSections_MissingSections(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(map[string]any{
			"page": map[string]any{"Title": "Test"},
		}).
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Nil(t, sections)
}

func TestGetSections_TypedSections(t *testing.T) {
	t.Parallel()

	expected := []markdown_dto.SectionData{
		{Title: "Introduction", Slug: "introduction", Level: 2},
		{Title: "Details", Slug: "details", Level: 3},
	}

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(map[string]any{
			"page": map[string]any{
				"Sections": expected,
			},
		}).
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Equal(t, expected, sections)
}

func TestGetSections_GenericMapSections(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(map[string]any{
			"page": map[string]any{
				"Sections": []any{
					map[string]any{"Title": "Intro", "Slug": "intro", "Level": 2},
					map[string]any{"Title": "Body", "Slug": "body", "Level": float64(3)},
				},
			},
		}).
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Len(t, sections, 2)
	assert.Equal(t, "Intro", sections[0].Title)
	assert.Equal(t, "intro", sections[0].Slug)
	assert.Equal(t, 2, sections[0].Level)
	assert.Equal(t, "Body", sections[1].Title)
	assert.Equal(t, 3, sections[1].Level)
}

func TestGetSections_Int64Levels(t *testing.T) {
	t.Parallel()

	rd := templater_dto.NewRequestDataBuilder().
		WithCollectionData(map[string]any{
			"page": map[string]any{
				"Sections": []any{
					map[string]any{"Title": "Intro", "Slug": "intro", "Level": int64(2)},
					map[string]any{"Title": "Details", "Slug": "details", "Level": int64(3)},
				},
			},
		}).
		Build()
	defer rd.Release()

	sections := GetSections(rd)
	assert.Len(t, sections, 2)
	assert.Equal(t, "Intro", sections[0].Title)
	assert.Equal(t, 2, sections[0].Level)
	assert.Equal(t, "Details", sections[1].Title)
	assert.Equal(t, 3, sections[1].Level)
}

func TestNewFilter(t *testing.T) {
	t.Parallel()

	f := NewFilter("status", FilterOpEquals, "published")
	assert.Equal(t, "status", f.Field)
	assert.Equal(t, FilterOpEquals, f.Operator)
	assert.Equal(t, "published", f.Value)
}

func TestAnd(t *testing.T) {
	t.Parallel()

	f1 := NewFilter("a", FilterOpEquals, 1)
	f2 := NewFilter("b", FilterOpEquals, 2)
	group := And(f1, f2)

	assert.Len(t, group.Filters, 2)
	assert.Equal(t, "AND", group.Logic)
}

func TestOr(t *testing.T) {
	t.Parallel()

	f1 := NewFilter("a", FilterOpEquals, 1)
	f2 := NewFilter("b", FilterOpEquals, 2)
	group := Or(f1, f2)

	assert.Len(t, group.Filters, 2)
	assert.Equal(t, "OR", group.Logic)
}

func TestNewSortOption(t *testing.T) {
	t.Parallel()

	s := NewSortOption("created_at", SortDesc)
	assert.Equal(t, "created_at", s.Field)
	assert.Equal(t, SortDesc, s.Order)
}

func TestNewPaginationOptions(t *testing.T) {
	t.Parallel()

	p := NewPaginationOptions(10, 20)
	assert.Equal(t, 10, p.Limit)
	assert.Equal(t, 20, p.Offset)
}

func TestDefaultNavigationConfig(t *testing.T) {
	t.Parallel()

	navigationConfig := DefaultNavigationConfig()
	assert.True(t, navigationConfig.GroupBySection)
	assert.False(t, navigationConfig.IncludeHidden)
	assert.Equal(t, 999, navigationConfig.DefaultOrder)
}

func TestCollectionNotFound(t *testing.T) {
	t.Parallel()

	cause := errors.New("item not found in store")
	err := CollectionNotFound("blog", "/blog/my-post", cause)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `collection "blog"`)
	assert.Contains(t, err.Error(), `route "/blog/my-post"`)

	cnf, ok := errors.AsType[*collectionNotFoundError](err)
	require.True(t, ok)
	assert.Equal(t, 404, cnf.StatusCode())
	assert.Equal(t, "COLLECTION_NOT_FOUND", cnf.ErrorCode())
	assert.Equal(t, cause, cnf.Unwrap())
}

func TestCollectionNotFound_ImplementsActionError(t *testing.T) {
	t.Parallel()

	err := CollectionNotFound("products", "/products/xyz", nil)

	actionErr, ok := errors.AsType[daemon_dto.ActionError](err)
	assert.True(t, ok)
	assert.Equal(t, 404, actionErr.StatusCode())
	assert.Equal(t, "COLLECTION_NOT_FOUND", actionErr.ErrorCode())
}

func TestCollectionNotFound_NilCause(t *testing.T) {
	t.Parallel()

	err := CollectionNotFound("pages", "/pages/missing", nil)

	cnf, ok := errors.AsType[*collectionNotFoundError](err)
	require.True(t, ok)
	assert.Nil(t, cnf.Unwrap())
}

func TestCollectionNotFound_SafeMessage(t *testing.T) {
	t.Parallel()

	t.Run("reports the route and withholds the collection", func(t *testing.T) {
		t.Parallel()

		cnf, ok := errors.AsType[*collectionNotFoundError](
			CollectionNotFound("blog", "/blog/my-post", errors.New("item not found in store")))
		require.True(t, ok)

		safe := cnf.SafeMessage()

		assert.Contains(t, safe, `route "/blog/my-post"`)
		assert.NotContains(t, safe, "blog\"", "the collection names internal project structure")
		assert.NotContains(t, safe, "item not found in store", "the cause is for logs, not users")
	})

	t.Run("satisfies the user-facing error contract", func(t *testing.T) {
		t.Parallel()

		err := CollectionNotFound("blog", "/blog/my-post", nil)

		safeErr, ok := errors.AsType[safeerror.Error](err)
		require.True(t, ok, "the rendering pipeline selects the safe message through this interface")
		assert.Equal(t, safeErr.SafeMessage(),
			safeerror.ExtractSafeMessage(err, false))
		assert.Equal(t, err.Error(), safeerror.ExtractSafeMessage(err, true),
			"development mode keeps the full detail")
	})

	t.Run("caps an over-long route", func(t *testing.T) {
		t.Parallel()

		route := "/blog/" + strings.Repeat("é", 4096)
		cnf, ok := errors.AsType[*collectionNotFoundError](CollectionNotFound("blog", route, nil))
		require.True(t, ok)

		safe := cnf.SafeMessage()

		assert.Less(t, len(safe), len(route),
			"a path parameter is caller-chosen, so it must not inflate every 404 it produces")
		assert.Contains(t, safe, routeTruncationMarker, "a truncated route is marked as partial")
	})

	t.Run("escapes control bytes in the route", func(t *testing.T) {
		t.Parallel()

		cnf, ok := errors.AsType[*collectionNotFoundError](
			CollectionNotFound("blog", "/blog/a\nb\tc", nil))
		require.True(t, ok)

		safe := cnf.SafeMessage()

		assert.NotContains(t, safe, "\n", "a raw newline would let a route forge a log line")
		assert.NotContains(t, safe, "\t")
		assert.Contains(t, safe, `\n`)
	})
}

func TestTruncateRoute(t *testing.T) {
	t.Parallel()

	t.Run("leaves a route within the cap untouched", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "/blog/my-post", truncateRoute("/blog/my-post"))
		assert.Equal(t, "", truncateRoute(""))

		exact := strings.Repeat("a", maxSafeMessageRouteBytes)
		assert.Equal(t, exact, truncateRoute(exact), "a route exactly at the cap still fits")
	})

	t.Run("marks and bounds a route past the cap", func(t *testing.T) {
		t.Parallel()

		result := truncateRoute(strings.Repeat("a", maxSafeMessageRouteBytes+1))

		assert.Len(t, result, maxSafeMessageRouteBytes,
			"the marker is counted inside the cap, so the result never exceeds it")
		assert.True(t, strings.HasSuffix(result, routeTruncationMarker))
	})

	t.Run("never severs a multi-byte rune", func(t *testing.T) {
		t.Parallel()

		for extra := range 8 {
			route := "/b/" + strings.Repeat("é", maxSafeMessageRouteBytes/2+extra)
			result := truncateRoute(route)

			assert.True(t, utf8.ValidString(result),
				"route of %d bytes produced invalid UTF-8: %q", len(route), result)
			assert.LessOrEqual(t, len(result), maxSafeMessageRouteBytes)
		}
	})

	t.Run("never severs a four-byte rune", func(t *testing.T) {
		t.Parallel()

		for extra := range 8 {
			route := "/b/" + strings.Repeat("😀", maxSafeMessageRouteBytes/4+extra)
			result := truncateRoute(route)

			assert.True(t, utf8.ValidString(result),
				"route of %d bytes produced invalid UTF-8: %q", len(route), result)
			assert.LessOrEqual(t, len(result), maxSafeMessageRouteBytes)
		}
	})
}
