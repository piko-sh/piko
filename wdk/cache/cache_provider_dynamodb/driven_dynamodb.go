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
	"context"
	"fmt"
	"iter"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// logKeyField is the attribute key used when logging DynamoDB cache keys.
	logKeyField = "key"

	// errFmtEncodeKey is the format string used when key encoding fails.
	errFmtEncodeKey = "failed to encode key: %w"
)

// DynamoDBAdapter implements the ProviderPort using a DynamoDB client. It
// supports generics by encoding keys to strings and using a type-driven
// EncodingRegistry for values.
type DynamoDBAdapter[K comparable, V any] struct {
	// expiryCalculator sets the expiry time for each key; optional.
	expiryCalculator cache.ExpiryCalculator[K, V]

	// refreshCalculator calculates when entries become ready for background
	// refresh; optional.
	refreshCalculator cache.RefreshCalculator[K, V]

	// sf deduplicates concurrent loads for the same key.
	sf singleflight.Group

	// registry encodes values before they are stored.
	registry *cache.EncodingRegistry

	// client is the DynamoDB client for storage operations.
	client *dynamodb.Client

	// keyRegistry stores encoders for complex key types; nil uses fmt.Sprintf.
	keyRegistry *cache.EncodingRegistry

	// schema is the search schema for this cache; nil means search is disabled.
	schema *cache.SearchSchema

	// fieldExtractor extracts field values from cached items for filter
	// matching and sorting in Query.
	fieldExtractor *cache_domain.FieldExtractor[V]

	// gsiFields maps search field names to their GSI index names. When a field
	// has a GSI, Query uses DynamoDB Query instead of Scan for that field,
	// reading only matching items rather than the full namespace.
	gsiFields map[string]string

	// namespace is the prefix added to all partition keys in DynamoDB.
	namespace string

	// tableName is the DynamoDB table used for storage.
	tableName string

	// hits tracks the number of cache hits for Stats.
	hits atomic.Uint64

	// misses tracks the number of cache misses for Stats.
	misses atomic.Uint64

	// ttl is the default time-to-live for cache entries.
	ttl time.Duration

	// operationTimeout is the time limit for a single DynamoDB operation.
	operationTimeout time.Duration

	// atomicOperationTimeout is the time limit for conditional write
	// operations.
	atomicOperationTimeout time.Duration

	// bulkOperationTimeout is the maximum time for bulk operations like
	// BatchGetItem and BatchWriteItem.
	bulkOperationTimeout time.Duration

	// flushTimeout is the time limit for InvalidateAll operations.
	flushTimeout time.Duration

	// searchTimeout is the time limit for search operations.
	searchTimeout time.Duration

	// maxComputeRetries is the maximum number of retry attempts for optimistic
	// locking in Compute methods.
	maxComputeRetries int

	// consistentReads enables strongly consistent reads when true.
	consistentReads bool
}

var _ cache.ProviderPort[any, any] = (*DynamoDBAdapter[any, any])(nil)

// encodeKey converts a key of type K to a DynamoDB partition key string.
//
// Takes key (K) which is the cache key to encode.
//
// Returns string which is the encoded key, with namespace prefix.
// Returns error when no encoder is registered for the key type or when
// marshalling fails.
func (a *DynamoDBAdapter[K, V]) encodeKey(key K) (string, error) {
	return cache_domain.EncodeKey(key, a.namespace, a.keyRegistry)
}

// decodeKey converts a DynamoDB key string back to a key of type K.
//
// Takes keyString (string) which is the DynamoDB key to decode.
//
// Returns K which is the decoded key value.
// Returns error when the namespace prefix is missing, decoding fails, or no
// encoder is registered for the key type.
func (a *DynamoDBAdapter[K, V]) decodeKey(keyString string) (K, error) {
	return cache_domain.DecodeKey[K](keyString, a.namespace, a.keyRegistry)
}

// decodeValue decodes bytes into a value of type V using the registry.
//
// Takes valBytes ([]byte) which contains the encoded data to decode.
//
// Returns V which is the decoded value.
// Returns error when the encoder cannot be found, unmarshalling fails, or type
// assertion fails.
func (a *DynamoDBAdapter[K, V]) decodeValue(valBytes []byte) (V, error) {
	return cache_domain.DecodeValue[V](valBytes, a.registry)
}

// encodeValue encodes a value of type V to bytes using the registry.
//
// Takes value (V) which is the value to encode.
//
// Returns []byte which contains the encoded value.
// Returns error when no encoder is found for the value type or encoding fails.
func (a *DynamoDBAdapter[K, V]) encodeValue(value V) ([]byte, error) {
	return cache_domain.EncodeValue(value, a.registry)
}

// calculateTTLUnix returns the Unix epoch seconds for the expiry time.
//
// Takes ttl (time.Duration) which is the duration until expiry.
//
// Returns int64 which is the Unix timestamp in seconds.
func calculateTTLUnix(ttl time.Duration) int64 {
	return time.Now().Add(ttl).Unix()
}

