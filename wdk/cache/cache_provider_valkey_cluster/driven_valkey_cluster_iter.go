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
	"iter"
	"reflect"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

// buildScanPattern returns the SCAN pattern for the configured namespace.
//
// Returns string which is the pattern for Valkey SCAN commands.
func (a *ValkeyClusterAdapter[K, V]) buildScanPattern() string {
	if a.namespace != "" {
		return a.namespace + "*"
	}
	return "*"
}

// scanNodeKeys scans a single cluster node for all non-tag keys matching the
// given pattern. It handles cursor-based pagination internally.
//
// Takes nodeClient (valkey.Client) which is the node to scan.
// Takes addr (string) which is the node address for logging.
// Takes pattern (string) which is the SCAN match pattern.
//
// Returns []string which contains the discovered cache keys, excluding tag
// metadata keys.
func (*ValkeyClusterAdapter[K, V]) scanNodeKeys(ctx context.Context, nodeClient valkey.Client, addr string, pattern string) []string {
	ctx, l := logger.From(ctx, log)

	var keys []string
	var cursor uint64

	for {
		response := nodeClient.Do(ctx, nodeClient.B().Scan().Cursor(cursor).Match(pattern).Count(scanBatchSize).Build())
		scanEntry, err := response.AsScanEntry()
		if err != nil {
			l.Trace("SCAN failed on cluster node during iteration",
				logger.String(logKeyNode, addr),
				logger.Error(err))
			break
		}

		for _, keyString := range scanEntry.Elements {
			if !isTagMetadataKey(keyString) {
				keys = append(keys, keyString)
			}
		}

		cursor = scanEntry.Cursor
		if cursor == 0 {
			break
		}
	}

	return keys
}

// yieldScannedKeys decodes each scanned key string, retrieves its value, and
// yields the key-value pair to the iterator consumer.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes keys ([]string) which contains the raw Valkey key strings to process.
// Takes yield (func(K, V) bool) which is the iterator callback.
//
// Returns bool which is false if the consumer stopped iteration early, or true
// if all keys were yielded successfully.
func (a *ValkeyClusterAdapter[K, V]) yieldScannedKeys(ctx context.Context, keys []string, yield func(K, V) bool) bool {
	ctx, l := logger.From(ctx, log)

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

// All returns an iterator over all key-value pairs in the cache
// namespace. In cluster mode, this scans all nodes.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found
// across all cluster nodes.
func (a *ValkeyClusterAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()
		scanPattern := a.buildScanPattern()

		for addr, nodeClient := range a.client.Nodes() {
			keys := a.scanNodeKeys(ctx, nodeClient, addr, scanPattern)
			if !a.yieldScannedKeys(ctx, keys, yield) {
				return
			}
		}
	}
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found across all cluster
// nodes.
func (a *ValkeyClusterAdapter[K, V]) Keys() iter.Seq[K] {
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
// Returns iter.Seq[V] which yields each value found across all
// cluster nodes.
func (a *ValkeyClusterAdapter[K, V]) Values() iter.Seq[V] {
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
// Returns cache.Entry[K, V] which contains the entry metadata and
// value.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access
// patterns or TTL.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes key (K) which identifies the cache entry to probe.
//
// Returns cache.Entry[K, V] which contains the entry metadata and
// value.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) ProbeEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	if err := ctx.Err(); err != nil {
		return cache.Entry[K, V]{}, false, err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster ProbeEntry exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf(errFmtEncodeKey, err)
	}

	cmds := []valkey.Completed{
		a.client.B().Get().Key(keyString).Build(),
		a.client.B().Ttl().Key(keyString).Build(),
	}
	results := a.client.DoMulti(timeoutCtx, cmds...)

	valBytes, err := results[0].AsBytes()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return cache.Entry[K, V]{}, false, nil
		}
		return cache.Entry[K, V]{}, false, fmt.Errorf("valkey cluster get failed for key %s: %w", keyString, err)
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
		return cache.Entry[K, V]{}, false, fmt.Errorf("type assertion failed after unmarshal for key %s", keyString)
	}

	ttlSeconds, _ := results[1].AsInt64()

	var expiresAtNano int64
	if ttlSeconds > 0 {
		expiresAtNano = time.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixNano()
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
// Returns int which is the sum of keys across all nodes, or zero on error.
//
// CLUSTER NOTE: This sums DBSIZE across all nodes.
func (a *ValkeyClusterAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	if a.namespace == "" {
		totalSize := 0
		for addr, nodeClient := range a.client.Nodes() {
			size, err := nodeClient.Do(ctx, nodeClient.B().Dbsize().Build()).AsInt64()
			if err != nil {
				l.Warn("Failed to get DBSIZE from cluster node",
					logger.String(logKeyNode, addr),
					logger.Error(err))
				continue
			}
			totalSize += int(size)
		}
		return totalSize
	}

	var count int
	scanPattern := a.namespace + "*"
	for addr, nodeClient := range a.client.Nodes() {
		var cursor uint64
		for {
			response, err := nodeClient.Do(ctx, nodeClient.B().Scan().Cursor(cursor).Match(scanPattern).Count(scanBatchSize).Build()).AsScanEntry()
			if err != nil {
				l.Warn("Failed to scan keys for EstimatedSize",
					logger.String(logKeyNode, addr),
					logger.Error(err))
				break
			}
			count += len(response.Elements)
			cursor = response.Cursor
			if cursor == 0 {
				break
			}
		}
	}
	return count
}

// parseValkeyStatsInfo extracts hits and misses from Valkey INFO stats output.
//
// Takes info (string) which contains the raw Valkey INFO stats response.
//
// Returns hits (int64) which is the keyspace_hits value from the stats.
// Returns misses (int64) which is the keyspace_misses value from the stats.
func (*ValkeyClusterAdapter[K, V]) parseValkeyStatsInfo(ctx context.Context, info string) (hits, misses int64) {
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
// nodes. If any node fails to respond, its stats are skipped.
func (a *ValkeyClusterAdapter[K, V]) Stats() cache.Stats {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster Stats exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	var totalHits, totalMisses uint64

	for addr, nodeClient := range a.client.Nodes() {
		info, err := nodeClient.Do(ctx, nodeClient.B().Info().Section("stats").Build()).ToString()
		if err != nil {
			l.Warn("Failed to get INFO from cluster node",
				logger.String(logKeyNode, addr),
				logger.Error(err))
			continue
		}

		hits, misses := a.parseValkeyStatsInfo(ctx, info)
		if hits > 0 {
			totalHits += safeconv.Int64ToUint64(hits)
		}
		if misses > 0 {
			totalMisses += safeconv.Int64ToUint64(misses)
		}
	}

	return cache.Stats{
		Hits:   totalHits,
		Misses: totalMisses,
	}
}

// Close releases the Valkey Cluster client connection.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error when resources cannot be released cleanly.
func (a *ValkeyClusterAdapter[K, V]) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.client.Close()
	return nil
}

// SetExpiresAfter updates the time to live for an existing key using the
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
func (a *ValkeyClusterAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if err := a.client.Do(timeoutCtx, a.client.B().Pexpire().Key(keyString).Milliseconds(expiresAfter.Milliseconds()).Build()).Error(); err != nil {
		return fmt.Errorf("valkey cluster PEXPIRE failed for key %s: %w", keyString, err)
	}

	return nil
}

// GetMaximum returns the maxmemory setting from one of the cluster nodes.
//
// Returns uint64 which is the maxmemory value in bytes, or zero if not found.
//
// In a correctly configured cluster, all nodes should have the same maxmemory
// setting. This reads from any node.
func (a *ValkeyClusterAdapter[K, V]) GetMaximum() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster GetMaximum exceeded %s timeout", a.operationTimeout))
	defer cancel()

	for _, nodeClient := range a.client.Nodes() {
		response := nodeClient.Do(ctx, nodeClient.B().ConfigGet().Parameter("maxmemory").Build())
		result, err := response.AsStrMap()
		if err != nil {
			continue
		}
		if maxString, ok := result["maxmemory"]; ok {
			var maxMemory uint64
			if _, err := fmt.Sscanf(maxString, "%d", &maxMemory); err == nil {
				return maxMemory
			}
		}
	}

	return 0
}

