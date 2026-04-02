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
	"iter"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

// buildScanPattern returns the SCAN pattern for the configured namespace.
//
// Returns string which is the pattern for Redis SCAN commands.
func (a *RedisClusterAdapter[K, V]) buildScanPattern() string {
	if a.namespace != "" {
		return a.namespace + "*"
	}
	return "*"
}

// collectAllKeys scans all master nodes and returns the combined list of
// non-tag cache keys matching the configured namespace.
//
// Returns []string which contains the discovered cache keys, excluding tag
// metadata keys.
//
// Safe for concurrent use. ForEachMaster runs callbacks concurrently, so keys are
// collected per-node then merged under a mutex.
func (a *RedisClusterAdapter[K, V]) collectAllKeys(ctx context.Context) []string {
	scanPattern := a.buildScanPattern()

	var mu sync.Mutex
	var allKeys []string

	_ = a.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		var nodeKeys []string
		scanIterator := client.Scan(ctx, 0, scanPattern, scanBatchSize).Iterator()
		for scanIterator.Next(ctx) {
			keyString := scanIterator.Val()
			if !isTagMetadataKey(keyString) {
				nodeKeys = append(nodeKeys, keyString)
			}
		}

		mu.Lock()
		allKeys = append(allKeys, nodeKeys...)
		mu.Unlock()

		return nil
	})

	return allKeys
}

// yieldScannedKeys decodes each scanned key string, retrieves its value, and
// yields the key-value pair to the iterator consumer.
//
// Takes keys ([]string) which contains the raw Redis key strings to process.
// Takes yield (func(K, V) bool) which is the iterator callback.
//
// Returns bool which is false if the consumer stopped iteration early, or true
// if all keys were yielded successfully.
func (a *RedisClusterAdapter[K, V]) yieldScannedKeys(ctx context.Context, keys []string, yield func(K, V) bool) bool {
	_, l := logger.From(ctx, log)

	for _, keyString := range keys {
		key, err := a.decodeKey(keyString)
		if err != nil {
			l.Trace("Failed to decode key during iteration",
				logger.String(logKeyField, keyString), logger.Error(err))
			continue
		}

		value, ok, getErr := a.GetIfPresent(ctx, key)
		if getErr != nil {
			l.Trace("Failed to get value during iteration",
				logger.String(logKeyField, keyString), logger.Error(getErr))
			continue
		}
		if !ok {
			continue
		}

		if !yield(key, value) {
			return false
		}
	}
	return true
}