// upsertItem stores a cache item using UpdateItem with ADD on the version
// attribute, atomically incrementing the version on every write to preserve
// the optimistic locking invariant that Compute relies on.
//
// Takes keyString (string) which is the encoded partition key.
// Takes valBytes ([]byte) which is the encoded value to store.
// Takes ttlUnix (int64) which is the Unix epoch expiry timestamp in seconds.
//
// Returns error when the DynamoDB UpdateItem call fails.
func (a *DynamoDBAdapter[K, V]) upsertItem(ctx context.Context, keyString string, valBytes []byte, ttlUnix int64) error {
	_, err := a.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
		UpdateExpression: aws.String("SET #val = :val, #ttl = :ttl, #ns = :ns ADD #ver :one"),
		ExpressionAttributeNames: map[string]string{
			"#val": attrValue,
			"#ttl": attrTTLUnix,
			"#ns":  attrNamespace,
			"#ver": attrVersion,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val": &types.AttributeValueMemberB{Value: valBytes},
			":ttl": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
			":ns":  &types.AttributeValueMemberS{Value: a.namespace},
			":one": &types.AttributeValueMemberN{Value: "1"},
		},
	})
	return err
}

// isItemExpired checks whether an item's TTL has passed, performing a
// client-side expiry check because DynamoDB TTL deletion is eventually
// consistent.
//
// Takes item (map[string]types.AttributeValue) which is the DynamoDB item.
//
// Returns bool which is true when the item's TTL has passed.
func isItemExpired(item map[string]types.AttributeValue) bool {
	ttlAttr, ok := item[attrTTLUnix]
	if !ok {
		return false
	}
	ttlNum, ok := ttlAttr.(*types.AttributeValueMemberN)
	if !ok {
		return false
	}
	ttlUnix, err := strconv.ParseInt(ttlNum.Value, 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() > ttlUnix
}

// GetIfPresent retrieves a value from the cache if it exists, without blocking
// or loading.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns V which is the cached value, or zero value if not found.
// Returns bool which indicates whether the key was present in the cache.
// Returns error when the operation fails (e.g. network error).
func (a *DynamoDBAdapter[K, V]) GetIfPresent(ctx context.Context, key K) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("dynamodb GetIfPresent exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	output, err := a.client.GetItem(timeoutCtx, &dynamodb.GetItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
		ConsistentRead: aws.Bool(a.consistentReads),
	})
	if err != nil {
		return *new(V), false, fmt.Errorf("dynamodb GetItem failed: %w", err)
	}

	if output.Item == nil {
		a.misses.Add(1)
		return *new(V), false, nil
	}

	if isItemExpired(output.Item) {
		a.misses.Add(1)
		return *new(V), false, nil
	}

	valAttr, ok := output.Item[attrValue]
	if !ok {
		a.misses.Add(1)
		return *new(V), false, nil
	}

	valBytes, ok := valAttr.(*types.AttributeValueMemberB)
	if !ok {
		return *new(V), false, fmt.Errorf("unexpected value attribute type for key %q", keyString)
	}

	value, err := a.decodeValue(valBytes.Value)
	if err != nil {
		return *new(V), false, fmt.Errorf("failed to decode value for key %q: %w", keyString, err)
	}

	a.hits.Add(1)
	return value, true, nil
}

// Get retrieves a value from the cache, loading it via the provided loader if
// not present.
//
// Takes key (K) which identifies the cached value to retrieve.
// Takes loader (Loader[K, V]) which loads the value if not already cached.
//
// Returns V which is the cached or newly loaded value.
// Returns error when key encoding fails, the loader fails, or type assertion
// fails.
func (a *DynamoDBAdapter[K, V]) Get(ctx context.Context, key K, loader cache.Loader[K, V]) (V, error) {
	ctx, l := logger.From(ctx, log)
	keyString, err := a.encodeKey(key)
	if err != nil {
		return *new(V), fmt.Errorf(errFmtEncodeKey, err)
	}

	result, err, _ := a.sf.Do(keyString, func() (any, error) {
		if v, ok, getErr := a.GetIfPresent(ctx, key); getErr != nil {
			return nil, getErr
		} else if ok {
			return v, nil
		}

		loadedVal, loadErr := loader.Load(ctx, key)
		if loadErr != nil {
			return nil, loadErr
		}

		if setErr := a.Set(ctx, key, loadedVal); setErr != nil {
			l.Warn("Failed to cache loaded value", logger.Error(setErr))
		}
		return loadedVal, nil
	})

	if err != nil {
		return *new(V), err
	}
	value, ok := result.(V)
	if !ok {
		return *new(V), fmt.Errorf("type assertion failed: expected %T, got %T", *new(V), result)
	}
	return value, nil
}

