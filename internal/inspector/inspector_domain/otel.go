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

package inspector_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the inspector_domain package.
	log = logger_domain.GetLogger("piko/internal/inspector/inspector_domain")

	// Meter is the OpenTelemetry meter instance for the inspector_domain package.
	Meter = otel.Meter("piko/internal/inspector/inspector_domain")

	// BuilderBuildCount is a counter metric that tracks the number of builds.
	BuilderBuildCount metric.Int64Counter

	// BuilderBuildDuration records the time taken to build documentation.
	BuilderBuildDuration metric.Float64Histogram

	// BuilderBuildErrorCount tracks the number of errors that happen during
	// builder builds.
	BuilderBuildErrorCount metric.Int64Counter

	// BuilderPackageLoadCount tracks how many packages the builder has loaded.
	BuilderPackageLoadCount metric.Int64Counter

	// BuilderPackageLoadDuration records how long it takes to load Go packages.
	BuilderPackageLoadDuration metric.Float64Histogram

	// BuilderPackageLoadErrorCount is a metric that tracks the number of package
	// load errors encountered during building.
	BuilderPackageLoadErrorCount metric.Int64Counter

	// BuilderCacheKeyGenCount counts the number of cache keys generated.
	BuilderCacheKeyGenCount metric.Int64Counter

	// BuilderCacheKeyGenDuration records the time taken to generate cache keys.
	BuilderCacheKeyGenDuration metric.Float64Histogram

	// BuilderCacheKeyGenErrorCount tracks the number of cache key generation
	// failures in the builder.
	BuilderCacheKeyGenErrorCount metric.Int64Counter

	// BuilderSourceParseCount tracks the number of source file parse operations.
	BuilderSourceParseCount metric.Int64Counter

	// BuilderSourceParseDuration records the time taken to parse source files.
	BuilderSourceParseDuration metric.Float64Histogram

	// BuilderSourceParseErrorCount tracks the number of source file parsing errors
	// found by the documentation builder.
	BuilderSourceParseErrorCount metric.Int64Counter

	// BuilderCacheGetCount tracks the number of cache retrieval operations.
	BuilderCacheGetCount metric.Int64Counter

	// BuilderCacheSetCount tracks the number of items added to the builder cache.
	BuilderCacheSetCount metric.Int64Counter

	// QuerierCacheHitCount tracks the number of cache hits in the querier.
	QuerierCacheHitCount metric.Int64Counter

	// QuerierCacheMissCount tracks the number of cache misses in the querier.
	QuerierCacheMissCount metric.Int64Counter
)

func init() {
	var err error

	BuilderBuildCount, err = Meter.Int64Counter(
		"inspector.builder.build.count",
		metric.WithDescription("Number of type querier initial build operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderBuildDuration, err = Meter.Float64Histogram(
		"inspector.builder.build.duration",
		metric.WithDescription("Duration of type querier initial build operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderBuildErrorCount, err = Meter.Int64Counter(
		"inspector.builder.build.errors",
		metric.WithDescription("Number of type querier initial build errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderPackageLoadCount, err = Meter.Int64Counter(
		"inspector.builder.package_load.count",
		metric.WithDescription("Number of type querier build and set operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderPackageLoadDuration, err = Meter.Float64Histogram(
		"inspector.builder.package_load.duration",
		metric.WithDescription("Duration of type querier build and set operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderPackageLoadErrorCount, err = Meter.Int64Counter(
		"inspector.builder.package_load.errors",
		metric.WithDescription("Number of type querier build and set errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderCacheKeyGenCount, err = Meter.Int64Counter(
		"inspector.builder.cache_key_generation.count",
		metric.WithDescription("Number of type querier generate cache key operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderCacheKeyGenDuration, err = Meter.Float64Histogram(
		"inspector.builder.cache_key_generation.duration",
		metric.WithDescription("Duration of type querier generate cache key operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderCacheKeyGenErrorCount, err = Meter.Int64Counter(
		"inspector.builder.cache_key_generation.errors",
		metric.WithDescription("Number of type querier generate cache key errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderSourceParseCount, err = Meter.Int64Counter(
		"inspector.builder.source_parse.count",
		metric.WithDescription("Number of type querier concurrent parse operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderSourceParseDuration, err = Meter.Float64Histogram(
		"inspector.builder.source_parse.duration",
		metric.WithDescription("Duration of type querier concurrent parse operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderSourceParseErrorCount, err = Meter.Int64Counter(
		"inspector.builder.source_parse.duration",
		metric.WithDescription("Number of type querier concurrent parse errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderCacheGetCount, err = Meter.Int64Counter(
		"inspector.builder.cache.gets",
		metric.WithDescription("Number of type querier get operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuilderCacheSetCount, err = Meter.Int64Counter(
		"inspector.builder.cache.sets",
		metric.WithDescription("Number of type querier set operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	QuerierCacheHitCount, err = Meter.Int64Counter(
		"inspector.querier.memoization.cache_hits",
		metric.WithDescription("Number of type resolution cache hits in memoization"),
	)
	if err != nil {
		otel.Handle(err)
	}

	QuerierCacheMissCount, err = Meter.Int64Counter(
		"inspector.querier.memoization.cache_misses",
		metric.WithDescription("Number of type resolution cache misses in memoization"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
