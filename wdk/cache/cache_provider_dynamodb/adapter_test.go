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
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

func newTestAdapter(namespace string) *DynamoDBAdapter[string, string] {
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))
	return &DynamoDBAdapter[string, string]{
		registry:  registry,
		namespace: namespace,
		tableName: "test-table",
		ttl:       time.Hour,
	}
}

func TestIsItemExpired_FutureTTL_NotExpired(t *testing.T) {
	futureUnix := time.Now().Add(10 * time.Minute).Unix()
	item := map[string]types.AttributeValue{
		attrTTLUnix: &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", futureUnix)},
	}
	assert.False(t, isItemExpired(item))
}

func TestIsItemExpired_PastTTL_Expired(t *testing.T) {
	pastUnix := time.Now().Add(-10 * time.Minute).Unix()
	item := map[string]types.AttributeValue{
		attrTTLUnix: &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", pastUnix)},
	}
	assert.True(t, isItemExpired(item))
}

func TestIsItemExpired_NoTTLAttribute_NotExpired(t *testing.T) {
	item := map[string]types.AttributeValue{
		attrPK: &types.AttributeValueMemberS{Value: "some-key"},
	}
	assert.False(t, isItemExpired(item))
}

func TestIsItemExpired_NonNumericTTL_NotExpired(t *testing.T) {
	item := map[string]types.AttributeValue{
		attrTTLUnix: &types.AttributeValueMemberS{Value: "not-a-number"},
	}
	assert.False(t, isItemExpired(item))
}

func TestIsItemExpired_ExactlyAtCurrentTime_EdgeCase(t *testing.T) {

	nowUnix := time.Now().Unix()
	item := map[string]types.AttributeValue{
		attrTTLUnix: &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nowUnix)},
	}

	assert.False(t, isItemExpired(item))
}

func TestIsItemExpired_MalformedNumber_NotExpired(t *testing.T) {
	item := map[string]types.AttributeValue{
		attrTTLUnix: &types.AttributeValueMemberN{Value: "abc"},
	}
	assert.False(t, isItemExpired(item))
}

func TestIsItemExpired_EmptyItem_NotExpired(t *testing.T) {
	item := map[string]types.AttributeValue{}
	assert.False(t, isItemExpired(item))
}

func TestIsItemExpired_BinaryTTLAttribute_NotExpired(t *testing.T) {

	item := map[string]types.AttributeValue{
		attrTTLUnix: &types.AttributeValueMemberB{Value: []byte("12345")},
	}
	assert.False(t, isItemExpired(item))
}

func TestCalculateTTLUnix_ReturnsFutureTimestamp(t *testing.T) {
	ttl := 30 * time.Minute
	result := calculateTTLUnix(ttl)
	assert.Greater(t, result, time.Now().Unix())
}

func TestCalculateTTLUnix_ApproximatelyCorrectDuration(t *testing.T) {
	ttl := 2 * time.Hour
	before := time.Now().Add(ttl).Unix()
	result := calculateTTLUnix(ttl)
	after := time.Now().Add(ttl).Unix()

	assert.InDelta(t, before, result, 2, "TTL unix should be approximately now + duration")
	assert.GreaterOrEqual(t, result, before-1)
	assert.LessOrEqual(t, result, after+1)
}

func TestCalculateTTLUnix_ShortDuration(t *testing.T) {
	ttl := 1 * time.Second
	now := time.Now().Unix()
	result := calculateTTLUnix(ttl)

	assert.InDelta(t, now+1, result, 2)
}

func TestCalculateTTLUnix_ZeroDuration(t *testing.T) {
	result := calculateTTLUnix(0)
	now := time.Now().Unix()
	assert.InDelta(t, now, result, 2)
}

func TestCalculateTTLUnix_LargeDuration(t *testing.T) {
	ttl := 365 * 24 * time.Hour
	expected := time.Now().Add(ttl).Unix()
	result := calculateTTLUnix(ttl)
	assert.InDelta(t, expected, result, 2)
}

func TestTagPrefix_ForwardMapping(t *testing.T) {
	namespace := "myapp:"
	tag := "session"
	expected := "myapp:tag:session"
	assert.Equal(t, expected, namespace+tagPrefix+tag)
}

func TestKeyTagsPrefix_ReverseMapping(t *testing.T) {
	namespace := "myapp:"
	key := "user-123"
	expected := "myapp:keytags:user-123"
	assert.Equal(t, expected, namespace+keyTagsPrefix+key)
}

func TestTagPrefix_EmptyNamespace(t *testing.T) {
	namespace := ""
	tag := "important"
	expected := "tag:important"
	assert.Equal(t, expected, namespace+tagPrefix+tag)
}