// Set stores a key-value pair in the cache with optional tags for grouped
// invalidation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes tags (...string) which provide optional grouping for bulk invalidation.
//
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) Set(ctx context.Context, key K, value V, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("dynamodb Set exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	ttl := a.ttl
	if a.expiryCalculator != nil {
		entry := cache.Entry[K, V]{
			Key:            key,
			Value:          value,
			Weight:         0,
			SnapshotAtNano: time.Now().UnixNano(),
		}
		ttl = a.expiryCalculator.ExpireAfterCreate(entry)
	}

	valBytes, err := a.encodeValue(value)
	if err != nil {
		return fmt.Errorf("failed to encode value: %w", err)
	}

	ttlUnix := calculateTTLUnix(ttl)

	if err := a.upsertItem(timeoutCtx, keyString, valBytes, ttlUnix); err != nil {
		return fmt.Errorf("dynamodb upsert failed: %w", err)
	}

	if len(tags) > 0 {
		if err := removeKeyFromTags(timeoutCtx, a.client, a.tableName, a.namespace, keyString); err != nil {
			l.Warn("Failed to clean old tags on key overwrite", logger.String(logKeyField, keyString), logger.Error(err))
		}

		if err := addTagsToKey(timeoutCtx, a.client, a.tableName, a.namespace, keyString, tags, ttlUnix); err != nil {
			l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
		}
	}

	return nil
}

// SetWithTTL stores a key-value pair with a custom expiry time for this entry.
//
// Takes key (K) which identifies the cache entry.
// Takes value (V) which is the data to store.
// Takes ttl (time.Duration) which sets how long the entry stays valid.
// Takes tags (...string) which links labels to the entry.
//
// Returns error when encoding, marshalling, or the DynamoDB operation fails.
func (a *DynamoDBAdapter[K, V]) SetWithTTL(ctx context.Context, key K, value V, ttl time.Duration, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("dynamodb SetWithTTL exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	valBytes, err := a.encodeValue(value)
	if err != nil {
		l.Warn("Failed to encode value", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("encode failed: %w", err)
	}

	ttlUnix := calculateTTLUnix(ttl)

	if err := a.upsertItem(timeoutCtx, keyString, valBytes, ttlUnix); err != nil {
		l.Warn("DynamoDB upsert with TTL failed", logger.String(logKeyField, keyString), logger.Error(err))
		return fmt.Errorf("dynamodb upsert failed: %w", err)
	}

	if len(tags) > 0 {
		if err := removeKeyFromTags(timeoutCtx, a.client, a.tableName, a.namespace, keyString); err != nil {
			l.Warn("Failed to clean old tags on key overwrite", logger.String(logKeyField, keyString), logger.Error(err))
		}

		if err := addTagsToKey(timeoutCtx, a.client, a.tableName, a.namespace, keyString, tags, ttlUnix); err != nil {
			l.Warn("Failed to add tags to key", logger.String(logKeyField, keyString), logger.Error(err))
		}
	}

	return nil
}

// Invalidate removes a key from the cache and cleans up its tag links.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to remove.
//
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) Invalidate(ctx context.Context, key K) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("dynamodb Invalidate exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := removeKeyFromTags(timeoutCtx, a.client, a.tableName, a.namespace, keyString); err != nil {
		l.Warn("Failed to remove key from tag sets", logger.String(logKeyField, keyString), logger.Error(err))
	}

	_, err = a.client.DeleteItem(timeoutCtx, &dynamodb.DeleteItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamodb DeleteItem failed: %w", err)
	}

	return nil
}

// InvalidateByTags removes all cache entries linked to the given tags.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes tags (...string) which specifies the tags whose entries to remove.
//
// Returns int which is the number of entries removed.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("dynamodb InvalidateByTags exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	count, err := performTagInvalidation(timeoutCtx, a.client, a.tableName, a.namespace, tags)
	if err != nil {
		return 0, fmt.Errorf("failed to invalidate by tags: %w", err)
	}
	return count, nil
}

// InvalidateAll removes all cache entries within the namespace by scanning the
// GSI and batch-deleting items.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	ctx, l := logger.From(ctx, log)
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.flushTimeout, fmt.Errorf("dynamodb InvalidateAll exceeded %s timeout", a.flushTimeout))
	defer cancel()

	l.Internal("Invalidating all keys in namespace",
		logger.String("namespace", a.namespace))

	paginator := a.newNamespacePaginator()

	deletedCount, err := a.drainDeletePages(timeoutCtx, paginator)
	if err != nil {
		return err
	}

	l.Internal("InvalidateAll completed",
		logger.String("namespace", a.namespace),
		logger.Int("keys_deleted", deletedCount))

	return nil
}

// newNamespacePaginator creates a query paginator that scans all items in the
// adapter's namespace via the GSI.
//
// Returns *dynamodb.QueryPaginator which iterates over namespace items.
func (a *DynamoDBAdapter[K, V]) newNamespacePaginator() *dynamodb.QueryPaginator {
	return dynamodb.NewQueryPaginator(a.client, &dynamodb.QueryInput{
		TableName:              aws.String(a.tableName),
		IndexName:              aws.String(gsiNamespaceName),
		KeyConditionExpression: aws.String("#ns = :ns"),
		ExpressionAttributeNames: map[string]string{
			"#ns": attrNamespace,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ns": &types.AttributeValueMemberS{Value: a.namespace},
		},
	})
}

// drainDeletePages iterates through paginator pages, collecting delete requests
// and flushing them in batches.
//
// Takes paginator (*dynamodb.QueryPaginator) which supplies the pages to drain.
//
// Returns int which is the total number of items deleted.
// Returns error when a page fetch fails or the context expires.
func (a *DynamoDBAdapter[K, V]) drainDeletePages(ctx context.Context, paginator *dynamodb.QueryPaginator) (int, error) {
	deletedCount := 0
	var writeRequests []types.WriteRequest

	for paginator.HasMorePages() {
		if ctx.Err() != nil {
			return deletedCount, ctx.Err()
		}

		page, err := paginator.NextPage(ctx)
		if err != nil {
			return deletedCount, fmt.Errorf("failed to scan namespace for InvalidateAll: %w", err)
		}

		writeRequests = collectDeleteRequests(page.Items, writeRequests)

		if len(writeRequests) >= batchWriteMaxItems {
			if err := a.executeBatchDelete(ctx, writeRequests); err != nil {
				return deletedCount, fmt.Errorf("batch delete during InvalidateAll failed: %w", err)
			}
			deletedCount += len(writeRequests)
			writeRequests = writeRequests[:0]
		}
	}

	if len(writeRequests) > 0 {
		if err := a.executeBatchDelete(ctx, writeRequests); err != nil {
			return deletedCount, fmt.Errorf("final batch delete during InvalidateAll failed: %w", err)
		}
		deletedCount += len(writeRequests)
	}

	return deletedCount, nil
}

