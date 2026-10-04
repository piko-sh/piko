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

func TestNewFullCapabilityModelInfo(t *testing.T) {
	t.Parallel()

	info := NewFullCapabilityModelInfo("claude-opus-4-6", "Claude Opus 4.6", "anthropic", 200000, 32000)

	assert.Equal(t, "claude-opus-4-6", info.ID)
	assert.Equal(t, "Claude Opus 4.6", info.Name)
	assert.Equal(t, "anthropic", info.Provider)
	assert.Equal(t, 200000, info.ContextWindow)
	assert.Equal(t, 32000, info.MaxOutputTokens)
	assert.Zero(t, info.Created)
	assert.True(t, info.SupportsStreaming)
	assert.True(t, info.SupportsTools)
	assert.True(t, info.SupportsStructuredOutput)
	assert.True(t, info.SupportsVision)
}

func TestNewEmbeddingModelInfo(t *testing.T) {
	t.Parallel()

	info := NewEmbeddingModelInfo("voyage-3.5", "voyage", 1700000000)

	assert.Equal(t, "voyage-3.5", info.ID)
	assert.Equal(t, "voyage-3.5", info.Name)
	assert.Equal(t, "voyage", info.Provider)
	assert.Equal(t, int64(1700000000), info.Created)
	assert.Zero(t, info.ContextWindow)
	assert.Zero(t, info.MaxOutputTokens)
	assert.False(t, info.SupportsStreaming)
	assert.False(t, info.SupportsTools)
	assert.False(t, info.SupportsStructuredOutput)
	assert.False(t, info.SupportsVision)
}
