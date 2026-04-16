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

package collection_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the collection_domain package.
	log = logger_domain.GetLogger("piko/internal/collection/collection_domain")

	// Meter is the OpenTelemetry meter for the collection domain.
	Meter = otel.Meter("piko/internal/collection/collection_domain")

	// NavigationBuildDuration records how long it takes to build navigation trees.
	NavigationBuildDuration metric.Float64Histogram

	// NavigationBuildCount counts how many times navigation has been built.
	NavigationBuildCount metric.Int64Counter

	// NavigationGroupCount tracks how many navigation groups have been created.
	NavigationGroupCount metric.Int64Counter

	// CollectionFetchDuration measures the time taken to fetch collection content.
	CollectionFetchDuration metric.Float64Histogram

	// CollectionFetchCount tracks the number of collection fetch operations.
	CollectionFetchCount metric.Int64Counter

	// CollectionFetchErrorCount tracks errors during collection fetching.
	CollectionFetchErrorCount metric.Int64Counter

	// CollectionItemCount tracks the number of items fetched per collection.
	CollectionItemCount metric.Int64Counter

	// HybridRevalidationCount tracks the number of hybrid revalidation operations.
	HybridRevalidationCount metric.Int64Counter

	// HybridRevalidationDuration records how long hybrid revalidation takes.
	HybridRevalidationDuration metric.Float64Histogram

	// HybridCacheHitCount tracks the number of times the hybrid cache returns a
	// result where the content has not changed.
	HybridCacheHitCount metric.Int64Counter

	// HybridCacheMissCount tracks hybrid cache misses (content changed, refetched).
	HybridCacheMissCount metric.Int64Counter

	// CollectionEncodeDuration measures the time taken to encode collections.
	CollectionEncodeDuration metric.Float64Histogram

	// CollectionEncodeCount tracks the number of encoding operations.
	CollectionEncodeCount metric.Int64Counter

	// CollectionDecodeCount tracks how many times data is converted from
	// its stored format back into Go values.
	CollectionDecodeCount metric.Int64Counter
)

func init() {
	var err error

	NavigationBuildDuration, err = Meter.Float64Histogram(
		"collection.domain.navigation_build_duration",
		metric.WithDescription("Duration of navigation tree building operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	NavigationBuildCount, err = Meter.Int64Counter(
		"collection.domain.navigation_build_count",
		metric.WithDescription("Number of navigation build operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	NavigationGroupCount, err = Meter.Int64Counter(
		"collection.domain.navigation_group_count",
		metric.WithDescription("Number of navigation groups created"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionFetchDuration, err = Meter.Float64Histogram(
		"collection.domain.fetch_duration",
		metric.WithDescription("Duration of collection fetch operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionFetchCount, err = Meter.Int64Counter(
		"collection.domain.fetch_count",
		metric.WithDescription("Number of collection fetch operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionFetchErrorCount, err = Meter.Int64Counter(
		"collection.domain.fetch_error_count",
		metric.WithDescription("Number of collection fetch errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionItemCount, err = Meter.Int64Counter(
		"collection.domain.item_count",
		metric.WithDescription("Number of items fetched per collection"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HybridRevalidationCount, err = Meter.Int64Counter(
		"collection.domain.hybrid_revalidation_count",
		metric.WithDescription("Number of hybrid revalidation operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HybridRevalidationDuration, err = Meter.Float64Histogram(
		"collection.domain.hybrid_revalidation_duration",
		metric.WithDescription("Duration of hybrid revalidation operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HybridCacheHitCount, err = Meter.Int64Counter(
		"collection.domain.hybrid_cache_hit_count",
		metric.WithDescription("Number of hybrid cache hits (content unchanged)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HybridCacheMissCount, err = Meter.Int64Counter(
		"collection.domain.hybrid_cache_miss_count",
		metric.WithDescription("Number of hybrid cache misses (content changed)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionEncodeDuration, err = Meter.Float64Histogram(
		"collection.domain.encode_duration",
		metric.WithDescription("Duration of collection encoding operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionEncodeCount, err = Meter.Int64Counter(
		"collection.domain.encode_count",
		metric.WithDescription("Number of collection encoding operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectionDecodeCount, err = Meter.Int64Counter(
		"collection.domain.decode_count",
		metric.WithDescription("Number of collection decoding operations"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
