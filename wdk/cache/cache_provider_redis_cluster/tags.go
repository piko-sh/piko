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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/redis/go-redis/v9"
)

const (
	// tagPrefix is the key prefix for tag metadata in Redis Cluster.
	tagPrefix = "tag:"

	// keyTagsPrefix is the Redis key prefix for storing the set of tags linked to
	// a cache key.
	keyTagsPrefix = "keytags:"
)

// clusterHashTag wraps a tag name in hash tag braces to ensure same-slot
// hashing in Redis Cluster. Only the content within {...} is used for slot
// calculation, which allows multi-key operations like SUNION to work within
// a tag's scope.
//
// Takes tag (string) which is the tag name to wrap in hash braces.
//
// Returns string which is the tag wrapped in braces, e.g. "{tagname}".
func clusterHashTag(tag string) string {
	return fmt.Sprintf("{%s}", tag)
}

// addTagsToKey links a cache key to a set of tags in Redis Cluster using sets.
// It creates both forward (tag to keys) and reverse (key to tags) mappings
// using a pipeline for efficiency.
//
// Tag operations use hash tags to ensure same-slot placement. The reverse index
// (key to tags) is stored per-key and may be on different nodes.
//
// Takes client (*redis.ClusterClient) which provides the Redis cluster connection.
// Takes key (string) which is the cache key to link with tags.
// Takes tags ([]string) which contains the tag names to link with the key.
//
// Returns error when the pipeline execution fails.
func addTagsToKey(ctx context.Context, client *redis.ClusterClient, namespace string, key string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	pipe := client.Pipeline()

	for _, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		pipe.SAdd(ctx, tagKey, key)
	}

	keyTagsKey := keyTagsPrefix + key
	pipe.SAdd(ctx, keyTagsKey, tags)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add tags for key '%s': %w", key, err)
	}
	return nil
}

// getKeysByTags retrieves all unique keys associated with the given tags.
//
// Takes client (*redis.ClusterClient) which provides the Redis cluster connection.
// Takes tags ([]string) which specifies the tags to look up.
//
// Returns []string which contains the unique keys associated with the tags.
// Returns error when the pipeline execution fails entirely.
//
// Uses individual SMEMBERS per tag key via a pipeline, then deduplicates
// the results in Go. This avoids SUNION which would require all tag keys to
// be on the same hash slot.
func getKeysByTags(ctx context.Context, client *redis.ClusterClient, namespace string, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	pipe := client.Pipeline()
	memberCommands := make([]*redis.StringSliceCmd, len(tags))
	for i, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		memberCommands[i] = pipe.SMembers(ctx, tagKey)
	}
	_, _ = pipe.Exec(ctx)

	seen := make(map[string]struct{})
	for _, command := range memberCommands {
		members, err := command.Result()
		if err != nil {
			continue
		}
		for _, m := range members {
			seen[m] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(seen)), nil
}

// performTagInvalidation removes all keys linked to the given tags and the
// tags themselves. This is an atomic operation within Redis Cluster using a
// pipeline.
//
// Takes client (*redis.ClusterClient) which provides the cluster connection.
// Takes tags ([]string) which specifies the tags whose keys should be removed.
//
// Returns int which is the number of keys that were removed.
// Returns error when the pipeline execution fails.
//
// Cluster note: This operation deletes keys that may be spread across different
// cluster nodes. The pipeline routes each DEL command to the correct node. Tag
// set deletions go to the same node because of hash tags.
func performTagInvalidation(ctx context.Context, client *redis.ClusterClient, namespace string, tags []string) (int, error) {
	if len(tags) == 0 {
		return 0, nil
	}

	keys, err := getKeysByTags(ctx, client, namespace, tags)
	if err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}

	pipe := client.Pipeline()

	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	for _, key := range keys {
		keyTagsKey := keyTagsPrefix + key
		pipe.Del(ctx, keyTagsKey)
	}

	for _, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		pipe.Del(ctx, tagKey)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to execute invalidation pipeline: %w", err)
	}

	return len(keys), nil
}

// removeKeyFromTags removes a key from all its linked tag sets.
// This prevents memory leaks when keys are deleted directly via Invalidate.
//
// Takes client (*redis.ClusterClient) which provides the Redis connection.
// Takes key (string) which is the cache key to remove from tag sets.
//
// Returns error when getting tags fails or the pipeline fails to run.
func removeKeyFromTags(ctx context.Context, client *redis.ClusterClient, namespace string, key string) error {
	keyTagsKey := keyTagsPrefix + key
	tags, err := client.SMembers(ctx, keyTagsKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return fmt.Errorf("failed to get tags for key '%s': %w", key, err)
	}

	if len(tags) == 0 {
		return nil
	}

	pipe := client.Pipeline()
	for _, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		pipe.SRem(ctx, tagKey, key)
	}
	pipe.Del(ctx, keyTagsKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove key '%s' from tag sets: %w", key, err)
	}

	return nil
}
