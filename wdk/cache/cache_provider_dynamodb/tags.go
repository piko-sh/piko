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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	// tagPrefix is the partition key infix for tag forward mappings.
	tagPrefix = "tag:"

	// keyTagsPrefix is the partition key infix for reverse tag mappings (key
	// -> tags).
	keyTagsPrefix = "keytags:"

	// batchWriteMaxItems is the maximum number of write requests per
	// BatchWriteItem call.
	batchWriteMaxItems = 25

	// batchGetMaxItems is the maximum number of keys per BatchGetItem call.
	batchGetMaxItems = 100
)

// addTagsToKey associates a key with a set of tags in DynamoDB. It creates
// both forward (tag -> key) and reverse (key -> tag) mappings using
// BatchWriteItem for efficiency.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes namespace (string) which is the namespace prefix.
// Takes key (string) which is the cache key to associate with tags.
// Takes tags ([]string) which contains the tag names.
// Takes ttlUnix (int64) which is the TTL for the tag mappings.
//
// Returns error when the batch write fails.
func addTagsToKey(ctx context.Context, client *dynamodb.Client, tableName string, namespace string, key string, tags []string, ttlUnix int64) error {
	if len(tags) == 0 {
		return nil
	}

	var writeRequests []types.WriteRequest

	for _, tag := range tags {
		forwardItem := map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: namespace + tagPrefix + tag},
			attrSK: &types.AttributeValueMemberS{Value: key},
		}
		if ttlUnix > 0 {
			forwardItem[attrTTLUnix] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)}
		}
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: forwardItem},
		})

		reverseItem := map[string]types.AttributeValue{
			attrPK: &types.AttributeValueMemberS{Value: namespace + keyTagsPrefix + key},
			attrSK: &types.AttributeValueMemberS{Value: tag},
		}
		if ttlUnix > 0 {
			reverseItem[attrTTLUnix] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttlUnix)}
		}
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: reverseItem},
		})
	}

	for i := 0; i < len(writeRequests); i += batchWriteMaxItems {
		end := min(i+batchWriteMaxItems, len(writeRequests))

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableName: writeRequests[i:end],
			},
		}

		output, err := client.BatchWriteItem(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to add tags for key %q: %w", key, err)
		}

		if err := retryUnprocessedWrites(ctx, client, output.UnprocessedItems); err != nil {
			return fmt.Errorf("failed to write unprocessed tag items for key %q: %w", key, err)
		}
	}

	return nil
}

// getKeysByTags retrieves all unique keys associated with the given tags by
// querying the forward tag mappings.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes namespace (string) which is the namespace prefix.
// Takes tags ([]string) which specifies the tags to look up.
//
// Returns []string which contains the unique keys associated with the tags.
// Returns error when a query operation fails.
func getKeysByTags(ctx context.Context, client *dynamodb.Client, tableName string, namespace string, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	keySet := make(map[string]struct{})
	for _, tag := range tags {
		tagPK := namespace + tagPrefix + tag

		paginator := dynamodb.NewQueryPaginator(client, &dynamodb.QueryInput{
			TableName:              aws.String(tableName),
			KeyConditionExpression: aws.String("#pk = :pk"),
			ProjectionExpression:   aws.String("#sk"),
			ExpressionAttributeNames: map[string]string{
				"#pk": attrPK,
				"#sk": attrSK,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: tagPK},
			},
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to query keys for tag %q: %w", tag, err)
			}
			for _, item := range page.Items {
				if sk, ok := item[attrSK].(*types.AttributeValueMemberS); ok {
					keySet[sk.Value] = struct{}{}
				}
			}
		}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	return keys, nil
}

