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

package cache_provider_redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
)

func TestRedisAdapter_SearchWithoutSchemaIsNotSupported(t *testing.T) {
	t.Parallel()

	adapter := &RedisAdapter[string, string]{}

	_, err := adapter.Search(t.Context(), "query", nil)
	require.ErrorIs(t, err, cache.ErrSearchNotSupported)
	_, err = adapter.Query(t.Context(), nil)
	require.ErrorIs(t, err, cache.ErrSearchNotSupported)
	assert.False(t, adapter.SupportsSearch())
	assert.Nil(t, adapter.GetSchema())
}

func TestRedisAdapter_SearchSchemaIsReported(t *testing.T) {
	t.Parallel()

	schema := vectorSchema()
	adapter := &RedisAdapter[string, string]{schema: schema}

	assert.True(t, adapter.SupportsSearch())
	assert.Same(t, schema, adapter.GetSchema())
}