// collectDeleteRequests appends delete requests for each item that has valid
// pk and sk attributes.
//
// Takes items ([]map[string]types.AttributeValue) which are the DynamoDB items
// to build delete requests for.
// Takes writeRequests ([]types.WriteRequest) which is the slice to append to.
//
// Returns []types.WriteRequest which is the updated slice with new delete
// requests appended.
func collectDeleteRequests(items []map[string]types.AttributeValue, writeRequests []types.WriteRequest) []types.WriteRequest {
	for _, item := range items {
		pk, pkOk := item[attrPK].(*types.AttributeValueMemberS)
		sk, skOk := item[attrSK].(*types.AttributeValueMemberS)
		if !pkOk || !skOk {
			continue
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					attrPK: &types.AttributeValueMemberS{Value: pk.Value},
					attrSK: &types.AttributeValueMemberS{Value: sk.Value},
				},
			},
		})
	}
	return writeRequests
}

// executeBatchDelete performs a BatchWriteItem call with delete requests.
//
// Takes requests ([]types.WriteRequest) which are the delete requests.
//
// Returns error when the batch write fails.
func (a *DynamoDBAdapter[K, V]) executeBatchDelete(ctx context.Context, requests []types.WriteRequest) error {
	output, err := a.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			a.tableName: requests,
		},
	})
	if err != nil {
		return fmt.Errorf("batch delete failed: %w", err)
	}
	return retryUnprocessedWrites(ctx, a.client, output.UnprocessedItems)
}

// BulkGet retrieves multiple values from the cache, loading missing ones via
// the bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader[K, V]) which loads values for any cache misses.
//
// Returns map[K]V which contains the retrieved and loaded values.
// Returns error when the DynamoDB operation or bulk loader fails.
func (a *DynamoDBAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) (map[K]V, error) {
	ctx, l := logger.From(ctx, log)
	if len(keys) == 0 {
		return make(map[K]V), nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("dynamodb BulkGet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	results := make(map[K]V, len(keys))
	mappings := a.encodeKeyMappings(l, keys)

	if len(mappings) == 0 {
		return results, nil
	}

	misses := a.fetchBulkGetChunks(timeoutCtx, l, mappings, results)

	if len(misses) > 0 {
		if err := a.loadAndCacheMisses(ctx, l, misses, bulkLoader, results); err != nil {
			return results, fmt.Errorf("loading cache misses: %w", err)
		}
	}

	return results, nil
}

// bulkGetKeyMapping associates an original key with its encoded string form.
type bulkGetKeyMapping[K comparable] struct {
	// originalKey is the typed cache key before encoding.
	originalKey K

	// keyString is the DynamoDB-encoded partition key string.
	keyString string
}

// encodeKeyMappings encodes all keys and returns their mappings, logging and
// skipping any that fail to encode.
//
// Takes keys ([]K) which are the cache keys to encode.
//
// Returns []bulkGetKeyMapping[K] which maps each successfully encoded key to
// its string form.
func (a *DynamoDBAdapter[K, V]) encodeKeyMappings(l logger.Logger, keys []K) []bulkGetKeyMapping[K] {
	var mappings []bulkGetKeyMapping[K]
	for _, key := range keys {
		keyString, err := a.encodeKey(key)
		if err != nil {
			l.Warn("Failed to encode key in BulkGet, skipping", logger.Error(err))
			continue
		}
		mappings = append(mappings, bulkGetKeyMapping[K]{originalKey: key, keyString: keyString})
	}
	return mappings
}

// fetchBulkGetChunks processes key mappings in batches, populating results and
// returning any keys that were not found (misses).
//
// Takes mappings ([]bulkGetKeyMapping[K]) which are the encoded key pairs to
// fetch.
// Takes results (map[K]V) which collects found values.
//
// Returns []K which contains the keys that were not found in DynamoDB.
func (a *DynamoDBAdapter[K, V]) fetchBulkGetChunks(ctx context.Context, l logger.Logger, mappings []bulkGetKeyMapping[K], results map[K]V) []K {
	var misses []K
	for i := 0; i < len(mappings); i += batchGetMaxItems {
		end := min(i+batchGetMaxItems, len(mappings))
		chunk := mappings[i:end]
		chunkMisses := a.fetchBulkGetChunk(ctx, l, chunk, results)
		misses = append(misses, chunkMisses...)
	}
	return misses
}

// fetchBulkGetChunk executes a single BatchGetItem call for a chunk of keys.
//
// Takes chunk ([]bulkGetKeyMapping[K]) which is the batch of keys to fetch.
// Takes results (map[K]V) which collects found values.
//
// Returns []K which contains keys not found in this chunk.
func (a *DynamoDBAdapter[K, V]) fetchBulkGetChunk(ctx context.Context, l logger.Logger, chunk []bulkGetKeyMapping[K], results map[K]V) []K {
	requestKeys := make([]map[string]types.AttributeValue, len(chunk))
	keyMap := make(map[string]K, len(chunk))
	for j, mapping := range chunk {
		requestKeys[j] = map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: mapping.keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		}
		keyMap[mapping.keyString] = mapping.originalKey
	}

	output, err := a.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			a.tableName: {
				Keys:           requestKeys,
				ConsistentRead: aws.Bool(a.consistentReads),
			},
		},
	})
	if err != nil {
		l.Warn("DynamoDB BatchGetItem failed", logger.Error(err))
		misses := make([]K, len(chunk))
		for i, mapping := range chunk {
			misses[i] = mapping.originalKey
		}
		return misses
	}

	return a.processBulkGetResponse(l, output, keyMap, chunk, results)
}

