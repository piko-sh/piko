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

package cache_provider_redis_cluster

import (
	"net"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
)

func TestBuildIndexSchema(t *testing.T) {
	commandPrefix := []any{"FT.CREATE", "idx:users", "ON", "JSON", "PREFIX", "1", "users:", "SCHEMA"}

	testCases := []struct {
		name                  string
		fields                []cache.FieldSchema
		expectedFieldsCommand []any
		expectedSkipped       []string
		expectedFieldCount    int
	}{
		{
			name:                  "no fields",
			fields:                nil,
			expectedFieldsCommand: nil,
			expectedSkipped:       nil,
			expectedFieldCount:    0,
		},
		{
			name: "every indexable type",
			fields: []cache.FieldSchema{
				cache.TextField("title"),
				cache.TagField("status"),
				cache.NumericField("age"),
				cache.GeoField("location"),
			},
			expectedFieldsCommand: []any{
				"$.title", "AS", "title", "TEXT",
				"$.status", "AS", "status", "TAG",
				"$.age", "AS", "age", "NUMERIC",
				"$.location", "AS", "location", "GEO",
			},
			expectedFieldCount: 4,
		},
		{
			name: "text weight and sortable options",
			fields: []cache.FieldSchema{
				{Name: "headline", Type: cache.FieldTypeText, Weight: 2.5, Sortable: true},
				cache.SortableNumericField("price"),
			},
			expectedFieldsCommand: []any{
				"$.headline", "AS", "headline", "TEXT", "WEIGHT", 2.5, "SORTABLE",
				"$.price", "AS", "price", "NUMERIC", "SORTABLE",
			},
			expectedFieldCount: 2,
		},
		{
			name: "vector fields are skipped and not counted",
			fields: []cache.FieldSchema{
				cache.TagField("category"),
				cache.VectorField("embedding", 384),
			},
			expectedFieldsCommand: []any{"$.category", "AS", "category", "TAG"},
			expectedSkipped:       []string{"embedding"},
			expectedFieldCount:    1,
		},
		{
			name: "only vector fields leaves nothing to index",
			fields: []cache.FieldSchema{
				cache.VectorField("embedding", 384),
				cache.VectorField("image_embedding", 512),
			},
			expectedFieldsCommand: nil,
			expectedSkipped:       []string{"embedding", "image_embedding"},
			expectedFieldCount:    0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			schema := buildIndexSchema("idx:users", "users:", testCase.fields)

			expectedArguments := append(append([]any{}, commandPrefix...), testCase.expectedFieldsCommand...)
			assert.Equal(t, expectedArguments, schema.arguments)
			assert.Equal(t, testCase.expectedSkipped, schema.skippedFields)
			assert.Equal(t, testCase.expectedFieldCount, schema.fieldCount)
		})
	}
}

func TestCreateIndex_RejectsSchemaWithoutIndexableFields(t *testing.T) {
	adapter := &RedisClusterAdapter[string, string]{}
	adapter.indexName = "idx:vectors"
	adapter.namespace = "vectors:"
	adapter.schema = cache.NewSearchSchema(cache.VectorField("embedding", 384))

	err := adapter.createIndex(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no indexable fields for RediSearch index idx:vectors")
}

func TestCreateIndex_WrapsCommandFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:       []string{address},
		DialTimeout: 100 * time.Millisecond,
		MaxRetries:  -1,
	})
	t.Cleanup(func() { _ = client.Close() })

	adapter := &RedisClusterAdapter[string, string]{}
	adapter.client = client
	adapter.indexName = "idx:mixed"
	adapter.namespace = "mixed:"
	adapter.schema = cache.NewSearchSchema(cache.TagField("category"), cache.VectorField("embedding", 384))

	err = adapter.createIndex(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create RediSearch index idx:mixed")
}
