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

package cache_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVectorWindowSeparatesCandidatesFromPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		topK       int
		limit      int
		offset     int
		candidates int
		page       int
	}{
		{name: "defaults", topK: 0, limit: 0, offset: 0, candidates: 10, page: 10},
		{name: "limit only", topK: 0, limit: 3, offset: 0, candidates: 3, page: 3},
		{name: "top k only", topK: 8, limit: 0, offset: 0, candidates: 8, page: 8},
		{name: "top k without a limit", topK: 8, limit: NoLimit, offset: 0, candidates: 8, page: 8},
		{name: "wide top k without a limit returns every candidate", topK: 50, limit: NoLimit, offset: 0, candidates: 50, page: 50},
		{name: "wide top k with a zero limit uses the default page", topK: 50, limit: 0, offset: 0, candidates: 50, page: 10},
		{name: "no limit without top k ranks the default window", topK: 0, limit: NoLimit, offset: 0, candidates: 10, page: 10},
		{name: "top k wider than the page", topK: 1000, limit: 2, offset: 0, candidates: 1000, page: 2},
		{name: "page wider than top k", topK: 2, limit: 1000, offset: 0, candidates: 2, page: 2},
		{name: "second page ranks enough candidates", topK: 0, limit: 10, offset: 10, candidates: 20, page: 10},
		{name: "default page after an offset", topK: 0, limit: 0, offset: 5, candidates: 15, page: 10},
		{name: "negative offset is ignored", topK: 0, limit: 4, offset: -3, candidates: 4, page: 4},
		{name: "explicit top k keeps priority over offset", topK: 6, limit: 4, offset: 10, candidates: 6, page: 4},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			search := SearchOptions{TopK: testCase.topK, Limit: testCase.limit, Offset: testCase.offset}
			candidates, page := search.VectorWindow(10)
			assert.Equal(t, testCase.candidates, candidates, "search candidates")
			assert.Equal(t, testCase.page, page, "search page")

			query := QueryOptions{TopK: testCase.topK, Limit: testCase.limit, Offset: testCase.offset}
			candidates, page = query.VectorWindow(10)
			assert.Equal(t, testCase.candidates, candidates, "query candidates")
			assert.Equal(t, testCase.page, page, "query page")
		})
	}
}