// processBulkGetResponse extracts values from a BatchGetItem response and
// identifies misses.
//
// Takes output (*dynamodb.BatchGetItemOutput) which is the BatchGetItem
// response.
// Takes keyMap (map[string]K) which maps encoded key strings to original keys.
// Takes chunk ([]bulkGetKeyMapping[K]) which is the batch that was requested.
// Takes results (map[K]V) which collects found values.
//
// Returns []K which contains the keys that were missed or unprocessed.
func (a *DynamoDBAdapter[K, V]) processBulkGetResponse(l logger.Logger, output *dynamodb.BatchGetItemOutput, keyMap map[string]K, chunk []bulkGetKeyMapping[K], results map[K]V) []K {
	var misses []K
	foundKeys := make(map[string]bool)

	for _, item := range output.Responses[a.tableName] {
		originalKey, keyString, isMiss := a.classifyBulkGetItem(l, item, keyMap, results)
		if keyString == "" {
			continue
		}
		foundKeys[keyString] = true
		if isMiss {
			misses = append(misses, originalKey)
		}
	}

	misses = appendUnfoundKeys(misses, chunk, foundKeys)
	misses = appendUnprocessedKeys(misses, output.UnprocessedKeys[a.tableName], keyMap)

	return misses
}

// appendUnfoundKeys adds keys that were not found in the response to the misses
// slice.
//
// Takes misses ([]K) which is the current list of missed keys.
// Takes chunk ([]bulkGetKeyMapping[K]) which is the batch that was requested.
// Takes foundKeys (map[string]bool) which tracks keys present in the response.
//
// Returns []K which is the updated misses slice.
func appendUnfoundKeys[K comparable](misses []K, chunk []bulkGetKeyMapping[K], foundKeys map[string]bool) []K {
	for _, mapping := range chunk {
		if !foundKeys[mapping.keyString] {
			misses = append(misses, mapping.originalKey)
		}
	}
	return misses
}

// appendUnprocessedKeys adds keys from the unprocessed keys response to the
// misses slice.
//
// Takes misses ([]K) which is the current list of missed keys.
// Takes unprocessed (types.KeysAndAttributes) which contains keys DynamoDB did
// not process.
// Takes keyMap (map[string]K) which maps encoded key strings to original keys.
//
// Returns []K which is the updated misses slice.
func appendUnprocessedKeys[K comparable](misses []K, unprocessed types.KeysAndAttributes, keyMap map[string]K) []K {
	for _, unprocessedKey := range unprocessed.Keys {
		if pkAttr, ok := unprocessedKey[attrPK].(*types.AttributeValueMemberS); ok {
			if originalKey, exists := keyMap[pkAttr.Value]; exists {
				misses = append(misses, originalKey)
			}
		}
	}
	return misses
}

// classifyBulkGetItem decodes a single item from a BatchGetItem response and
// stores it in results or marks it as a miss.
//
// Takes item (map[string]types.AttributeValue) which is the DynamoDB item to
// classify.
// Takes keyMap (map[string]K) which maps encoded key strings to original keys.
// Takes results (map[K]V) which collects successfully decoded values.
//
// Returns K which is the original key (zero value if unrecognised).
// Returns string which is the encoded key string (empty if unrecognised).
// Returns bool which is true when the item is expired or cannot be decoded.
func (a *DynamoDBAdapter[K, V]) classifyBulkGetItem(l logger.Logger, item map[string]types.AttributeValue, keyMap map[string]K, results map[K]V) (K, string, bool) {
	var zeroKey K
	pkAttr, ok := item[attrPK].(*types.AttributeValueMemberS)
	if !ok {
		return zeroKey, "", false
	}
	keyString := pkAttr.Value
	originalKey, exists := keyMap[keyString]
	if !exists {
		return zeroKey, "", false
	}

	if isItemExpired(item) {
		return originalKey, keyString, true
	}

	valAttr, ok := item[attrValue].(*types.AttributeValueMemberB)
	if !ok {
		return originalKey, keyString, true
	}

	value, decodeErr := a.decodeValue(valAttr.Value)
	if decodeErr != nil {
		l.Warn("Failed to decode value in BulkGet",
			logger.String(logKeyField, keyString), logger.Error(decodeErr))
		return originalKey, keyString, true
	}

	results[originalKey] = value
	return originalKey, keyString, false
}

