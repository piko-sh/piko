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
	"piko.sh/safeconv"
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

// primaryNodes finds nodes to treat as primaries, keyed by address.
//
// A replica holds a copy of its primary's keys, so anything that walks or counts the
// keyspace node by node must skip replicas or it sees every key once per copy. A node
// whose role cannot be read is kept, since missing its keys would be worse than seeing
// some twice. Callers that yield keys also drop repeats.
//
// Returns map[string]valkey.Client which holds known primaries and nodes whose role could
// not be read, keyed by address.
func (a *ValkeyClusterAdapter[K, V]) primaryNodes(ctx context.Context) map[string]valkey.Client {
	ctx, l := logger.From(ctx, log)
	roleCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster ROLE exceeded %s timeout", a.operationTimeout))
	defer cancel()

	nodes := a.clusterNodes()
	primaries := make(map[string]valkey.Client, len(nodes))
	for addr, nodeClient := range nodes {
		primary, err := nodeIsPrimary(roleCtx, nodeClient)
		if err != nil {
			l.Trace("Could not read the role of a cluster node; treating it as a primary",
				logger.String(logKeyNode, addr),
				logger.Error(err))
			primaries[addr] = nodeClient
			continue
		}
		if primary {
			primaries[addr] = nodeClient
		}
	}
	return primaries
}

// clusterNodes returns every node the client knows of, primaries and replicas alike.
//
// Returns map[string]valkey.Client which holds the nodes, keyed by address.
func (a *ValkeyClusterAdapter[K, V]) clusterNodes() map[string]valkey.Client {
	if a.listNodes != nil {
		return a.listNodes()
	}
	return a.client.Nodes()
}

// scanNodeKeys scans a single cluster node for all non-tag keys matching the given
// pattern. It handles cursor-based pagination internally.
//
// Takes nodeClient (valkey.Client) which is the node to scan.
// Takes addr (string) which is the node address for logging.
// Takes pattern (string) which is the SCAN match pattern.
//
// Returns []string which contains the discovered cache keys, excluding tag metadata keys.
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

// yieldScannedKeys decodes each scanned key string, retrieves its value, and yields the
// key-value pair to the iterator consumer.
//
// Takes keys ([]string) which contains the raw Valkey key strings to process.
// Takes yield (func(K, V) bool) which is the iterator callback.
//
// Returns bool which is false if the consumer stopped iteration early, or true if all
// keys were yielded successfully.
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

// All returns an iterator over all key-value pairs in the cache namespace.
//
// In cluster mode, this scans known primaries and nodes whose role could not be read.
// Each key is yielded once.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found during the scan.
func (a *ValkeyClusterAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()
		scanPattern := a.buildScanPattern()
		seen := make(map[string]struct{})

		for addr, nodeClient := range a.primaryNodes(ctx) {
			keys := dropSeenKeys(a.scanNodeKeys(ctx, nodeClient, addr, scanPattern), seen)
			if !a.yieldScannedKeys(ctx, keys, yield) {
				return
			}
		}
	}
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found across all cluster nodes.
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
// Returns iter.Seq[V] which yields each value found across all cluster nodes.
func (a *ValkeyClusterAdapter[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range a.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// GetEntry retrieves the full entry metadata for a key including TTL information.
//
// A cancelled or expired context returns its error before work begins.
//
// Takes key (K) which identifies the cache entry to retrieve.
//
// Returns cache.Entry[K, V] which contains the entry metadata and value.
// Returns bool which indicates whether the key was found.
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) GetEntry(ctx context.Context, key K) (cache.Entry[K, V], bool, error) {
	return a.ProbeEntry(ctx, key)
}

// ProbeEntry retrieves entry metadata without affecting access patterns or TTL.
//
// A cancelled or expired context returns its error before work begins.
//
// Takes key (K) which identifies the cache entry to probe.
//
// Returns cache.Entry[K, V] which contains the entry metadata and value.
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
		a.client.B().Pttl().Key(keyString).Build(),
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

	ttlMilliseconds, err := results[1].AsInt64()
	if err != nil {
		return cache.Entry[K, V]{}, false, fmt.Errorf("valkey cluster PTTL failed for key %s: %w", keyString, err)
	}

	now := time.Now()
	entry := cache.Entry[K, V]{
		Key:               key,
		Value:             value,
		Weight:            0,
		ExpiresAtNano:     expiryFromPTTL(now, ttlMilliseconds),
		RefreshableAtNano: 0,
		SnapshotAtNano:    now.UnixNano(),
	}

	return entry, true, nil
}

