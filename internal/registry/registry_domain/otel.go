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

package registry_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the registry domain package.
	log = logger_domain.GetLogger("piko/internal/registry/registry_domain")

	// meter is the OpenTelemetry meter for registry domain metrics.
	meter = otel.Meter("piko/internal/registry/registry_domain")

	// registryServiceUpsertArtefactDuration measures the latency of artefact
	// upsert operations. This is a core mutation operation critical for
	// understanding registry write performance.
	registryServiceUpsertArtefactDuration metric.Float64Histogram

	// registryServiceAddVariantDuration measures the latency of variant addition
	// operations. Tracks performance of capability output registration, a key part
	// of the asset pipeline.
	registryServiceAddVariantDuration metric.Float64Histogram

	// registryServiceDeleteArtefactDuration measures the latency of artefact
	// deletion operations. Important for understanding cleanup performance and GC
	// hint generation overhead.
	registryServiceDeleteArtefactDuration metric.Float64Histogram

	// registryServiceGetArtefactDuration measures the latency of single artefact
	// retrieval. High-frequency read operation; essential for monitoring cache
	// effectiveness and query performance.
	registryServiceGetArtefactDuration metric.Float64Histogram

	// registryServiceGetMultipleArtefactsDuration measures the latency of batch
	// artefact retrieval. Critical for understanding bulk read performance and
	// cache batch efficiency.
	registryServiceGetMultipleArtefactsDuration metric.Float64Histogram

	// registryServiceUpsertArtefactErrorCount tracks failures in artefact upsert
	// operations. High error rates indicate storage backend issues or validation
	// problems.
	registryServiceUpsertArtefactErrorCount metric.Int64Counter

	// registryServiceUpsertArtefactSkippedCount tracks upsert operations that were
	// skipped because the artefact was unchanged. This optimisation prevents
	// unnecessary event storms during rapid page reloads when dynamic assets are
	// re-registered with identical profiles.
	registryServiceUpsertArtefactSkippedCount metric.Int64Counter

	// registryServiceAddVariantErrorCount tracks failures in variant addition
	// operations. Indicates issues with capability output registration or metadata
	// inconsistencies.
	registryServiceAddVariantErrorCount metric.Int64Counter

	// registryServiceDeleteArtefactErrorCount tracks failures in artefact deletion
	// operations. May indicate ref counting bugs or database transaction failures.
	registryServiceDeleteArtefactErrorCount metric.Int64Counter

	// registryServiceBlobDeduplicationHitCount tracks how often content
	// deduplication saves storage.
	//
	// When an artefact is uploaded with content that already exists (same
	// hash), the blob is reused. A high hit rate indicates effective
	// deduplication and storage savings.
	registryServiceBlobDeduplicationHitCount metric.Int64Counter

	// registryServiceBlobRefCountIncrementCount tracks blob reference count
	// increments. Each increment represents a new artefact or variant
	// referencing a blob.
	registryServiceBlobRefCountIncrementCount metric.Int64Counter

	// registryServiceBlobRefCountDecrementCount tracks blob reference count
	// decrements. Each decrement represents an artefact or variant no longer
	// referencing a blob.
	registryServiceBlobRefCountDecrementCount metric.Int64Counter

	// registryServiceCacheHitCount tracks successful cache lookups for artefacts.
	// High hit rate reduces database load and improves response times.
	registryServiceCacheHitCount metric.Int64Counter

	// registryServiceCacheMissCount tracks cache misses requiring database
	// queries. High miss rate may indicate cache sizing issues or high cardinality
	// in queries.
	registryServiceCacheMissCount metric.Int64Counter

	// registryServiceVariantInvalidationCount tracks cascading variant
	// invalidations where a source artefact change causes dependent
	// variants to become stale and require regeneration, with high
	// counts indicating significant rework in the asset pipeline.
	registryServiceVariantInvalidationCount metric.Int64Counter

	// registryServiceEventPublishCount tracks registry events published to the
	// orchestrator. Events trigger capability processing; count reflects system
	// activity level.
	registryServiceEventPublishCount metric.Int64Counter
)

// getTextMapPropagator returns the global OpenTelemetry text map propagator.
//
// Returns propagation.TextMapPropagator which handles trace context
// propagation across service boundaries.
func getTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

func init() {
	var err error

	registryServiceUpsertArtefactDuration, err = meter.Float64Histogram(
		"registry.domain.upsert_artefact.duration",
		metric.WithDescription("Duration of artefact upsert operations (create or update)"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceAddVariantDuration, err = meter.Float64Histogram(
		"registry.domain.add_variant.duration",
		metric.WithDescription("Duration of variant addition operations (capability outputs)"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceDeleteArtefactDuration, err = meter.Float64Histogram(
		"registry.domain.delete_artefact.duration",
		metric.WithDescription("Duration of artefact deletion operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceGetArtefactDuration, err = meter.Float64Histogram(
		"registry.domain.get_artefact.duration",
		metric.WithDescription("Duration of single artefact retrieval (includes cache lookup)"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceGetMultipleArtefactsDuration, err = meter.Float64Histogram(
		"registry.domain.get_multiple_artefacts.duration",
		metric.WithDescription("Duration of batch artefact retrieval operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceUpsertArtefactErrorCount, err = meter.Int64Counter(
		"registry.domain.upsert_artefact.error_count",
		metric.WithDescription("Total number of upsert artefact errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceUpsertArtefactSkippedCount, err = meter.Int64Counter(
		"registry.domain.upsert_artefact.skipped_count",
		metric.WithDescription("Number of upsert operations skipped because artefact was unchanged"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceAddVariantErrorCount, err = meter.Int64Counter(
		"registry.domain.add_variant.error_count",
		metric.WithDescription("Total number of add variant errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceDeleteArtefactErrorCount, err = meter.Int64Counter(
		"registry.domain.delete_artefact.error_count",
		metric.WithDescription("Total number of delete artefact errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceBlobDeduplicationHitCount, err = meter.Int64Counter(
		"registry.domain.blob_deduplication.hit_count",
		metric.WithDescription("Number of times content deduplication saved storage by reusing existing blobs"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceBlobRefCountIncrementCount, err = meter.Int64Counter(
		"registry.domain.blob_ref_count.increment_count",
		metric.WithDescription("Total number of blob reference count increments"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceBlobRefCountDecrementCount, err = meter.Int64Counter(
		"registry.domain.blob_ref_count.decrement_count",
		metric.WithDescription("Total number of blob reference count decrements"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceCacheHitCount, err = meter.Int64Counter(
		"registry.domain.cache.hit_count",
		metric.WithDescription("Number of successful cache lookups for artefacts"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceCacheMissCount, err = meter.Int64Counter(
		"registry.domain.cache.miss_count",
		metric.WithDescription("Number of cache misses requiring database queries"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceVariantInvalidationCount, err = meter.Int64Counter(
		"registry.domain.variant_invalidation.count",
		metric.WithDescription("Number of variants marked as stale due to source changes (cascading invalidation)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	registryServiceEventPublishCount, err = meter.Int64Counter(
		"registry.domain.event_publish.count",
		metric.WithDescription("Total number of registry events published to the orchestrator"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
