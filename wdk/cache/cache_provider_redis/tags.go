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
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	// tagPrefix is the key prefix for tag sets in Redis.
	tagPrefix = "tag:"

	// keyTagsPrefix is the Redis key prefix for storing a key's associated tags.
	keyTagsPrefix = "keytags:"
)

// addTagsToKey associates a key with a set of tags in Redis using Sets. It
// creates both forward (tag -> keys) and reverse (key -> tags) mappings using a
// pipeline for efficiency.
//
// Takes client (*redis.Client) which provides the Redis connection.
// Takes key (string) which is the cache key to associate with tags.
// Takes tags ([]string) which contains the tag names to associate with the key.
//
// Returns error when the pipeline execution fails.
func addTagsToKey(ctx context.Context, client *redis.Client, namespace string, key string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	pipe := client.Pipeline()

	for _, tag := range tags {
		tagKey := namespace + tagPrefix + tag
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
// It uses SUNION for an efficient union operation on the server side.
//
// Takes client (*redis.Client) which provides the Redis connection.
// Takes tags ([]string) which specifies the tags to look up.
//
// Returns []string which contains the unique keys associated with the tags.
// Returns error when the Redis union operation fails.
func getKeysByTags(ctx context.Context, client *redis.Client, namespace string, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = namespace + tagPrefix + tag
	}

	keys, err := client.SUnion(ctx, tagKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys by tags: %w", err)
	}

	return keys, nil
}

// performTagInvalidation removes all keys associated with the given tags and
// the tags themselves. This is an atomic operation within Redis using a
// pipeline.
//
// Takes client (*redis.Client) which provides the Redis connection.
// Takes tags ([]string) which specifies the tags whose keys should be removed.
//
// Returns int which is the number of keys that were invalidated.
// Returns error when the pipeline execution fails.
func performTagInvalidation(ctx context.Context, client *redis.Client, namespace string, tags []string) (int, error) {
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

	pipe.Del(ctx, keys...)

	for _, key := range keys {
		keyTagsKey := keyTagsPrefix + key
		pipe.Del(ctx, keyTagsKey)
	}

	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = namespace + tagPrefix + tag
	}
	pipe.Del(ctx, tagKeys...)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to execute invalidation pipeline: %w", err)
	}

	return len(keys), nil
}

// removeKeyFromTags removes a key from all its associated tag sets.
// This prevents memory leaks when keys are deleted directly via Invalidate().
//
// Takes client (*redis.Client) which provides the Redis connection.
// Takes key (string) which is the cache key to remove from tag sets.
//
// Returns error when fetching tags or removing the key from tag sets fails.
func removeKeyFromTags(ctx context.Context, client *redis.Client, namespace string, key string) error {
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
		tagKey := namespace + tagPrefix + tag
		pipe.SRem(ctx, tagKey, key)
	}
	pipe.Del(ctx, keyTagsKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove key '%s' from tag sets: %w", key, err)
	}

	return nil
}