// EstimatedSize returns the approximate total number of keys in the cluster.
//
// It counts known primaries and nodes whose role could not be read. Unknown-role replicas
// may make the count too high. Failed node queries may leave it incomplete.
//
// Returns int which is the sum of keys counted on the selected nodes.
func (a *ValkeyClusterAdapter[K, V]) EstimatedSize() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster EstimatedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	primaries := a.primaryNodes(ctx)
	if a.namespace == "" {
		totalSize := 0
		for addr, nodeClient := range primaries {
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
	for addr, nodeClient := range primaries {
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

// Stats combines statistics from known primaries and nodes whose role could not be read.
//
// Unknown-role replicas may make the totals too high. Nodes that fail to respond are
// skipped.
//
// Returns cache.Stats which contains the combined hit and miss counts.
func (a *ValkeyClusterAdapter[K, V]) Stats() cache.Stats {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster Stats exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	var totalHits, totalMisses uint64

	for addr, nodeClient := range a.primaryNodes(ctx) {
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
		Hits:             totalHits,
		Misses:           totalMisses,
		Evictions:        0,
		LoadSuccessCount: 0,
		LoadFailureCount: 0,
		TotalLoadTime:    0,
	}
}

// Close releases this namespace.
//
// The Valkey Cluster client is shared by every namespace of the provider and is closed by
// ValkeyClusterProvider.Close, so closing one namespace leaves the others usable.
//
// Returns error when ctx is already done.
func (*ValkeyClusterAdapter[K, V]) Close(ctx context.Context) error {
	return ctx.Err()
}

// SetExpiresAfter updates the time to live for an existing key using the PEXPIRE command,
// or PERSIST for cache.NoExpiry. It never creates a key.
//
// A cancelled or expired context returns its error before work begins.
//
// Takes key (K) which identifies the cache entry to update.
// Takes expiresAfter (time.Duration) which specifies the new time-to-live. cache.NoExpiry
// removes the TTL. Other nonpositive values do nothing.
//
// Returns error when the operation fails.
func (a *ValkeyClusterAdapter[K, V]) SetExpiresAfter(ctx context.Context, key K, expiresAfter time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if expiresAfter <= 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.operationTimeout, fmt.Errorf("valkey cluster SetExpiresAfter exceeded %s timeout", a.operationTimeout))
	defer cancel()

	keyString, err := a.encodeKey(key)
	if err != nil {
		return fmt.Errorf(errFmtEncodeKey, err)
	}

	if expiresAfter == cache.NoExpiry {
		if err := a.client.Do(timeoutCtx, a.client.B().Persist().Key(keyString).Build()).Error(); err != nil {
			return fmt.Errorf("valkey cluster PERSIST failed for key %s: %w", keyString, err)
		}
		return nil
	}

	milliseconds := max(expiresAfter, time.Millisecond).Milliseconds()
	if err := a.client.Do(timeoutCtx, a.client.B().Pexpire().Key(keyString).Milliseconds(milliseconds).Build()).Error(); err != nil {
		return fmt.Errorf("valkey cluster PEXPIRE failed for key %s: %w", keyString, err)
	}

	return nil
}

// GetMaximum returns the maxmemory setting from one of the cluster nodes.
//
// Cluster nodes should have the same maxmemory setting.
//
// Returns uint64 which is the maxmemory value in bytes, or zero if none is readable.
func (a *ValkeyClusterAdapter[K, V]) GetMaximum() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster GetMaximum exceeded %s timeout", a.operationTimeout))
	defer cancel()

	for _, nodeClient := range a.clusterNodes() {
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

// SetMaximum is not supported by the Valkey Cluster provider as it is a server-level
// configuration.
func (*ValkeyClusterAdapter[K, V]) SetMaximum(_ uint64) {
	_, l := logger.From(context.Background(), log)
	l.Warn("SetMaximum is not supported by the Valkey Cluster provider and will have no effect.")
}

// WeightedSize sums memory usage from known primaries and nodes whose role could not be
// read.
//
// Replicas with known roles are excluded. An unknown-role replica may make the sum too
// high, while a failed node query may leave it incomplete.
//
// Returns uint64 which is the sum of used_memory from responding selected nodes.
func (a *ValkeyClusterAdapter[K, V]) WeightedSize() uint64 {
	ctx, cancel := context.WithTimeoutCause(context.Background(), a.operationTimeout, fmt.Errorf("valkey cluster WeightedSize exceeded %s timeout", a.operationTimeout))
	defer cancel()
	_, l := logger.From(ctx, log)

	var totalUsed uint64
	for addr, nodeClient := range a.primaryNodes(ctx) {
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

// SetRefreshableAfter is a no-op as Valkey Cluster does not natively support refresh
// scheduling.
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

// expiryFromPTTL turns a PTTL reply into an absolute expiry in Unix nanoseconds.
//
// PTTL counts whole milliseconds, so a key with less than one millisecond left reports
// zero. That is read as one millisecond left, not as no expiry, because a key without a
// TTL reports a negative value. Treating zero as no expiry would make a transaction
// rollback remove the key's TTL.
//
// Takes now (time.Time) which is when the reply was read.
// Takes ttlMilliseconds (int64) which is the PTTL reply in milliseconds.
//
// Returns int64 which is the expiry, or zero when the key has no TTL.
func expiryFromPTTL(now time.Time, ttlMilliseconds int64) int64 {
	if ttlMilliseconds < 0 {
		return 0
	}
	return now.Add(max(time.Duration(ttlMilliseconds)*time.Millisecond, time.Millisecond)).UnixNano()
}

// nodeIsPrimary asks a node for its role.
//
// Takes nodeClient (valkey.Client) which is the node to ask.
//
// Returns bool which is true when the node is a primary.
// Returns error when the role cannot be read.
func nodeIsPrimary(ctx context.Context, nodeClient valkey.Client) (bool, error) {
	reply, err := nodeClient.Do(ctx, nodeClient.B().Role().Build()).ToArray()
	if err != nil {
		return false, fmt.Errorf("reading node role: %w", err)
	}
	if len(reply) == 0 {
		return false, errEmptyRoleReply
	}
	role, err := reply[0].ToString()
	if err != nil {
		return false, fmt.Errorf("reading node role name: %w", err)
	}
	return isPrimaryRole(role), nil
}

// isPrimaryRole reports whether a ROLE reply names a primary. Valkey and Redis both
// report a primary as "master"; "primary" is accepted too.
//
// Takes role (string) which is the first element of the ROLE reply.
//
// Returns bool which is true for a primary.
func isPrimaryRole(role string) bool {
	return strings.EqualFold(role, "master") || strings.EqualFold(role, "primary")
}

// dropSeenKeys removes keys already seen on another node and records the rest as seen, so
// a key held by two nodes (a replica that could not be told apart, or a slot being moved)
// is yielded once.
//
// Takes keys ([]string) which are the keys scanned from one node.
// Takes seen (map[string]struct{}) which holds the keys already yielded.
//
// Returns []string which holds the keys not seen before, in their scan order.
func dropSeenKeys(keys []string, seen map[string]struct{}) []string {
	fresh := keys[:0]
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		fresh = append(fresh, key)
	}
	return fresh
}
