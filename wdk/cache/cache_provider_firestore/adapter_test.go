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

package cache_provider_firestore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func newTestAdapter(namespace string) *FirestoreAdapter[string, string] {
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))
	return &FirestoreAdapter[string, string]{
		registry:             registry,
		namespace:            namespace,
		ttl:                  time.Hour,
		enableTTLClientCheck: true,
		batchSize:            500,
	}
}

func TestFieldConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "__value", fieldValue, "fieldValue constant")
	assert.Equal(t, "__tags", fieldTags, "fieldTags constant")
	assert.Equal(t, "__ttl", fieldTTL, "fieldTTL constant")
	assert.Equal(t, "__created", fieldCreated, "fieldCreated constant")
	assert.Equal(t, "__updated", fieldUpdated, "fieldUpdated constant")
	assert.Equal(t, "__version", fieldVersion, "fieldVersion constant")
	assert.Equal(t, 30, maxArrayContainsAny, "maxArrayContainsAny constant")
}

func TestIsExpired(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("ttl-test")

	t.Run("TTL in the future is not expired", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			fieldTTL: time.Now().Add(time.Hour),
		}
		assert.False(t, adapter.isExpired(data))
	})

	t.Run("TTL in the past is expired", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			fieldTTL: time.Now().Add(-time.Hour),
		}
		assert.True(t, adapter.isExpired(data))
	})

	t.Run("No TTL field means not expired", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			fieldValue: []byte("hello"),
		}
		assert.False(t, adapter.isExpired(data))
	})

	t.Run("Zero time is treated as expired", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			fieldTTL: time.Time{},
		}

		assert.True(t, adapter.isExpired(data))
	})

	t.Run("Non-time TTL value is not expired", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			fieldTTL: "not-a-time",
		}

		assert.False(t, adapter.isExpired(data))
	})

	t.Run("Nil data map is not expired", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{}
		assert.False(t, adapter.isExpired(data))
	})

	t.Run("Client-side TTL check disabled returns false even when expired", func(t *testing.T) {
		t.Parallel()

		disabledAdapter := newTestAdapter("ttl-disabled")
		disabledAdapter.enableTTLClientCheck = false

		data := map[string]any{
			fieldTTL: time.Now().Add(-time.Hour),
		}
		assert.False(t, disabledAdapter.isExpired(data))
	})
}

func TestEncodeKey(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("encode-test")

	t.Run("Simple string key encodes without modification", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey("hello")
		require.NoError(t, err)
		assert.Equal(t, "hello", encoded)
	})

	t.Run("Key with slash is percent-encoded", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey("path/to/key")
		require.NoError(t, err)
		assert.NotContains(t, encoded, "/", "encoded key must not contain /")
		assert.Contains(t, encoded, "%2F", "slashes should be encoded as %%2F")
	})

	t.Run("Key with percent sign is encoded to avoid ambiguity", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey("100%done")
		require.NoError(t, err)
		assert.Contains(t, encoded, "%25", "percent signs should be encoded as %%25")
	})

	t.Run("Empty string key encodes successfully", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey("")
		require.NoError(t, err)
		assert.Equal(t, "", encoded)
	})

	t.Run("Key with special characters encodes safely", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey("user:123/profile?active=true")
		require.NoError(t, err)
		assert.NotContains(t, encoded, "/", "no raw slashes allowed in Firestore doc IDs")
	})
}

