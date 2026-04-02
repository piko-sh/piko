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

package ast_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/ast/ast_adapters")

	// meter is the OpenTelemetry meter for the ast_adapters package.
	meter = otel.Meter("piko/internal/ast/ast_adapters")

	l2CacheMetrics cacheMetrics
)

// cacheMetrics holds the OpenTelemetry metrics for the cache layer.
type cacheMetrics struct {
	// latency records L2 cache operation times in milliseconds.
	latency metric.Float64Histogram

	// hits tracks the total number of cache hits.
	hits metric.Int64Counter

	// misses counts the total number of cache misses.
	misses metric.Int64Counter

	// sets counts the total number of cache set operations.
	sets metric.Int64Counter

	// deletes counts the total number of delete operations on the cache.
	deletes metric.Int64Counter

	// evictions counts items removed from the L2 cache when their TTL expires.
	evictions metric.Int64Counter

	// errors counts failed cache operations.
	errors metric.Int64Counter

	// serviceLatency records the latency of cache service operations
	// in milliseconds.
	serviceLatency metric.Float64Histogram

	// serviceHits counts cache hits from the overall cache service.
	serviceHits metric.Int64Counter

	// serviceMisses counts cache misses from the cache service.
	serviceMisses metric.Int64Counter

	// l1Hits counts the number of cache hits from the L1 in-memory cache.
	l1Hits metric.Int64Counter

	// l1Misses counts cache misses from the L1 in-memory cache.
	l1Misses metric.Int64Counter
}

func init() {
	var err error

	l2CacheMetrics.latency, err = meter.Float64Histogram(
		"piko.ast.cache.l2.latency",
		metric.WithDescription("The latency of L2 cache operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.hits, err = meter.Int64Counter(
		"piko.ast.cache.l2.hits",
		metric.WithDescription("Total number of cache hits from the L2 cache."),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.misses, err = meter.Int64Counter(
		"piko.ast.cache.l2.misses",
		metric.WithDescription("Total number of cache misses from the L2 cache."),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.sets, err = meter.Int64Counter(
		"piko.ast.cache.l2.sets",
		metric.WithDescription("Total number of set operations on the L2 cache."),
		metric.WithUnit("{set}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.deletes, err = meter.Int64Counter(
		"piko.ast.cache.l2.deletes",
		metric.WithDescription("Total number of delete operations on the L2 cache."),
		metric.WithUnit("{delete}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.evictions, err = meter.Int64Counter(
		"piko.ast.cache.l2.evictions",
		metric.WithDescription("Total number of items evicted from the L2 cache due to TTL expiration."),
		metric.WithUnit("{eviction}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.errors, err = meter.Int64Counter(
		"piko.ast.cache.l2.errors",
		metric.WithDescription("Total number of errors encountered by the L2 cache, such as file corruption."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.serviceLatency, err = meter.Float64Histogram(
		"piko.ast.cache.service.latency",
		metric.WithDescription("The latency of overall cache service operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.serviceHits, err = meter.Int64Counter(
		"piko.ast.cache.service.hits",
		metric.WithDescription("Total number of cache hits from the overall cache service."),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.serviceMisses, err = meter.Int64Counter(
		"piko.ast.cache.service.misses",
		metric.WithDescription("Total number of cache misses from the overall cache service."),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.l1Hits, err = meter.Int64Counter(
		"piko.ast.cache.l1.hits",
		metric.WithDescription("Total number of cache hits from the L1 (in-memory) cache."),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	l2CacheMetrics.l1Misses, err = meter.Int64Counter(
		"piko.ast.cache.l1.misses",
		metric.WithDescription("Total number of cache misses from the L1 (in-memory) cache, triggering an L2 lookup."),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
