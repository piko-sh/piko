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

package coordinator_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

// This file sets up all observability (logging, metrics, tracing) for the
// coordinator domain.

var (
	log = logger_domain.GetLogger("piko/internal/coordinator/coordinator_domain")

	// meter is the OpenTelemetry metric.Meter for the coordinator domain.
	// All coordinator metrics are registered against this meter.
	meter = otel.Meter("piko/internal/coordinator/coordinator_domain")

	// buildCount counts the total number of build operations started due to
	// cache misses.
	buildCount metric.Int64Counter

	// buildDuration measures the time taken for a full, uncached build operation,
	// in milliseconds.
	buildDuration metric.Float64Histogram

	// cacheHitCount counts the number of times a build was avoided due to a Tier 2
	// (annotation) cache hit.
	cacheHitCount metric.Int64Counter

	// cacheMissCount counts the number of times a build was required because Tier
	// 2 (annotation) cache missed.
	cacheMissCount metric.Int64Counter

	// cacheErrorCount counts errors from the cache adapter itself (e.g., disk I/O
	// errors).
	cacheErrorCount metric.Int64Counter

	// inputHashDuration measures the time taken to calculate the full input hash
	// (Tier 2) for a build.
	inputHashDuration metric.Float64Histogram

	// introspectionCacheHitCount counts the number of times the Tier 1
	// (introspection) cache was hit. This indicates a template-only change where
	// we can skip expensive type introspection.
	introspectionCacheHitCount metric.Int64Counter

	// introspectionCacheMissCount counts the number of times the Tier 1
	// (introspection) cache was missed. This indicates a script block or .go file
	// changed, requiring full type introspection.
	introspectionCacheMissCount metric.Int64Counter

	// introspectionHashDuration measures the time taken to calculate the
	// introspection hash (Tier 1).
	introspectionHashDuration metric.Float64Histogram

	// fastPathBuildCount counts builds with a Tier 1 cache hit (fast path).
	// These builds skip Phase 1 (type introspection) and only run
	// Phase 2 (annotation).
	fastPathBuildCount metric.Int64Counter

	// slowPathBuildCount counts the number of builds that used the slow path (both
	// caches missed). These builds run full Phase 1 + Phase 2.
	slowPathBuildCount metric.Int64Counter

	// partialBuildDuration measures the time taken for a fast path build (Phase 2
	// only), in milliseconds.
	partialBuildDuration metric.Float64Histogram
)

func init() {
	var err error

	buildCount, err = meter.Int64Counter(
		"piko.coordinator.build.count",
		metric.WithDescription("Total number of full project annotation builds initiated."),
		metric.WithUnit("{build}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	buildDuration, err = meter.Float64Histogram(
		"piko.coordinator.build.duration",
		metric.WithDescription("The duration of a single, uncached project annotation build."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheHitCount, err = meter.Int64Counter(
		"piko.coordinator.cache.hits",
		metric.WithDescription("Total number of times a build result was served from the cache."),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheMissCount, err = meter.Int64Counter(
		"piko.coordinator.cache.misses",
		metric.WithDescription("Total number of times a build was required due to a cache miss."),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheErrorCount, err = meter.Int64Counter(
		"piko.coordinator.cache.errors",
		metric.WithDescription("Total number of errors returned from the cache storage adapter."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	inputHashDuration, err = meter.Float64Histogram(
		"piko.coordinator.input_hash.duration",
		metric.WithDescription("The duration of calculating the full input hash (Tier 2) for a build."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	introspectionCacheHitCount, err = meter.Int64Counter(
		"piko.coordinator.cache.introspection.hits",
		metric.WithDescription("Total number of times Tier 1 (introspection) cache was hit, enabling fast path builds."),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	introspectionCacheMissCount, err = meter.Int64Counter(
		"piko.coordinator.cache.introspection.misses",
		metric.WithDescription("Total number of times Tier 1 (introspection) cache was missed, requiring full introspection."),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	introspectionHashDuration, err = meter.Float64Histogram(
		"piko.coordinator.introspection_hash.duration",
		metric.WithDescription("The duration of calculating the introspection hash (Tier 1, script blocks + .go files only)."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fastPathBuildCount, err = meter.Int64Counter(
		"piko.coordinator.build.fast_path.count",
		metric.WithDescription("Total number of fast path builds (Tier 1 cache hit, only Phase 2 annotation runs)."),
		metric.WithUnit("{build}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	slowPathBuildCount, err = meter.Int64Counter(
		"piko.coordinator.build.slow_path.count",
		metric.WithDescription("Total number of slow path builds (both caches missed, full Phase 1 + Phase 2)."),
		metric.WithUnit("{build}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	partialBuildDuration, err = meter.Float64Histogram(
		"piko.coordinator.build.partial.duration",
		metric.WithDescription("The duration of a fast path build (Phase 2 only, no type introspection)."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
