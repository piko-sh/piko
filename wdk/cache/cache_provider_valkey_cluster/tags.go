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

package cache_provider_valkey_cluster

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/valkey-io/valkey-go"
)

const (
	// tagPrefix is the key prefix for tag metadata in Valkey Cluster.
	tagPrefix = "tag:"

	// keyTagsPrefix is the Valkey key prefix for storing the set of tags linked to
	// a cache key.
	keyTagsPrefix = "keytags:"
)

// clusterHashTag wraps a tag name in hash tag braces to ensure same-slot
// hashing in Valkey Cluster. Only the content within {...} is used for slot
// calculation, which allows multi-key operations like SUNION to work within
// a tag's scope.
//
// Takes tag (string) which is the tag name to wrap in hash braces.
//
// Returns string which is the tag wrapped in braces, e.g. "{tagname}".
func clusterHashTag(tag string) string {
	return fmt.Sprintf("{%s}", tag)
}

// addTagsToKey links a cache key to a set of tags in Valkey Cluster using sets.
// It creates both forward (tag to keys) and reverse (key to tags) mappings
// using DoMulti for efficiency.
//
// Tag operations use hash tags to ensure same-slot placement. The reverse index
// (key to tags) is stored per-key and may be on different nodes.
//
// Takes client (valkey.Client) which provides the Valkey cluster connection.
// Takes key (string) which is the cache key to link with tags.
// Takes tags ([]string) which contains the tag names to link with the key.
//
// Returns error when the pipeline execution fails.
func addTagsToKey(ctx context.Context, client valkey.Client, namespace string, key string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	cmds := make(valkey.Commands, 0, len(tags)+1)

	for _, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		cmds = append(cmds, client.B().Sadd().Key(tagKey).Member(key).Build())
	}

	keyTagsKey := keyTagsPrefix + key
	cmds = append(cmds, client.B().Sadd().Key(keyTagsKey).Member(tags...).Build())

	for _, response := range client.DoMulti(ctx, cmds...) {
		if err := response.Error(); err != nil {
			return fmt.Errorf("failed to add tags for key '%s': %w", key, err)
		}
	}
	return nil
}

// getKeysByTags retrieves all unique keys associated with the given tags,
// using individual SMEMBERS per tag key via DoMulti and deduplicating in Go
// to avoid SUNION which would panic in cluster mode when tag keys hash to
// different slots.
//
// Takes client (valkey.Client) which provides the Valkey cluster connection.
// Takes tags ([]string) which specifies the tags to look up.
//
// Returns []string which contains the unique keys associated with the tags.
// Returns error when the pipeline execution fails entirely.
func getKeysByTags(ctx context.Context, client valkey.Client, namespace string, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	cmds := make(valkey.Commands, len(tags))
	for i, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		cmds[i] = client.B().Smembers().Key(tagKey).Build()
	}

	seen := make(map[string]struct{})
	for _, response := range client.DoMulti(ctx, cmds...) {
		members, err := response.AsStrSlice()
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
// tags themselves. Uses DoMulti for efficiency.
//
// Takes client (valkey.Client) which provides the cluster connection.
// Takes tags ([]string) which specifies the tags whose keys should be removed.
//
// Returns int which is the number of keys that were removed.
// Returns error when the pipeline execution fails.
//
// Cluster note: This operation deletes keys that may be spread across different
// cluster nodes. DoMulti routes each DEL command to the correct node. Tag
// set deletions go to the same node because of hash tags.
func performTagInvalidation(ctx context.Context, client valkey.Client, namespace string, tags []string) (int, error) {
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

	cmds := make(valkey.Commands, 0, len(keys)*2+len(tags))

	for _, key := range keys {
		cmds = append(cmds, client.B().Del().Key(key).Build())
	}

	for _, key := range keys {
		keyTagsKey := keyTagsPrefix + key
		cmds = append(cmds, client.B().Del().Key(keyTagsKey).Build())
	}

	for _, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		cmds = append(cmds, client.B().Del().Key(tagKey).Build())
	}

	for _, response := range client.DoMulti(ctx, cmds...) {
		if err := response.Error(); err != nil {
			return 0, fmt.Errorf("failed to execute invalidation pipeline: %w", err)
		}
	}

	return len(keys), nil
}

// removeKeyFromTags removes a key from all its linked tag sets.
// This prevents memory leaks when keys are deleted directly via Invalidate.
//
// Takes client (valkey.Client) which provides the Valkey connection.
// Takes key (string) which is the cache key to remove from tag sets.
//
// Returns error when getting tags fails or the pipeline fails to run.
func removeKeyFromTags(ctx context.Context, client valkey.Client, namespace string, key string) error {
	keyTagsKey := keyTagsPrefix + key
	tags, err := client.Do(ctx, client.B().Smembers().Key(keyTagsKey).Build()).AsStrSlice()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil
		}
		return fmt.Errorf("failed to get tags for key '%s': %w", key, err)
	}

	if len(tags) == 0 {
		return nil
	}

	cmds := make(valkey.Commands, 0, len(tags)+1)
	for _, tag := range tags {
		tagKey := tagPrefix + clusterHashTag(namespace+tag)
		cmds = append(cmds, client.B().Srem().Key(tagKey).Member(key).Build())
	}
	cmds = append(cmds, client.B().Del().Key(keyTagsKey).Build())

	for _, response := range client.DoMulti(ctx, cmds...) {
		if err := response.Error(); err != nil {
			return fmt.Errorf("failed to remove key '%s' from tag sets: %w", key, err)
		}
	}

	return nil
}
