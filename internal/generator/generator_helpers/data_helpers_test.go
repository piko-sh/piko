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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/templater/templater_dto"
)

type testPageData struct {
	Title  string `json:"Title"`
	Author string `json:"Author"`
}

func TestGetData(t *testing.T) {
	t.Parallel()

	t.Run("nil CollectionData returns zero", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithCollectionData(nil).
			Build()
		defer r.Release()
		got := GetData[testPageData](r)
		assert.Equal(t, testPageData{}, got)
	})

	t.Run("non-map CollectionData returns zero", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithCollectionData("not a map").
			Build()
		defer r.Release()
		got := GetData[testPageData](r)
		assert.Equal(t, testPageData{}, got)
	})

	t.Run("map missing page key returns zero", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithCollectionData(map[string]any{"other": "val"}).
			Build()
		defer r.Release()
		got := GetData[testPageData](r)
		assert.Equal(t, testPageData{}, got)
	})

	t.Run("direct type assertion succeeds", func(t *testing.T) {
		t.Parallel()

		expected := testPageData{Title: "Hello", Author: "World"}
		r := templater_dto.NewRequestDataBuilder().
			WithCollectionData(map[string]any{"page": expected}).
			Build()
		defer r.Release()
		got := GetData[testPageData](r)
		assert.Equal(t, expected, got)
	})

	t.Run("JSON fallback path succeeds", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithCollectionData(map[string]any{
				"page": map[string]any{
					"Title":  "Fallback Title",
					"Author": "Fallback Author",
				},
			}).
			Build()
		defer r.Release()
		got := GetData[testPageData](r)
		assert.Equal(t, "Fallback Title", got.Title)
		assert.Equal(t, "Fallback Author", got.Author)
	})

	t.Run("non-map page value with wrong type returns zero", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithCollectionData(map[string]any{"page": 42}).
			Build()
		defer r.Release()
		got := GetData[testPageData](r)
		assert.Equal(t, testPageData{}, got)
	})
}

func TestGetDataNilRequestAndInterfaceTargets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		build func() *templater_dto.RequestData
		read  func(r *templater_dto.RequestData) any
		want  any
		name  string
	}{
		{
			name:  "nil request with struct target returns zero",
			build: func() *templater_dto.RequestData { return nil },
			read:  func(r *templater_dto.RequestData) any { return GetData[testPageData](r) },
			want:  testPageData{},
		},
		{
			name:  "nil request with interface target returns nil",
			build: func() *templater_dto.RequestData { return nil },
			read:  func(r *templater_dto.RequestData) any { return GetData[fmt.Stringer](r) },
			want:  nil,
		},
		{
			name:  "nil request with empty interface target returns nil",
			build: func() *templater_dto.RequestData { return nil },
			read:  func(r *templater_dto.RequestData) any { return GetData[any](r) },
			want:  nil,
		},
		{
			name: "missing page with interface target returns nil",
			build: func() *templater_dto.RequestData {
				return templater_dto.NewRequestDataBuilder().WithCollectionData(map[string]any{}).Build()
			},
			read: func(r *templater_dto.RequestData) any { return GetData[fmt.Stringer](r) },
			want: nil,
		},
		{
			name: "nil page value with struct target returns zero",
			build: func() *templater_dto.RequestData {
				return templater_dto.NewRequestDataBuilder().WithCollectionData(map[string]any{"page": nil}).Build()
			},
			read: func(r *templater_dto.RequestData) any { return GetData[testPageData](r) },
			want: testPageData{},
		},
		{
			name: "page map with empty interface target decodes the map",
			build: func() *templater_dto.RequestData {
				return templater_dto.NewRequestDataBuilder().
					WithCollectionData(map[string]any{"page": map[string]any{"Title": "Hello"}}).
					Build()
			},
			read: func(r *templater_dto.RequestData) any { return GetData[any](r) },
			want: map[string]any{"Title": "Hello"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			r := testCase.build()
			if r != nil {
				defer r.Release()
			}
			assert.NotPanics(t, func() {
				assert.Equal(t, testCase.want, testCase.read(r))
			})
		})
	}
}