// loadAndCacheMisses uses the bulk loader to fetch missing keys, caches them,
// and adds them to results.
//
// Takes misses ([]K) which are the keys not found in DynamoDB.
// Takes bulkLoader (cache.BulkLoader[K, V]) which loads values for the missed
// keys.
// Takes results (map[K]V) which collects the loaded values.
//
// Returns error when the bulk loader fails.
func (a *DynamoDBAdapter[K, V]) loadAndCacheMisses(ctx context.Context, l logger.Logger, misses []K, bulkLoader cache.BulkLoader[K, V], results map[K]V) error {
	loaded, err := bulkLoader.BulkLoad(ctx, misses)
	if err != nil {
		return fmt.Errorf("bulk loader failed: %w", err)
	}
	for k, v := range loaded {
		if setErr := a.Set(ctx, k, v); setErr != nil {
			l.Warn("Failed to cache loaded value in BulkGet", logger.Error(setErr))
		}
		results[k] = v
	}
	return nil
}

// BulkSet stores multiple key-value pairs in the cache using BatchWriteItem
// for efficiency.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with the
// keys.
//
// Returns error when the batch write fails.
func (a *DynamoDBAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	ctx, l := logger.From(ctx, log)
	if len(items) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("dynamodb BulkSet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	writeRequests, err := a.buildAndFlushBulkSetItems(timeoutCtx, l, items, tags)
	if err != nil {
		return fmt.Errorf("bulk set items: %w", err)
	}

	if len(writeRequests) > 0 {
		if err := a.executeBatchWrite(timeoutCtx, writeRequests); err != nil {
			l.Warn("Failed to execute final BulkSet batch", logger.Error(err))
			return fmt.Errorf("bulk set final batch write failed: %w", err)
		}
	}

	return nil
}

// buildAndFlushBulkSetItems iterates over items, encodes each one, and flushes
// the write buffer when it reaches the batch limit.
//
// Takes items (map[K]V) which are the key-value pairs to encode and write.
// Takes tags ([]string) which are the optional tags to associate with each key.
//
// Returns []types.WriteRequest which contains any remaining unflushed requests.
// Returns error when encoding or a batch write fails.
func (a *DynamoDBAdapter[K, V]) buildAndFlushBulkSetItems(
	ctx context.Context,
	l logger.Logger,
	items map[K]V,
	tags []string,
) ([]types.WriteRequest, error) {
	var writeRequests []types.WriteRequest

	for key, value := range items {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		request, keyString, ttlUnix, ok := a.buildBulkSetRequest(l, key, value)
		if !ok {
			continue
		}

		writeRequests = append(writeRequests, request)

		if len(writeRequests) >= batchWriteMaxItems {
			if err := a.executeBatchWrite(ctx, writeRequests); err != nil {
				l.Warn("Failed to execute BulkSet batch", logger.Error(err))
				return nil, fmt.Errorf("bulk set batch write failed: %w", err)
			}
			writeRequests = writeRequests[:0]
		}

		a.applyBulkSetTags(ctx, l, keyString, tags, ttlUnix)
	}

	return writeRequests, nil
}

// applyBulkSetTags associates tags with a key if tags are present.
//
// Takes keyString (string) which is the encoded partition key.
// Takes tags ([]string) which are the tags to associate.
// Takes ttlUnix (int64) which is the Unix epoch expiry for the tag mappings.
func (a *DynamoDBAdapter[K, V]) applyBulkSetTags(ctx context.Context, l logger.Logger, keyString string, tags []string, ttlUnix int64) {
	if len(tags) == 0 {
		return
	}
	if err := addTagsToKey(ctx, a.client, a.tableName, a.namespace, keyString, tags, ttlUnix); err != nil {
		l.Warn("Failed to add tags to key in BulkSet",
			logger.String(logKeyField, keyString), logger.Error(err))
	}
}

// buildBulkSetRequest encodes a key-value pair into a DynamoDB write request
// for use in BulkSet.
//
// Takes key (K) which is the cache key to encode.
// Takes value (V) which is the value to encode and store.
//
// Returns types.WriteRequest which is the prepared write request.
// Returns string which is the encoded key string.
// Returns int64 which is the TTL Unix timestamp.
// Returns bool which is true when encoding succeeded.
func (a *DynamoDBAdapter[K, V]) buildBulkSetRequest(l logger.Logger, key K, value V) (types.WriteRequest, string, int64, bool) {
	keyString, err := a.encodeKey(key)
	if err != nil {
		l.Warn("Failed to encode key in BulkSet, skipping", logger.Error(err))
		return types.WriteRequest{}, "", 0, false
	}

	valBytes, err := a.encodeValue(value)
	if err != nil {
		l.Warn("Failed to encode value in BulkSet",
			logger.String(logKeyField, keyString), logger.Error(err))
		return types.WriteRequest{}, "", 0, false
	}

	entryTTL := a.ttl
	if a.expiryCalculator != nil {
		entry := cache.Entry[K, V]{
			Key: key, Value: value, SnapshotAtNano: time.Now().UnixNano(),
		}
		entryTTL = a.expiryCalculator.ExpireAfterCreate(entry)
	}

	ttlUnix := calculateTTLUnix(entryTTL)

	versionNano := time.Now().UnixNano()

	item := map[string]types.AttributeValue{
		attrPK:        &types.AttributeValueMemberS{Value: keyString},
		attrSK:        &types.AttributeValueMemberS{Value: skData},
		attrValue:     &types.AttributeValueMemberB{Value: valBytes},
		attrTTLUnix:   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
		attrVersion:   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", versionNano)},
		attrNamespace: &types.AttributeValueMemberS{Value: a.namespace},
	}

	return types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}, keyString, ttlUnix, true
}

