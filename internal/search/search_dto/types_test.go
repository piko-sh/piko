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

package search_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSearchConfig(t *testing.T) {
	t.Parallel()

	config := DefaultSearchConfig("hello world")

	assert.Equal(t, "hello world", config.Query)
	assert.Nil(t, config.Fields)
	assert.InDelta(t, 0.3, config.FuzzyThreshold, 0.001)
	assert.Equal(t, 0, config.Limit)
	assert.Equal(t, 0, config.Offset)
	assert.InDelta(t, 0.0, config.MinScore, 0.001)
	assert.False(t, config.CaseSensitive)
	assert.True(t, config.EnableFuzzyFallback)
	assert.InDelta(t, 0.85, config.FuzzySimilarityThreshold, 0.001)
	assert.Equal(t, 3, config.FuzzyMaxResults)
}

func TestDefaultSearchConfig_EmptyQuery(t *testing.T) {
	t.Parallel()

	config := DefaultSearchConfig("")

	assert.Empty(t, config.Query)
	assert.True(t, config.EnableFuzzyFallback)
}