// SetMaximum is not supported by the Valkey Cluster provider as it is a
// server-level configuration.
func (*ValkeyClusterAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the Valkey Cluster provider and will have no effect.")
}

// WeightedSize returns the total memory usage across all cluster nodes.
//
// Returns uint64 which is the sum of used_memory from all nodes, or
// zero if the cluster cannot be queried.
func (a *ValkeyClusterAdapter[K, V]) WeightedSize() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster WeightedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	var totalUsed uint64
	for addr, nodeClient := range a.client.Nodes() {
		info, err := nodeClient.Do(ctx, nodeClient.B().Info().Section("memory").Build()).ToString()
		if err != nil {
			l.Warn("Failed to get memory info from cluster node",
				logger.String(logKeyNode, addr),
				logger.Error(err))
			continue
		}

		for line := range strings.SplitSeq(info, "\r\n") {
			if strings.HasPrefix(line, "used_memory:") {
				var used uint64
				if _, err := fmt.Sscanf(line, "used_memory:%d", &used); err != nil {
					l.Trace("Failed to parse used_memory from cluster node",
						logger.String("line", line), logger.Error(err))
					break
				}
				totalUsed += used
				break
			}
		}
	}

	return totalUsed
}

// SetRefreshableAfter is a no-op as Valkey Cluster does not natively support
// refresh scheduling.
//
// Takes ctx (context.Context) for cancellation and timeout.
//
// Returns error which is always nil as this is a no-op.
func (*ValkeyClusterAdapter[K, V]) SetRefreshableAfter(ctx context.Context, _ K, _ time.Duration) error {
	_, l := logger.From(ctx, log)

	l.Internal("SetRefreshableAfter is not natively supported by the Valkey Cluster provider.")
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