// executeBatchWrite performs a BatchWriteItem call with the provided requests.
//
// Takes requests ([]types.WriteRequest) which are the write requests.
//
// Returns error when the batch write fails.
func (a *DynamoDBAdapter[K, V]) executeBatchWrite(ctx context.Context, requests []types.WriteRequest) error {
	output, err := a.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			a.tableName: requests,
		},
	})
	if err != nil {
		return fmt.Errorf("batch write failed: %w", err)
	}
	return retryUnprocessedWrites(ctx, a.client, output.UnprocessedItems)
}

// BulkRefresh updates several cache entries in the background using the bulk
// loader.
//
// Takes keys ([]K) which specifies the cache keys to refresh.
// Takes bulkLoader (BulkLoader) which loads values for the given keys.
//
// Safe for concurrent use. Starts a goroutine that runs the bulk loader and
// updates the cache.
func (a *DynamoDBAdapter[K, V]) BulkRefresh(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) {
	ctx, l := logger.From(ctx, log)
	go func() {
		defer goroutine.RecoverPanic(ctx, "cache.dynamodbBulkRefresh")
		loaded, err := bulkLoader.BulkLoad(ctx, keys)
		if err != nil {
			l.Warn("Bulk refresh failed", logger.Error(err))
			return
		}
		for k, v := range loaded {
			if setErr := a.Set(ctx, k, v); setErr != nil {
				l.Warn("Failed to set value during bulk refresh", logger.Error(setErr))
			}
		}
	}()
}

// Refresh asynchronously refreshes a single cache entry using the provided
// loader.
//
// Takes key (K) which identifies the cache entry to refresh.
// Takes loader (Loader[K, V]) which loads the fresh value.
//
// Returns <-chan LoadResult[V] which receives the loaded value or error once
// the background goroutine completes.
//
// Safe for concurrent use.
func (a *DynamoDBAdapter[K, V]) Refresh(ctx context.Context, key K, loader cache.Loader[K, V]) <-chan cache.LoadResult[V] {
	ctx, l := logger.From(ctx, log)
	resultChan := make(chan cache.LoadResult[V], 1)
	go func() {
		defer close(resultChan)
		defer goroutine.RecoverPanic(ctx, "cache.dynamodbRefresh")
		value, err := loader.Load(ctx, key)
		if err == nil {
			if setErr := a.Set(ctx, key, value); setErr != nil {
				l.Warn("Failed to set value during refresh", logger.Error(setErr))
			}
		}
		resultChan <- cache.LoadResult[V]{Value: value, Err: err}
	}()
	return resultChan
}

// All returns an iterator over all key-value pairs in the cache namespace.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found in the
// namespace via DynamoDB GSI query.
func (a *DynamoDBAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()
		_, l := logger.From(ctx, log)

		paginator := dynamodb.NewScanPaginator(a.client, &dynamodb.ScanInput{
			TableName:        aws.String(a.tableName),
			FilterExpression: aws.String("#ns = :ns AND #sk = :sk"),
			ExpressionAttributeNames: map[string]string{
				"#ns": attrNamespace,
				"#sk": attrSK,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":ns": &types.AttributeValueMemberS{Value: a.namespace},
				":sk": &types.AttributeValueMemberS{Value: skData},
			},
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				l.Warn("Failed to scan during All iteration", logger.Error(err))
				return
			}

			if !a.yieldPageItems(l, page.Items, yield) {
				return
			}
		}
	}
}