// All returns an iterator over all key-value pairs in the cache namespace,
// collecting keys from all master nodes via ForEachMaster before yielding
// them from the calling goroutine because ForEachMaster runs callbacks
// concurrently and yield must not be called from multiple goroutines.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found across all
// master nodes in the namespace.
func (a *RedisClusterAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()
		keys := a.collectAllKeys(ctx)
		a.yieldScannedKeys(ctx, keys, yield)
	}
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found in the namespace across
// all master nodes.
func (a *RedisClusterAdapter[K, V]) Keys() iter.Seq[K] {
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
// Returns iter.Seq[V] which yields each value found in the namespace across
// all master nodes.
func (a *RedisClusterAdapter[K, V]) Values() iter.Seq[V] {
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
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns Entry[K, V] which contains the value and metadata such
// as expiry time.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access patterns or TTL.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to probe.
//
// Returns Entry[K, V] which contains the value and metadata such
// as expiry time.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	if err := ctx.Err(); err != nil {
		return cache.Entry[K, V]{}, false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis cluster ProbeEntry exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf(errFmtEncodeKey, err)
	}

	pipe := a.client.Pipeline()
	getCmd := pipe.Get(timeoutCtx, keyString)
	ttlCmd := pipe.TTL(timeoutCtx, keyString)
	if _, err = pipe.Exec(timeoutCtx); err != nil {
		if errors.Is(err, redis.Nil) {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("redis pipeline failed: %w", err)
	}

	return a.buildProbeEntry(key, keyString, getCmd, ttlCmd)
}

// buildProbeEntry constructs a cache.Entry from the pipeline
// command results.
//
// Takes key (K) which is the typed cache key for the entry.
// Takes keyString (string) which is the encoded key string used
// for error messages.
// Takes getCmd (*redis.StringCmd) which holds the GET result
// from the pipeline.
// Takes ttlCmd (*redis.DurationCmd) which holds the TTL result
// from the pipeline.
//
// Returns cache.Entry[K, V] which contains the decoded value
// and expiry metadata.
// Returns bool which indicates whether the key was found.
// Returns error when decoding or unmarshalling fails.
func (a *RedisClusterAdapter[K, V]) buildProbeEntry(
	key K, keyString string, getCmd *redis.StringCmd, ttlCmd *redis.DurationCmd,
) (cache.Entry[K, V], bool, error) {
	valBytes, err := getCmd.Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("failed to get value bytes: %w", err)
	}

	var v V
	encoder, err := a.registry.GetByType(reflect.TypeOf(v))
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("failed to find encoder for type: %w", err)
	}

	unmarshalled, err := encoder.UnmarshalAny(valBytes)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	value, ok := unmarshalled.(V)
	if !ok {
		return cache.Entry[K, V]{}, false, fmt.Errorf("type assertion failed after unmarshal for key %q", keyString)
	}

	ttl, _ := ttlCmd.Result()

	var expiresAtNano int64
	if ttl > 0 {
		expiresAtNano = time.Now().Add(ttl).UnixNano()
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

// EstimatedSize returns the approximate total number of keys in the cluster.
//
// Returns int which is the sum of keys across all master nodes, or zero on
// error.
//
// CLUSTER NOTE: This sums DBSIZE across all master nodes.
func (a *RedisClusterAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis cluster EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	if a.namespace == "" {
		totalSize := 0
		err := a.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			size, err := client.DBSize(ctx).Result()
			if err != nil {
				return err
			}
			totalSize += int(size)
			return nil
		})

		if err != nil {
			l.Warn("Failed to get DBSize from Redis Cluster", logger.Error(err))
			return 0
		}

		return totalSize
	}

	var count int
	scanPattern := a.namespace + "*"
	err := a.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		scanIterator := client.Scan(ctx, 0, scanPattern, scanBatchSize).Iterator()
		for scanIterator.Next(ctx) {
			count++
		}
		return scanIterator.Err()
	})

	if err != nil {
		l.Warn("Failed to scan keys for EstimatedSize", logger.Error(err))
	}

	return count
}

// parseRedisStatsInfo extracts hits and misses from Redis INFO stats output.
//
// Takes info (string) which contains the raw Redis INFO stats response.
//
// Returns hits (int64) which is the keyspace_hits value from the stats.
// Returns misses (int64) which is the keyspace_misses value from the stats.
func (*RedisClusterAdapter[K, V]) parseRedisStatsInfo(ctx context.Context, info string) (hits, misses int64) {
	_, l := logger.From(ctx, log)

	for line := range strings.SplitSeq(info, "\r\n") {
		if strings.HasPrefix(line, "keyspace_hits:") {
			if _, err := fmt.Sscanf(line, "keyspace_hits:%d", &hits); err != nil {
				l.Trace("Failed to parse keyspace_hits from cluster node",
					logger.String("line", line), logger.Error(err))
			}
		}
		if strings.HasPrefix(line, "keyspace_misses:") {
			if _, err := fmt.Sscanf(line, "keyspace_misses:%d", &misses); err != nil {
				l.Trace("Failed to parse keyspace_misses from cluster node",
					logger.String("line", line), logger.Error(err))
			}
		}
	}
	return hits, misses
}

