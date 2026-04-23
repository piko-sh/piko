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

//go:build !(js && wasm)

package runtime

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_domain"
)

type sampleDoc struct {
	Title string
	Score int
}

func TestHydrateSearchResultsReflectPopulatesFields(t *testing.T) {
	t.Parallel()

	domainResults := []collection_domain.SearchResult{
		{
			Item:        map[string]any{"Title": "first", "Score": 10},
			Score:       0.9,
			FieldScores: map[string]float64{"Title": 0.9},
		},
		{
			Item:        map[string]any{"Title": "second", "Score": 7},
			Score:       0.5,
			FieldScores: map[string]float64{"Title": 0.5},
		},
	}

	slice := hydrateSearchResultsReflect(t.Context(), domainResults, reflect.TypeFor[sampleDoc]())

	require.Equal(t, 2, slice.Len())

	first := slice.Index(0)
	require.Equal(t, "first", first.FieldByName("Item").FieldByName("Title").String())
	require.InDelta(t, 0.9, first.FieldByName("Score").Float(), 1e-9)
}

func TestHydrateSearchResultsReflectCapsResults(t *testing.T) {
	t.Parallel()

	excess := maxHydratedSearchResults + 50
	domainResults := make([]collection_domain.SearchResult, excess)
	for index := range domainResults {
		domainResults[index] = collection_domain.SearchResult{
			Item:  map[string]any{"Title": "x"},
			Score: 0.1,
		}
	}

	slice := hydrateSearchResultsReflect(t.Context(), domainResults, reflect.TypeFor[sampleDoc]())
	require.Equal(t, maxHydratedSearchResults, slice.Len())
}

func TestMakeEmptySearchResultSliceHasZeroLength(t *testing.T) {
	t.Parallel()

	slice := makeEmptySearchResultSlice(reflect.TypeFor[sampleDoc]())

	require.Equal(t, 0, slice.Len())
	require.Equal(t, reflect.Slice, slice.Kind())
}
