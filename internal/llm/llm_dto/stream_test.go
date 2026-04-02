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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChunkEvent(t *testing.T) {
	t.Parallel()

	chunk := &StreamChunk{
		ID:    "cmpl-1",
		Model: "gpt-4o",
		Delta: &MessageDelta{Content: new("hello")},
	}

	event := NewChunkEvent(chunk)

	assert.Equal(t, StreamEventChunk, event.Type)
	assert.Equal(t, chunk, event.Chunk)
	assert.False(t, event.Done)
	assert.Nil(t, event.Error)
	assert.Nil(t, event.FinalResponse)
}

func TestNewDoneEvent(t *testing.T) {
	t.Parallel()

	t.Run("with final response", func(t *testing.T) {
		t.Parallel()

		response := &CompletionResponse{ID: "cmpl-1"}
		event := NewDoneEvent(response)

		assert.Equal(t, StreamEventDone, event.Type)
		assert.True(t, event.Done)
		assert.Equal(t, response, event.FinalResponse)
		assert.Nil(t, event.Chunk)
		assert.Nil(t, event.Error)
	})

	t.Run("without final response", func(t *testing.T) {
		t.Parallel()

		event := NewDoneEvent(nil)

		assert.Equal(t, StreamEventDone, event.Type)
		assert.True(t, event.Done)
		assert.Nil(t, event.FinalResponse)
	})
}

func TestNewErrorEvent(t *testing.T) {
	t.Parallel()

	err := errors.New("connection lost")
	event := NewErrorEvent(err)

	assert.Equal(t, StreamEventError, event.Type)
	assert.Equal(t, err, event.Error)
	assert.False(t, event.Done)
	assert.Nil(t, event.Chunk)
	assert.Nil(t, event.FinalResponse)
}