// removeKeyFromTags removes a key from all its associated tag forward
// mappings, then deletes the reverse mapping. This prevents stale references
// when keys are deleted directly via Invalidate().
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes namespace (string) which is the namespace prefix.
// Takes key (string) which is the cache key to remove from tag sets.
//
// Returns error when fetching tags or removing the key fails.
func removeKeyFromTags(ctx context.Context, client *dynamodb.Client, tableName string, namespace string, key string) error {
	reversePK := namespace + keyTagsPrefix + key

	tags, err := queryTagsForKey(ctx, client, tableName, reversePK)
	if err != nil {
		return fmt.Errorf("failed to get tags for key %q: %w", key, err)
	}

	if len(tags) == 0 {
		return nil
	}

	writeRequests := buildTagRemovalRequests(namespace, key, reversePK, tags)

	return executeBatchWriteChunks(ctx, client, tableName, writeRequests,
		fmt.Sprintf("failed to remove key %q from tag sets", key))
}

// queryTagsForKey queries the reverse mapping to find all tags for a key.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes reversePK (string) which is the reverse mapping partition key.
//
// Returns []string which contains the tag names.
// Returns error when the query fails.
func queryTagsForKey(ctx context.Context, client *dynamodb.Client, tableName string, reversePK string) ([]string, error) {
	paginator := dynamodb.NewQueryPaginator(client, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("#pk = :pk"),
		ProjectionExpression:   aws.String("#sk"),
		ExpressionAttributeNames: map[string]string{
			"#pk": attrPK,
			"#sk": attrSK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: reversePK},
		},
	})

	var tags []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("querying reverse tag mapping: %w", err)
		}
		for _, item := range page.Items {
			if sk, ok := item[attrSK].(*types.AttributeValueMemberS); ok {
				tags = append(tags, sk.Value)
			}
		}
	}
	return tags, nil
}

// buildTagRemovalRequests creates delete requests for both forward (tag -> key)
// and reverse (key -> tag) mappings.
//
// Takes namespace (string) which is the namespace prefix.
// Takes key (string) which is the cache key to remove from tag sets.
// Takes reversePK (string) which is the reverse mapping partition key.
// Takes tags ([]string) which are the tags to remove.
//
// Returns []types.WriteRequest which contains the delete requests.
func buildTagRemovalRequests(namespace string, key string, reversePK string, tags []string) []types.WriteRequest {
	writeRequests := make([]types.WriteRequest, 0, len(tags)*2)

	for _, tag := range tags {
		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					attrPK: &types.AttributeValueMemberS{Value: namespace + tagPrefix + tag},
					attrSK: &types.AttributeValueMemberS{Value: key},
				},
			},
		})
	}

	for _, tag := range tags {
		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					attrPK: &types.AttributeValueMemberS{Value: reversePK},
					attrSK: &types.AttributeValueMemberS{Value: tag},
				},
			},
		})
	}

	return writeRequests
}

// executeBatchWriteChunks writes requests in chunks of batchWriteMaxItems,
// retrying unprocessed items.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes writeRequests ([]types.WriteRequest) which are the requests to write.
// Takes errPrefix (string) which is used to format error messages.
//
// Returns error when a batch write or retry fails.
func executeBatchWriteChunks(ctx context.Context, client *dynamodb.Client, tableName string, writeRequests []types.WriteRequest, errPrefix string) error {
	for i := 0; i < len(writeRequests); i += batchWriteMaxItems {
		end := min(i+batchWriteMaxItems, len(writeRequests))
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableName: writeRequests[i:end],
			},
		}
		output, err := client.BatchWriteItem(ctx, input)
		if err != nil {
			return fmt.Errorf("%s: %w", errPrefix, err)
		}
		if err := retryUnprocessedWrites(ctx, client, output.UnprocessedItems); err != nil {
			return fmt.Errorf("%s (unprocessed items): %w", errPrefix, err)
		}
	}
	return nil
}

