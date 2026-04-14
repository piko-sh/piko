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

package cache_provider_dynamodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func newStringAdapter(namespace string) *DynamoDBAdapter[string, string] {
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))
	return &DynamoDBAdapter[string, string]{
		registry:  registry,
		namespace: namespace,
		tableName: "test-table",
	}
}

func newIntKeyAdapter(namespace string) *DynamoDBAdapter[int, string] {
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))
	return &DynamoDBAdapter[int, string]{
		registry:  registry,
		namespace: namespace,
		tableName: "test-table",
	}
}

func TestEncodeKey_StringKey_WithNamespace(t *testing.T) {
	adapter := newStringAdapter("ns:")
	encoded, err := adapter.encodeKey("my-key")
	require.NoError(t, err)
	assert.Equal(t, "ns:my-key", encoded)
}

func TestEncodeKey_StringKey_EmptyNamespace(t *testing.T) {
	adapter := newStringAdapter("")
	encoded, err := adapter.encodeKey("my-key")
	require.NoError(t, err)
	assert.Equal(t, "my-key", encoded)
}

func TestDecodeKey_StringKey_WithNamespace(t *testing.T) {
	adapter := newStringAdapter("ns:")
	decoded, err := adapter.decodeKey("ns:my-key")
	require.NoError(t, err)
	assert.Equal(t, "my-key", decoded)
}

func TestDecodeKey_StringKey_EmptyNamespace(t *testing.T) {
	adapter := newStringAdapter("")
	decoded, err := adapter.decodeKey("my-key")
	require.NoError(t, err)
	assert.Equal(t, "my-key", decoded)
}

func TestEncodeDecodeKey_StringRoundTrip(t *testing.T) {
	adapter := newStringAdapter("app:")

	original := "user:42:profile"
	encoded, err := adapter.encodeKey(original)
	require.NoError(t, err)

	decoded, err := adapter.decodeKey(encoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestEncodeDecodeKey_StringRoundTrip_VariousValues(t *testing.T) {
	adapter := newStringAdapter("cache:")

	testCases := []string{
		"simple",
		"with spaces",
		"with/slashes",
		"with:colons",
		"unicode-\u00e9\u00e8\u00ea",
		"",
		"a",
		"very-long-key-" + string(make([]byte, 200)),
	}

	for _, original := range testCases {
		t.Run(original, func(t *testing.T) {
			encoded, err := adapter.encodeKey(original)
			require.NoError(t, err)

			decoded, err := adapter.decodeKey(encoded)
			require.NoError(t, err)

			assert.Equal(t, original, decoded)
		})
	}
}

func TestEncodeKey_NamespacePrefixApplied(t *testing.T) {
	adapter := newStringAdapter("myapp:")
	encoded, err := adapter.encodeKey("item")
	require.NoError(t, err)

	assert.Contains(t, encoded, "myapp:")
	assert.True(t, len(encoded) > len("myapp:"))
}

func TestDecodeKey_MissingNamespacePrefix_ReturnsError(t *testing.T) {
	adapter := newStringAdapter("ns:")
	_, err := adapter.decodeKey("wrong-prefix:my-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "namespace prefix")
}

func TestEncodeKey_ColonSuffixInNamespace(t *testing.T) {

	adapter := newStringAdapter("test:")
	encoded, err := adapter.encodeKey("hello")
	require.NoError(t, err)
	assert.Equal(t, "test:hello", encoded)
}

func TestEncodeKey_IntKey_WithoutKeyRegistry(t *testing.T) {
	adapter := newIntKeyAdapter("ns:")
	encoded, err := adapter.encodeKey(42)
	require.NoError(t, err)

	assert.Equal(t, "ns:42", encoded)
}

func TestDecodeKey_IntKey_WithoutKeyRegistry(t *testing.T) {
	adapter := newIntKeyAdapter("ns:")
	decoded, err := adapter.decodeKey("ns:42")
	require.NoError(t, err)
	assert.Equal(t, 42, decoded)
}

func TestEncodeDecodeKey_IntRoundTrip(t *testing.T) {
	adapter := newIntKeyAdapter("nums:")

	original := 12345
	encoded, err := adapter.encodeKey(original)
	require.NoError(t, err)

	decoded, err := adapter.decodeKey(encoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestEncodeDecodeKey_IntRoundTrip_Zero(t *testing.T) {
	adapter := newIntKeyAdapter("nums:")

	encoded, err := adapter.encodeKey(0)
	require.NoError(t, err)

	decoded, err := adapter.decodeKey(encoded)
	require.NoError(t, err)

	assert.Equal(t, 0, decoded)
}

func TestEncodeDecodeKey_IntRoundTrip_Negative(t *testing.T) {
	adapter := newIntKeyAdapter("nums:")

	encoded, err := adapter.encodeKey(-99)
	require.NoError(t, err)

	decoded, err := adapter.decodeKey(encoded)
	require.NoError(t, err)

	assert.Equal(t, -99, decoded)
}

func TestEncodeDecodeValue_StringRoundTrip(t *testing.T) {
	adapter := newStringAdapter("test:")

	original := "hello world"
	encoded, err := adapter.encodeValue(original)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	decoded, err := adapter.decodeValue(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestEncodeDecodeValue_EmptyString(t *testing.T) {
	adapter := newStringAdapter("test:")

	encoded, err := adapter.encodeValue("")
	require.NoError(t, err)

	decoded, err := adapter.decodeValue(encoded)
	require.NoError(t, err)
	assert.Equal(t, "", decoded)
}

func TestEncodeDecodeValue_UnicodeString(t *testing.T) {
	adapter := newStringAdapter("test:")

	original := "caf\u00e9 \u2603 \U0001f600"
	encoded, err := adapter.encodeValue(original)
	require.NoError(t, err)

	decoded, err := adapter.decodeValue(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestNamespaceFormatting_ColonAppended(t *testing.T) {

	namespace := "myns"
	if namespace[len(namespace)-1] != ':' {
		namespace = namespace + ":"
	}
	assert.Equal(t, "myns:", namespace)
}

func TestNamespaceFormatting_AlreadyHasColon(t *testing.T) {
	namespace := "myns:"
	if namespace[len(namespace)-1] != ':' {
		namespace = namespace + ":"
	}
	assert.Equal(t, "myns:", namespace)
}

func TestNamespaceFormatting_MultipleColons(t *testing.T) {
	namespace := "a:b:c:"
	if namespace[len(namespace)-1] != ':' {
		namespace = namespace + ":"
	}
	assert.Equal(t, "a:b:c:", namespace)
}