// Stats returns combined statistics from all cluster nodes.
//
// Returns cache.Stats which contains the total hit and miss counts across all
// master nodes. If any node fails to respond, returns an empty Stats struct.
func (a *RedisClusterAdapter[K, V]) Stats() cache.Stats {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis cluster Stats exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	var totalHits, totalMisses uint64

	err := a.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		info, err := client.Info(ctx, "stats").Result()
		if err != nil {
			return err
		}

		hits, misses := a.parseRedisStatsInfo(ctx, info)
		if hits > 0 {
			totalHits += safeconv.Int64ToUint64(hits)
		}
		if misses > 0 {
			totalMisses += safeconv.Int64ToUint64(misses)
		}
		return nil
	})

	if err != nil {
		l.Warn("Failed to get INFO from Redis Cluster", logger.Error(err))
		return cache.Stats{}
	}

	return cache.Stats{
		Hits:   totalHits,
		Misses: totalMisses,
	}
}

// Close releases the Redis Cluster client connection.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when the client cannot be closed cleanly.
func (a *RedisClusterAdapter[K, V]) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.client.Close(); err != nil {
		return fmt.Errorf("error closing Redis Cluster client: %w", err)
	}
	return nil
}

// SetExpiresAfter updates the time to live for an existing key using the Redis
// EXPIRE command.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new time to live.
//
// Returns error when the operation fails.
func (a *RedisClusterAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("redis cluster SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := a.client.PExpire(timeoutCtx, keyString, expiresAfter).Err(); err != nil {
		return fmt.Errorf("redis cluster PExpire failed: %w", err)
	}

	return nil
}

// GetMaximum returns the maxmemory setting from one of the cluster nodes.
//
// Returns uint64 which is the maxmemory value in bytes, or zero if not found.
//
// In a correctly configured cluster, all nodes should have the same maxmemory
// setting. This reads from any master node.
func (a *RedisClusterAdapter[K, V]) GetMaximum() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis cluster GetMaximum exceeded %s timeout", a.operationTimeout))
	defer cancel()

	var maxMemory uint64
	_ = a.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		configResult, err := client.ConfigGet(ctx, "maxmemory").Result()
		if err != nil {
			return err
		}
		if maxString, ok := configResult["maxmemory"]; ok {
			_, _ = fmt.Sscanf(maxString, "%d", &maxMemory)
		}
		return nil
	})

	return maxMemory
}

// SetMaximum is not supported by the Redis Cluster provider as it is a
// server-level configuration.
func (*RedisClusterAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the Redis Cluster provider and will have no effect.")
}

// WeightedSize returns the total memory usage across all cluster nodes.
//
// Returns uint64 which is the sum of used_memory from all master nodes, or
// zero if the cluster cannot be queried.
func (a *RedisClusterAdapter[K, V]) WeightedSize() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("redis cluster WeightedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	var totalUsed uint64
	err := a.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		_, innerL := logger.From(ctx, log)

		info, err := client.Info(ctx, "memory").Result()
		if err != nil {
			return err
		}

		for line := range strings.SplitSeq(info, "\r\n") {
			if strings.HasPrefix(line, "used_memory:") {
				var used uint64
				if _, err := fmt.Sscanf(line, "used_memory:%d", &used); err != nil {
					innerL.Trace("Failed to parse used_memory from cluster node", logger.String("line", line), logger.Error(err))
					break
				}
				totalUsed += used
				break
			}
		}
		return nil
	})

	if err != nil {
		l.Warn("Failed to get memory info from Redis Cluster", logger.Error(err))
		return 0
	}

	return totalUsed
}

// SetRefreshableAfter is a no-op as Redis Cluster does not natively support
// refresh scheduling.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error (always nil as this is a no-op).
func (*RedisClusterAdapter[K, V]) SetRefreshableAfter(ctx context.Context, _ K, _ time.Duration) error {
	_, l := logger.From(ctx, log)

	l.Internal("SetRefreshableAfter is not natively supported by the Redis Cluster provider.")
	return nil
}

// isTagMetadataKey reports whether the key is used for internal tag tracking.
//
// Takes keyString (string) which is the key to check.
//
// Returns bool which is true if the key has a tag metadata prefix.
func isTagMetadataKey(keyString string) bool {
	return strings.HasPrefix(keyString, tagPrefix) || strings.HasPrefix(keyString, keyTagsPrefix)
}
