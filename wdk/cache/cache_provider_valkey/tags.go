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

package cache_provider_valkey

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
)

const (
	// tagPrefix is the key prefix for tag sets in Valkey.
	tagPrefix = "tag:"

	// keyTagsPrefix is the Valkey key prefix for storing a key's associated tags.
	keyTagsPrefix = "keytags:"
)

// addTagsToKey associates a key with a set of tags in Valkey using Sets. It
// creates both forward (tag -> keys) and reverse (key -> tags) mappings using
// DoMulti for efficiency.
//
// Takes client (valkey.Client) which provides the Valkey connection.
// Takes key (string) which is the cache key to associate with tags.
// Takes tags ([]string) which contains the tag names to associate with the key.
//
// Returns error when the pipeline execution fails.
func addTagsToKey(ctx context.Context, client valkey.Client, namespace string, key string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	cmds := make(valkey.Commands, 0, len(tags)+1)

	for _, tag := range tags {
		tagKey := namespace + tagPrefix + tag
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

// getKeysByTags retrieves all unique keys associated with the given tags.
// It uses SUNION for an efficient union operation on the server side.
//
// Takes client (valkey.Client) which provides the Valkey connection.
// Takes tags ([]string) which specifies the tags to look up.
//
// Returns []string which contains the unique keys associated with the tags.
// Returns error when the Valkey union operation fails.
func getKeysByTags(ctx context.Context, client valkey.Client, namespace string, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = namespace + tagPrefix + tag
	}

	keys, err := client.Do(ctx, client.B().Sunion().Key(tagKeys...).Build()).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys by tags: %w", err)
	}

	return keys, nil
}

// performTagInvalidation removes all keys associated with the given tags and
// the tags themselves. This is an atomic operation within Valkey using
// DoMulti.
//
// Takes client (valkey.Client) which provides the Valkey connection.
// Takes tags ([]string) which specifies the tags whose keys should be removed.
//
// Returns int which is the number of keys that were invalidated.
// Returns error when the pipeline execution fails.
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

	cmds := make(valkey.Commands, 0, len(keys)+len(tags)+1)

	cmds = append(cmds, client.B().Del().Key(keys...).Build())

	for _, key := range keys {
		keyTagsKey := keyTagsPrefix + key
		cmds = append(cmds, client.B().Del().Key(keyTagsKey).Build())
	}

	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = namespace + tagPrefix + tag
	}
	cmds = append(cmds, client.B().Del().Key(tagKeys...).Build())

	for _, response := range client.DoMulti(ctx, cmds...) {
		if err := response.Error(); err != nil {
			return 0, fmt.Errorf("failed to execute invalidation pipeline: %w", err)
		}
	}

	return len(keys), nil
}

// removeKeyFromTags removes a key from all its associated tag sets.
// This prevents memory leaks when keys are deleted directly via Invalidate().
//
// Takes client (valkey.Client) which provides the Valkey connection.
// Takes key (string) which is the cache key to remove from tag sets.
//
// Returns error when fetching tags or removing the key from tag sets fails.
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
		tagKey := namespace + tagPrefix + tag
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