func TestKeyTagsPrefix_EmptyNamespace(t *testing.T) {
	namespace := ""
	key := "item-42"
	expected := "keytags:item-42"
	assert.Equal(t, expected, namespace+keyTagsPrefix+key)
}

func TestTagPrefix_SpecialCharactersInNamespace(t *testing.T) {
	namespace := "app/v2:"
	tag := "user:active"
	expected := "app/v2:tag:user:active"
	assert.Equal(t, expected, namespace+tagPrefix+tag)
}

func TestKeyTagsPrefix_SpecialCharactersInNamespace(t *testing.T) {
	namespace := "app/v2:"
	key := "key:with:colons"
	expected := "app/v2:keytags:key:with:colons"
	assert.Equal(t, expected, namespace+keyTagsPrefix+key)
}

func TestTagPrefixConstant(t *testing.T) {
	assert.Equal(t, "tag:", tagPrefix)
}

func TestKeyTagsPrefixConstant(t *testing.T) {
	assert.Equal(t, "keytags:", keyTagsPrefix)
}

func TestItemConstruction_AllAttributesPresent(t *testing.T) {
	keyString := "ns:mykey"
	valBytes := []byte(`"hello"`)
	ttlUnix := calculateTTLUnix(time.Hour)
	namespace := "ns:"

	item := map[string]types.AttributeValue{
		attrPK:        &types.AttributeValueMemberS{Value: keyString},
		attrSK:        &types.AttributeValueMemberS{Value: skData},
		attrValue:     &types.AttributeValueMemberB{Value: valBytes},
		attrTTLUnix:   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
		attrVersion:   &types.AttributeValueMemberN{Value: "1"},
		attrNamespace: &types.AttributeValueMemberS{Value: namespace},
	}

	assert.Len(t, item, 6)

	pkAttr, ok := item[attrPK].(*types.AttributeValueMemberS)
	require.True(t, ok, "pk should be a string attribute")
	assert.Equal(t, keyString, pkAttr.Value)

	skAttr, ok := item[attrSK].(*types.AttributeValueMemberS)
	require.True(t, ok, "sk should be a string attribute")
	assert.Equal(t, "#DATA", skAttr.Value)
	assert.Equal(t, skData, skAttr.Value)

	valAttr, ok := item[attrValue].(*types.AttributeValueMemberB)
	require.True(t, ok, "val should be a binary attribute")
	assert.Equal(t, valBytes, valAttr.Value)

	ttlAttr, ok := item[attrTTLUnix].(*types.AttributeValueMemberN)
	require.True(t, ok, "ttl_unix should be a number attribute")
	assert.Equal(t, fmt.Sprintf("%d", ttlUnix), ttlAttr.Value)

	versionAttr, ok := item[attrVersion].(*types.AttributeValueMemberN)
	require.True(t, ok, "version should be a number attribute")
	assert.Equal(t, "1", versionAttr.Value)

	nsAttr, ok := item[attrNamespace].(*types.AttributeValueMemberS)
	require.True(t, ok, "ns should be a string attribute")
	assert.Equal(t, namespace, nsAttr.Value)
}

func TestSkDataConstant(t *testing.T) {
	assert.Equal(t, "#DATA", skData)
}

func TestBatchWriteMaxItems_Is25(t *testing.T) {
	assert.Equal(t, 25, batchWriteMaxItems)
}

func TestBatchGetMaxItems_Is100(t *testing.T) {
	assert.Equal(t, 100, batchGetMaxItems)
}