// performTagInvalidation removes all keys associated with the given tags,
// then removes the tag and reverse mappings.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes namespace (string) which is the namespace prefix.
// Takes tags ([]string) which specifies the tags whose keys should be removed.
//
// Returns int which is the number of keys that were invalidated.
// Returns error when the operation fails.
func performTagInvalidation(ctx context.Context, client *dynamodb.Client, tableName string, namespace string, tags []string) (int, error) {
	if len(tags) == 0 {
		return 0, nil
	}

	keys, err := getKeysByTags(ctx, client, tableName, namespace, tags)
	if err != nil {
		return 0, fmt.Errorf("fetching keys by tags: %w", err)
	}
	if len(keys) == 0 {
		return 0, nil
	}

	allTagsByKey, err := collectAllTagsForKeys(ctx, client, tableName, namespace, keys)
	if err != nil {
		return 0, fmt.Errorf("collecting all tags for matched keys: %w", err)
	}

	writeRequests := buildTagInvalidationRequests(namespace, allTagsByKey, keys)

	if err := executeBatchWriteChunks(ctx, client, tableName, writeRequests,
		"failed to execute tag invalidation batch write"); err != nil {
		return 0, err
	}

	return len(keys), nil
}

// collectAllTagsForKeys queries the reverse mapping for each key to discover
// every tag attached to it.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes tableName (string) which identifies the table.
// Takes namespace (string) which is the namespace prefix.
// Takes keys ([]string) which are the cache keys to look up.
//
// Returns map[string][]string which maps each key to its associated tags.
// Returns error when a reverse mapping query fails.
func collectAllTagsForKeys(ctx context.Context, client *dynamodb.Client, tableName string, namespace string, keys []string) (map[string][]string, error) {
	result := make(map[string][]string, len(keys))
	for _, key := range keys {
		reversePK := namespace + keyTagsPrefix + key
		tags, err := queryTagsForKey(ctx, client, tableName, reversePK)
		if err != nil {
			return nil, fmt.Errorf("collecting tags for key %q: %w", key, err)
		}
		result[key] = tags
	}
	return result, nil
}

// buildTagInvalidationRequests creates delete requests for data items, all
// forward tag mappings, and all reverse mappings for the matched keys.
//
// Takes namespace (string) which is the namespace prefix.
// Takes allTagsByKey (map[string][]string) which maps keys to their tags.
// Takes keys ([]string) which are the keys to invalidate.
//
// Returns []types.WriteRequest which contains all the delete requests.
func buildTagInvalidationRequests(namespace string, allTagsByKey map[string][]string, keys []string) []types.WriteRequest {
	var writeRequests []types.WriteRequest

	for _, key := range keys {
		writeRequests = append(writeRequests, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: map[string]types.AttributeValue{
					attrPK: &types.AttributeValueMemberS{Value: key},
					attrSK: &types.AttributeValueMemberS{Value: skData},
				},
			},
		})
	}

	for _, key := range keys {
		for _, tag := range allTagsByKey[key] {
			writeRequests = append(writeRequests,
				types.WriteRequest{
					DeleteRequest: &types.DeleteRequest{
						Key: map[string]types.AttributeValue{
							attrPK: &types.AttributeValueMemberS{Value: namespace + tagPrefix + tag},
							attrSK: &types.AttributeValueMemberS{Value: key},
						},
					},
				},
				types.WriteRequest{
					DeleteRequest: &types.DeleteRequest{
						Key: map[string]types.AttributeValue{
							attrPK: &types.AttributeValueMemberS{Value: namespace + keyTagsPrefix + key},
							attrSK: &types.AttributeValueMemberS{Value: tag},
						},
					},
				},
			)
		}
	}

	return writeRequests
}

// retryUnprocessedWrites retries any unprocessed items from a BatchWriteItem
// call. It uses a simple retry loop until all items are processed or the
// context is cancelled.
//
// Takes client (*dynamodb.Client) which provides the DynamoDB connection.
// Takes unprocessed (map[string][]types.WriteRequest) which contains the items
// that were not processed.
//
// Returns error when the retry exhausts the context.
func retryUnprocessedWrites(ctx context.Context, client *dynamodb.Client, unprocessed map[string][]types.WriteRequest) error {
	for len(unprocessed) > 0 {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		output, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: unprocessed,
		})
		if err != nil {
			return fmt.Errorf("batch write retry failed: %w", err)
		}
		unprocessed = output.UnprocessedItems
	}
	return nil
}