// yieldPageItems decodes and yields each non-expired item from a scan page.
//
// Takes items ([]map[string]types.AttributeValue) which are the DynamoDB items
// from a single scan page.
// Takes yield (func(K, V) bool) which receives each decoded key-value pair.
//
// Returns false when the yield function signals early termination.
func (a *DynamoDBAdapter[K, V]) yieldPageItems(l logger.Logger, items []map[string]types.AttributeValue, yield func(K, V) bool) bool {
	for _, item := range items {
		if isItemExpired(item) {
			continue
		}

		pkAttr, ok := item[attrPK].(*types.AttributeValueMemberS)
		if !ok {
			continue
		}

		key, err := a.decodeKey(pkAttr.Value)
		if err != nil {
			l.Trace("Failed to decode key during iteration",
				logger.String(logKeyField, pkAttr.Value), logger.Error(err))
			continue
		}

		valAttr, ok := item[attrValue].(*types.AttributeValueMemberB)
		if !ok {
			continue
		}

		value, err := a.decodeValue(valAttr.Value)
		if err != nil {
			l.Trace("Failed to decode value during iteration",
				logger.String(logKeyField, pkAttr.Value), logger.Error(err))
			continue
		}

		if !yield(key, value) {
			return false
		}
	}
	return true
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found in the namespace.
func (a *DynamoDBAdapter[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range a.All() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns an iterator over all values in the cache namespace.
//
// Returns iter.Seq[V] which yields each value found in the namespace.
func (a *DynamoDBAdapter[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range a.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// GetEntry retrieves the full entry metadata for a key including TTL
// information.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns Entry[K, V] which contains the value and metadata.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access patterns or TTL.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to probe.
//
// Returns cache.Entry[K, V] which is the entry snapshot.
// Returns bool which indicates whether the entry exists.
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("dynamodb ProbeEntry exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf(errFmtEncodeKey, err)
	}

	output, err := a.client.GetItem(timeoutCtx, &dynamodb.GetItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
		ConsistentRead: aws.Bool(a.consistentReads),
	})
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("dynamodb GetItem failed: %w", err)
	}

	if output.Item == nil {
		return cache.Entry[K, V]{}, false, nil
	}

	if isItemExpired(output.Item) {
		return cache.Entry[K, V]{}, false, nil
	}

	valAttr, ok := output.Item[attrValue].(*types.AttributeValueMemberB)
	if !ok {
		return cache.Entry[K, V]{}, false, nil
	}

	value, err := a.decodeValue(valAttr.Value)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("failed to decode value: %w", err)
	}

	var expiresAtNano int64
	if ttlAttr, ok := output.Item[attrTTLUnix].(*types.AttributeValueMemberN); ok {
		ttlUnix, parseErr := strconv.ParseInt(ttlAttr.Value, 10, 64)
		if parseErr == nil {
			expiresAtNano = ttlUnix * int64(time.Second)
		}
	}

	entry := cache.Entry[K, V]{
		Key:               key,
		Value:             value,
		Weight:            0,
		ExpiresAtNano:     expiresAtNano,
		RefreshableAtNano: 0,
		SnapshotAtNano:    time.Now().UnixNano(),
	}

	return entry, true, nil
}

// EstimatedSize returns the approximate number of data entries in the cache
// namespace.
//
// Returns int which is the count of items, or zero if the query fails.
func (a *DynamoDBAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("dynamodb EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	count := 0
	paginator := dynamodb.NewScanPaginator(a.client, &dynamodb.ScanInput{
		TableName:        aws.String(a.tableName),
		FilterExpression: aws.String("#ns = :ns AND #sk = :sk"),
		ExpressionAttributeNames: map[string]string{
			"#ns": attrNamespace,
			"#sk": attrSK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ns": &types.AttributeValueMemberS{Value: a.namespace},
			":sk": &types.AttributeValueMemberS{Value: skData},
		},
		Select: types.SelectCount,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			l.Warn("Failed to scan for EstimatedSize", logger.Error(err))
			return count
		}
		count += int(page.Count)
	}

	return count
}

// Stats returns cache statistics based on local atomic counters.
//
// Returns cache.Stats which contains hit and miss counts.
func (a *DynamoDBAdapter[K, V]) Stats() cache.Stats {
	return cache.Stats{
		Hits:             a.hits.Load(),
		Misses:           a.misses.Load(),
		Evictions:        0,
		LoadSuccessCount: 0,
		LoadFailureCount: 0,
		TotalLoadTime:    0,
	}
}

// Close releases any resources used by the cache. For DynamoDB, this is a
// no-op as the SDK client does not require explicit closing.
//
// Returns error (always nil).
func (*DynamoDBAdapter[K, V]) Close(_ context.Context) error {
	return nil
}

// SetExpiresAfter updates the time-to-live for an existing key.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new time-to-live.
//
// Returns error when the operation fails.
func (a *DynamoDBAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("dynamodb SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	ttlUnix := calculateTTLUnix(expiresAfter)

	_, err = a.client.UpdateItem(timeoutCtx, &dynamodb.UpdateItemInput{
		TableName: aws.String(a.tableName),
		Key: map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: keyString},
			attrSK: &types.AttributeValueMemberS{Value: skData},
		},
		UpdateExpression: aws.String("SET #ttl = :ttl"),
		ExpressionAttributeNames: map[string]string{
			"#ttl": attrTTLUnix,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ttl": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamodb UpdateItem for TTL failed: %w", err)
	}

	return nil
}

// GetMaximum returns 0 as DynamoDB does not have a client-side maximum
// capacity concept.
//
// Returns uint64 which is always 0.
func (*DynamoDBAdapter[K, V]) GetMaximum() uint64 {
	return 0
}

// SetMaximum is not supported by the DynamoDB provider.
//
// DynamoDB manages capacity at the service level.
func (*DynamoDBAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the DynamoDB provider and will have no effect.")
}

// WeightedSize returns 0 as DynamoDB does not expose client-side memory
// usage.
//
// Returns uint64 which is always 0.
func (*DynamoDBAdapter[K, V]) WeightedSize() uint64 {
	_, l := logger.From(context.Background(), log)
	l.Warn("WeightedSize is not supported by the DynamoDB provider.")
	return 0
}

// SetRefreshableAfter is a no-op as DynamoDB does not natively support refresh
// scheduling.
//
// Returns error (always nil for this no-op implementation).
func (*DynamoDBAdapter[K, V]) SetRefreshableAfter(ctx context.Context, _ K, _ time.Duration) error {
	_, l := logger.From(ctx, log)
	l.Internal("SetRefreshableAfter is not natively supported by the DynamoDB provider.")
	return nil
}
