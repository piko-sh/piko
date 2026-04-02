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

package llm_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVectorSearchResponse_FirstResult(t *testing.T) {
	t.Parallel()

	t.Run("with results", func(t *testing.T) {
		t.Parallel()

		response := &VectorSearchResponse{
			Results: []VectorSearchResult{
				{ID: "doc1", Score: 0.95, Content: "hello"},
				{ID: "doc2", Score: 0.80, Content: "world"},
			},
		}
		first := response.FirstResult()
		assert.NotNil(t, first)
		assert.Equal(t, "doc1", first.ID)
		assert.InDelta(t, float32(0.95), first.Score, 0.001)
		assert.Equal(t, "hello", first.Content)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		response := &VectorSearchResponse{}
		assert.Nil(t, response.FirstResult())
	})
}

func TestVectorSearchResponse_HasResults(t *testing.T) {
	t.Parallel()

	t.Run("with results", func(t *testing.T) {
		t.Parallel()

		response := &VectorSearchResponse{
			Results: []VectorSearchResult{{ID: "doc1"}},
		}
		assert.True(t, response.HasResults())
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		response := &VectorSearchResponse{}
		assert.False(t, response.HasResults())
	})
}
