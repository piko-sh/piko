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

func TestEmbeddingResponse_FirstEmbedding(t *testing.T) {
	t.Parallel()

	t.Run("with embeddings", func(t *testing.T) {
		t.Parallel()

		response := &EmbeddingResponse{
			Embeddings: []Embedding{
				{Index: 0, Vector: []float32{0.1, 0.2, 0.3}},
				{Index: 1, Vector: []float32{0.4, 0.5, 0.6}},
			},
		}
		first := response.FirstEmbedding()
		assert.Equal(t, 0, first.Index)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, first.Vector)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		response := &EmbeddingResponse{}
		first := response.FirstEmbedding()
		assert.Equal(t, Embedding{}, first)
	})
}

func TestEmbeddingResponse_FirstVector(t *testing.T) {
	t.Parallel()

	t.Run("with embeddings", func(t *testing.T) {
		t.Parallel()

		response := &EmbeddingResponse{
			Embeddings: []Embedding{
				{Vector: []float32{1.0, 2.0}},
			},
		}
		assert.Equal(t, []float32{1.0, 2.0}, response.FirstVector())
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		response := &EmbeddingResponse{}
		assert.Nil(t, response.FirstVector())
	})
}

func TestEncodingFormatHelpers(t *testing.T) {
	t.Parallel()

	t.Run("float", func(t *testing.T) {
		t.Parallel()

		f := EncodingFormatFloat()
		assert.NotNil(t, f)
		assert.Equal(t, "float", *f)
	})

	t.Run("base64", func(t *testing.T) {
		t.Parallel()

		b := EncodingFormatBase64()
		assert.NotNil(t, b)
		assert.Equal(t, "base64", *b)
	})
}
