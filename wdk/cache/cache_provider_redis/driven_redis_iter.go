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
	"iter"

	"piko.sh/piko/wdk/logger"
)

// All returns an iterator over all key-value pairs in the cache namespace.
//
// Returns iter.Seq2[K, V] which yields each key-value pair found in the namespace via
// Redis SCAN.
func (a *RedisAdapter[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ctx := context.Background()

		scanPattern := a.allScanPattern()
		scanIterator := a.client.Scan(ctx, 0, scanPattern, 100).Iterator()
		for scanIterator.Next(ctx) {
			if !a.yieldScannedEntry(ctx, scanIterator.Val(), yield) {
				return
			}
		}
	}
}

// allScanPattern returns the Redis SCAN pattern for iterating all keys in the adapter's
// namespace.
//
// Returns string which is the wildcard pattern scoped to the configured namespace.
func (a *RedisAdapter[K, V]) allScanPattern() string {
	if a.namespace != "" {
		return a.namespace + scanAllPattern
	}
	return scanAllPattern
}

// yieldScannedEntry decodes a single scanned key, fetches its value, and yields it to the
// iterator consumer.
//
// Takes keyString (string) which is the raw Redis key to decode and look up.
// Takes yield (func(K, V) bool) which is the iterator callback that receives the decoded
// key and its value.
//
// Returns bool which is false when the consumer stopped iteration early, or true if
// processing should continue.
func (a *RedisAdapter[K, V]) yieldScannedEntry(ctx context.Context, keyString string, yield func(K, V) bool) bool {
	_, l := logger.From(ctx, log)

	key, err := a.decodeKey(keyString)
	if err != nil {
		l.Trace("Failed to decode key during iteration",
			logger.String(logKeyField, keyString),
			logger.Error(err))
		return true
	}

	value, ok, getErr := a.GetIfPresent(ctx, key)
	if getErr != nil {
		l.Trace("Failed to get value during iteration",
			logger.String(logKeyField, keyString),
			logger.Error(getErr))
		return true
	}
	if ok {
		return yield(key, value)
	}
	return true
}

// Keys returns an iterator over all keys in the cache namespace.
//
// Returns iter.Seq[K] which yields each key found in the namespace.
func (a *RedisAdapter[K, V]) Keys() iter.Seq[K] {
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
func (a *RedisAdapter[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range a.All() {
			if !yield(v) {
				return
			}
		}
	}
}