func chunkSlice[T any](items []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(items); i += chunkSize {
		end := min(i+chunkSize, len(items))
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func TestChunkSlice_ExactMultiple(t *testing.T) {
	items := make([]int, 50)
	for i := range items {
		items[i] = i
	}
	chunks := chunkSlice(items, 25)
	assert.Len(t, chunks, 2)
	assert.Len(t, chunks[0], 25)
	assert.Len(t, chunks[1], 25)
}

func TestChunkSlice_NonExactMultiple(t *testing.T) {
	items := make([]int, 30)
	for i := range items {
		items[i] = i
	}
	chunks := chunkSlice(items, 25)
	assert.Len(t, chunks, 2)
	assert.Len(t, chunks[0], 25)
	assert.Len(t, chunks[1], 5)
}

func TestChunkSlice_SmallerThanChunkSize(t *testing.T) {
	items := make([]int, 10)
	chunks := chunkSlice(items, 25)
	assert.Len(t, chunks, 1)
	assert.Len(t, chunks[0], 10)
}

func TestChunkSlice_EmptySlice(t *testing.T) {
	var items []int
	chunks := chunkSlice(items, 25)
	assert.Empty(t, chunks)
}

func TestChunkSlice_SingleElement(t *testing.T) {
	items := []int{42}
	chunks := chunkSlice(items, 25)
	assert.Len(t, chunks, 1)
	assert.Equal(t, []int{42}, chunks[0])
}

func TestChunkSlice_ChunkSizeOfOne(t *testing.T) {
	items := []int{1, 2, 3}
	chunks := chunkSlice(items, 1)
	assert.Len(t, chunks, 3)
	for _, chunk := range chunks {
		assert.Len(t, chunk, 1)
	}
}

func TestChunkSlice_WriteRequestsBatchSize(t *testing.T) {

	items := make([]string, 63)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}
	chunks := chunkSlice(items, batchWriteMaxItems)
	assert.Len(t, chunks, 3)
	assert.Len(t, chunks[0], 25)
	assert.Len(t, chunks[1], 25)
	assert.Len(t, chunks[2], 13)
}

func TestConfig_RegistryRequired(t *testing.T) {

	config := Config{}
	assert.Nil(t, config.Registry, "zero-value Config should have nil Registry")
}

func TestConfig_DefaultTableName(t *testing.T) {
	config := Config{}
	assert.Empty(t, config.TableName, "zero-value Config should have empty TableName")
	assert.Equal(t, "piko_cache", defaultTableName, "defaultTableName constant should be piko_cache")
}

func TestConfig_ZeroValues(t *testing.T) {
	config := Config{}

	assert.Equal(t, time.Duration(0), config.DefaultTTL)
	assert.Equal(t, time.Duration(0), config.OperationTimeout)
	assert.Equal(t, time.Duration(0), config.AtomicOperationTimeout)
	assert.Equal(t, time.Duration(0), config.BulkOperationTimeout)
	assert.Equal(t, time.Duration(0), config.FlushTimeout)
	assert.Equal(t, time.Duration(0), config.SearchTimeout)
	assert.Equal(t, 0, config.MaxComputeRetries)
	assert.False(t, config.AutoCreateTable)
	assert.False(t, config.ConsistentReads)
	assert.Empty(t, config.Region)
	assert.Empty(t, config.EndpointURL)
	assert.Nil(t, config.AWSConfig)
	assert.Nil(t, config.KeyRegistry)
}

func TestAttributeNameConstants(t *testing.T) {
	assert.Equal(t, "pk", attrPK)
	assert.Equal(t, "sk", attrSK)
	assert.Equal(t, "val", attrValue)
	assert.Equal(t, "ttl_unix", attrTTLUnix)
	assert.Equal(t, "version", attrVersion)
	assert.Equal(t, "ns", attrNamespace)
}

func TestDefaultTableName(t *testing.T) {
	assert.Equal(t, "piko_cache", defaultTableName)
}

func TestGSINamespaceName(t *testing.T) {
	assert.Equal(t, "gsi_ns", gsiNamespaceName)
}

func TestTableActiveCheckInterval(t *testing.T) {
	assert.Equal(t, 500*time.Millisecond, tableActiveCheckInterval)
}

func TestTableActiveMaxAttempts(t *testing.T) {
	assert.Equal(t, 60, tableActiveMaxAttempts)
}

func TestIsConditionalCheckFailed_WithConditionError(t *testing.T) {
	err := &types.ConditionalCheckFailedException{
		Message: aws.String("conditional check failed"),
	}
	assert.True(t, isConditionalCheckFailed(err))
}

func TestIsConditionalCheckFailed_WithOtherError(t *testing.T) {
	err := fmt.Errorf("some other error")
	assert.False(t, isConditionalCheckFailed(err))
}

func TestIsConditionalCheckFailed_WithWrappedConditionError(t *testing.T) {
	inner := &types.ConditionalCheckFailedException{
		Message: aws.String("wrapped conditional check"),
	}
	err := fmt.Errorf("outer: %w", inner)
	assert.True(t, isConditionalCheckFailed(err))
}

func TestIsConditionalCheckFailed_WithNilError(t *testing.T) {
	assert.False(t, isConditionalCheckFailed(nil))
}

func TestSupportsSearch_AlwaysFalse(t *testing.T) {
	adapter := newTestAdapter("test:")
	assert.False(t, adapter.SupportsSearch())
}

func TestGetSchema_NilByDefault(t *testing.T) {
	adapter := newTestAdapter("test:")
	assert.Nil(t, adapter.GetSchema())
}

func TestProviderName(t *testing.T) {
	provider := &DynamoDBProvider{}
	assert.Equal(t, "dynamodb", provider.Name())
}