func TestDecodeKey(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("decode-test")

	t.Run("Simple key round-trips correctly", func(t *testing.T) {
		t.Parallel()

		original := "hello"
		encoded, err := adapter.encodeKey(original)
		require.NoError(t, err)

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("Key with slash round-trips correctly", func(t *testing.T) {
		t.Parallel()

		original := "path/to/key"
		encoded, err := adapter.encodeKey(original)
		require.NoError(t, err)

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("Key with percent sign round-trips correctly", func(t *testing.T) {
		t.Parallel()

		original := "100%done"
		encoded, err := adapter.encodeKey(original)
		require.NoError(t, err)

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("Key with percent and slash round-trips correctly", func(t *testing.T) {
		t.Parallel()

		original := "user/100%2Fencoded"
		encoded, err := adapter.encodeKey(original)
		require.NoError(t, err)

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("Empty string round-trips correctly", func(t *testing.T) {
		t.Parallel()

		original := ""
		encoded, err := adapter.encodeKey(original)
		require.NoError(t, err)

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("Complex path round-trips correctly", func(t *testing.T) {
		t.Parallel()

		original := "tenant/123/user/456/profile"
		encoded, err := adapter.encodeKey(original)
		require.NoError(t, err)
		assert.NotContains(t, encoded, "/")

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
}

func TestEncodeKeyWithIntKey(t *testing.T) {
	t.Parallel()

	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))
	adapter := &FirestoreAdapter[int, string]{
		registry:             registry,
		namespace:            "int-key-test",
		ttl:                  time.Hour,
		enableTTLClientCheck: true,
		batchSize:            500,
	}

	t.Run("Integer key encodes via Sprintf", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey(42)
		require.NoError(t, err)
		assert.Equal(t, "42", encoded)
	})

	t.Run("Integer key round-trips", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeKey(12345)
		require.NoError(t, err)

		decoded, err := adapter.decodeKey(encoded)
		require.NoError(t, err)
		assert.Equal(t, 12345, decoded)
	})
}

func TestBuildDocumentData(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("build-test")

	t.Run("Document has all required fields", func(t *testing.T) {
		t.Parallel()

		before := time.Now()
		data, err := adapter.buildDocumentData("test-value", []string{"tag1", "tag2"}, time.Hour)
		after := time.Now()

		require.NoError(t, err)
		require.NotNil(t, data)

		valBytes, ok := data[fieldValue].([]byte)
		require.True(t, ok, "__value should be []byte")
		assert.NotEmpty(t, valBytes, "__value should not be empty")

		tags, ok := data[fieldTags].([]string)
		require.True(t, ok, "__tags should be []string")
		assert.Equal(t, []string{"tag1", "tag2"}, tags)

		ttlTime, ok := data[fieldTTL].(time.Time)
		require.True(t, ok, "__ttl should be time.Time")
		assert.True(t, ttlTime.After(before.Add(time.Hour-time.Second)), "TTL should be approximately now+1h")
		assert.True(t, ttlTime.Before(after.Add(time.Hour+time.Second)), "TTL should be approximately now+1h")

		createdTime, ok := data[fieldCreated].(time.Time)
		require.True(t, ok, "__created should be time.Time")
		assert.True(t, !createdTime.Before(before) && !createdTime.After(after),
			"__created should be between before and after")

		updatedTime, ok := data[fieldUpdated].(time.Time)
		require.True(t, ok, "__updated should be time.Time")
		assert.True(t, !updatedTime.Before(before) && !updatedTime.After(after),
			"__updated should be between before and after")

		version, ok := data[fieldVersion].(int64)
		require.True(t, ok, "__version should be int64")
		assert.Equal(t, int64(1), version)
	})

	t.Run("Empty tags results in empty slice", func(t *testing.T) {
		t.Parallel()

		data, err := adapter.buildDocumentData("value", []string{}, time.Hour)
		require.NoError(t, err)

		tags, ok := data[fieldTags].([]string)
		require.True(t, ok, "__tags should be []string")
		assert.Empty(t, tags, "__tags should be empty slice")
	})

	t.Run("Nil tags results in nil in the document", func(t *testing.T) {
		t.Parallel()

		data, err := adapter.buildDocumentData("value", nil, time.Hour)
		require.NoError(t, err)

		assert.Nil(t, data[fieldTags], "__tags should be nil when no tags provided")
	})

	t.Run("Short TTL sets near-future expiry", func(t *testing.T) {
		t.Parallel()

		before := time.Now()
		data, err := adapter.buildDocumentData("value", nil, time.Second)
		require.NoError(t, err)

		ttlTime, ok := data[fieldTTL].(time.Time)
		require.True(t, ok, "__ttl should be time.Time")
		assert.True(t, ttlTime.Before(before.Add(2*time.Second)),
			"TTL with 1s duration should expire within 2s from now")
	})

	t.Run("Zero TTL sets expiry at approximately now", func(t *testing.T) {
		t.Parallel()

		before := time.Now()
		data, err := adapter.buildDocumentData("value", nil, 0)
		require.NoError(t, err)

		ttlTime, ok := data[fieldTTL].(time.Time)
		require.True(t, ok, "__ttl should be time.Time")

		assert.True(t, ttlTime.After(before.Add(-time.Second)),
			"zero TTL should be approximately now")
	})

	t.Run("Value is encoded and decodable", func(t *testing.T) {
		t.Parallel()

		data, err := adapter.buildDocumentData("hello world", nil, time.Hour)
		require.NoError(t, err)

		valBytes, ok := data[fieldValue].([]byte)
		require.True(t, ok, "__value should be []byte")
		decoded, err := adapter.decodeValue(valBytes)
		require.NoError(t, err)
		assert.Equal(t, "hello world", decoded)
	})
}

func TestBatchTags(t *testing.T) {
	t.Parallel()

	t.Run("0 tags produces 0 batches", func(t *testing.T) {
		t.Parallel()

		batches := batchTags(nil)
		assert.Nil(t, batches)
		assert.Len(t, batches, 0)
	})

	t.Run("Empty slice produces 0 batches", func(t *testing.T) {
		t.Parallel()

		batches := batchTags([]string{})
		assert.Nil(t, batches)
	})

	t.Run("10 tags produce 1 batch", func(t *testing.T) {
		t.Parallel()

		tags := makeTags(10)
		batches := batchTags(tags)

		require.Len(t, batches, 1)
		assert.Len(t, batches[0], 10)
	})

	t.Run("30 tags produce 1 batch", func(t *testing.T) {
		t.Parallel()

		tags := makeTags(30)
		batches := batchTags(tags)

		require.Len(t, batches, 1)
		assert.Len(t, batches[0], 30)
	})

	t.Run("31 tags produce 2 batches", func(t *testing.T) {
		t.Parallel()

		tags := makeTags(31)
		batches := batchTags(tags)

		require.Len(t, batches, 2)
		assert.Len(t, batches[0], 30)
		assert.Len(t, batches[1], 1)
	})

	t.Run("60 tags produce 2 batches", func(t *testing.T) {
		t.Parallel()

		tags := makeTags(60)
		batches := batchTags(tags)

		require.Len(t, batches, 2)
		assert.Len(t, batches[0], 30)
		assert.Len(t, batches[1], 30)
	})

	t.Run("61 tags produce 3 batches", func(t *testing.T) {
		t.Parallel()

		tags := makeTags(61)
		batches := batchTags(tags)

		require.Len(t, batches, 3)
		assert.Len(t, batches[0], 30)
		assert.Len(t, batches[1], 30)
		assert.Len(t, batches[2], 1)
	})

	t.Run("All original tags are preserved across batches", func(t *testing.T) {
		t.Parallel()

		tags := makeTags(75)
		batches := batchTags(tags)

		var flattened []string
		for _, batch := range batches {
			flattened = append(flattened, batch...)
		}

		assert.Equal(t, tags, flattened, "flattened batches must equal original tags")
	})

	t.Run("1 tag produces 1 batch with 1 element", func(t *testing.T) {
		t.Parallel()

		batches := batchTags([]string{"solo"})

		require.Len(t, batches, 1)
		assert.Equal(t, []string{"solo"}, batches[0])
	})
}

func TestCollectionPath(t *testing.T) {
	t.Parallel()

	t.Run("Default prefix and simple namespace", func(t *testing.T) {
		t.Parallel()

		path := collectionPath("piko_cache", "my-namespace")
		assert.Equal(t, "piko_cache/my-namespace/entries", path)
	})

	t.Run("Custom prefix", func(t *testing.T) {
		t.Parallel()

		path := collectionPath("custom_prefix", "sessions")
		assert.Equal(t, "custom_prefix/sessions/entries", path)
	})

	t.Run("Namespace with special characters", func(t *testing.T) {
		t.Parallel()

		path := collectionPath("piko_cache", "tenant:123")
		assert.Equal(t, "piko_cache/tenant:123/entries", path)
	})

	t.Run("Empty namespace", func(t *testing.T) {
		t.Parallel()

		path := collectionPath("piko_cache", "")
		assert.Equal(t, "piko_cache//entries", path)
	})

	t.Run("Empty prefix", func(t *testing.T) {
		t.Parallel()

		path := collectionPath("", "namespace")
		assert.Equal(t, "/namespace/entries", path)
	})
}

func TestApplyConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("Empty config receives all defaults", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		applyConfigDefaults(&config)

		assert.Equal(t, defaultDatabaseID, config.DatabaseID,
			"DatabaseID should default to (default)")
		assert.Equal(t, defaultCollectionPrefix, config.CollectionPrefix,
			"CollectionPrefix should default to piko_cache")
		assert.Equal(t, defaultBatchSize, config.BatchSize,
			"BatchSize should default to 500")
		assert.True(t, config.EnableTTLClientCheck,
			"EnableTTLClientCheck should default to true")
		assert.NotZero(t, config.DefaultTTL,
			"DefaultTTL should be set by ApplyProviderDefaults")
		assert.NotZero(t, config.OperationTimeout,
			"OperationTimeout should be set by ApplyProviderDefaults")
		assert.NotZero(t, config.AtomicOperationTimeout,
			"AtomicOperationTimeout should be set by ApplyProviderDefaults")
		assert.NotZero(t, config.BulkOperationTimeout,
			"BulkOperationTimeout should be set by ApplyProviderDefaults")
		assert.NotZero(t, config.FlushTimeout,
			"FlushTimeout should be set by ApplyProviderDefaults")
		assert.NotZero(t, config.SearchTimeout,
			"SearchTimeout should be set by ApplyProviderDefaults")
		assert.NotZero(t, config.MaxComputeRetries,
			"MaxComputeRetries should be set by ApplyProviderDefaults")
	})

	t.Run("Explicit values are not overridden", func(t *testing.T) {
		t.Parallel()

		config := Config{
			DatabaseID:       "my-db",
			CollectionPrefix: "my_prefix",
			BatchSize:        100,
			DefaultTTL:       5 * time.Minute,
		}
		applyConfigDefaults(&config)

		assert.Equal(t, "my-db", config.DatabaseID)
		assert.Equal(t, "my_prefix", config.CollectionPrefix)
		assert.Equal(t, 100, config.BatchSize)
		assert.Equal(t, 5*time.Minute, config.DefaultTTL)
	})

	t.Run("Negative BatchSize is replaced with default", func(t *testing.T) {
		t.Parallel()

		config := Config{BatchSize: -1}
		applyConfigDefaults(&config)

		assert.Equal(t, defaultBatchSize, config.BatchSize)
	})

	t.Run("Zero BatchSize is replaced with default", func(t *testing.T) {
		t.Parallel()

		config := Config{BatchSize: 0}
		applyConfigDefaults(&config)

		assert.Equal(t, defaultBatchSize, config.BatchSize)
	})
}

func TestNewFirestoreProvider_NilRegistryFails(t *testing.T) {
	t.Parallel()

	_, err := NewFirestoreProvider(Config{
		Registry:     nil,
		EmulatorHost: "localhost:9999",
		ProjectID:    "test-project",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "EncodingRegistry")
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "(default)", defaultDatabaseID)
	assert.Equal(t, "piko_cache", defaultCollectionPrefix)
	assert.Equal(t, 500, defaultBatchSize)
}

func TestEncodeDecodeValue(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("encode-value-test")

	t.Run("String value round-trips", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeValue("hello world")
		require.NoError(t, err)
		require.NotEmpty(t, encoded)

		decoded, err := adapter.decodeValue(encoded)
		require.NoError(t, err)
		assert.Equal(t, "hello world", decoded)
	})

	t.Run("Empty string round-trips", func(t *testing.T) {
		t.Parallel()

		encoded, err := adapter.encodeValue("")
		require.NoError(t, err)

		decoded, err := adapter.decodeValue(encoded)
		require.NoError(t, err)
		assert.Equal(t, "", decoded)
	})

	t.Run("String with special characters round-trips", func(t *testing.T) {
		t.Parallel()

		original := "hello\nworld\t\"quoted\" <html>&amp;"
		encoded, err := adapter.encodeValue(original)
		require.NoError(t, err)

		decoded, err := adapter.decodeValue(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
}

func TestStats(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("stats-test")

	stats := adapter.Stats()
	assert.Equal(t, uint64(0), stats.Hits, "initial hits")
	assert.Equal(t, uint64(0), stats.Misses, "initial misses")

	adapter.hits.Add(5)
	adapter.misses.Add(3)

	stats = adapter.Stats()
	assert.Equal(t, uint64(5), stats.Hits, "hits after increment")
	assert.Equal(t, uint64(3), stats.Misses, "misses after increment")
}

func TestCloseIsNoop(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("close-test")
	err := adapter.Close(nil)
	assert.NoError(t, err)
}

func TestGetMaximumReturnsZero(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("max-test")
	assert.Equal(t, uint64(0), adapter.GetMaximum())
}

func TestWeightedSizeReturnsZero(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("weight-test")
	assert.Equal(t, uint64(0), adapter.WeightedSize())
}

func TestSetMaximumIsNoop(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("setmax-test")

	adapter.SetMaximum(1000)
}

func TestSupportsSearchReturnsFalse(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("search-test")
	assert.False(t, adapter.SupportsSearch())
}

func TestGetSchemaReturnsNil(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("schema-test")
	assert.Nil(t, adapter.GetSchema())
}

func TestProviderName(t *testing.T) {
	t.Parallel()

	provider := &FirestoreProvider{}
	assert.Equal(t, "firestore", provider.Name())
}

func TestEncodeKeyDeterministic(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter("deterministic-test")

	keys := []string{
		"simple",
		"with/slash",
		"with%percent",
		"with/slash/and%percent",
		"",
		"a/b/c/d/e",
		strings.Repeat("long", 100),
	}

	for _, key := range keys {
		t.Run(fmt.Sprintf("key=%q", key), func(t *testing.T) {
			t.Parallel()

			encoded1, err := adapter.encodeKey(key)
			require.NoError(t, err)

			encoded2, err := adapter.encodeKey(key)
			require.NoError(t, err)

			assert.Equal(t, encoded1, encoded2, "encoding should be deterministic")

			decoded, err := adapter.decodeKey(encoded1)
			require.NoError(t, err)
			assert.Equal(t, key, decoded, "decode(encode(key)) == key")
		})
	}
}

func makeTags(n int) []string {
	tags := make([]string, n)
	for i := range n {
		tags[i] = fmt.Sprintf("tag-%d", i)
	}
	return tags
}
