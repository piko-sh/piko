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

package resolver_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/resolver/resolver_adapters")

	// meter is the OpenTelemetry meter for resolver adapter instrumentation.
	meter = otel.Meter("piko/internal/resolver/resolver_adapters")

	// moduleDetectionCount is a counter metric that tracks how many Go modules
	// are found during analysis.
	moduleDetectionCount metric.Int64Counter

	// moduleDetectionDuration records how long it takes to detect Go modules.
	moduleDetectionDuration metric.Float64Histogram

	// moduleDetectionErrorCount tracks the number of errors that occur during
	// module detection.
	moduleDetectionErrorCount metric.Int64Counter

	// pathResolutionCount is a metric counter that tracks how many times path
	// resolution is done.
	pathResolutionCount metric.Int64Counter

	// pathResolutionErrorCount is a metric counter that tracks errors when
	// resolving paths during documentation processing.
	pathResolutionErrorCount metric.Int64Counter

	// goModuleCacheResolutionCount tracks the number of Go module cache
	// resolution attempts for metrics reporting.
	goModuleCacheResolutionCount metric.Int64Counter

	// goModuleCacheResolutionDuration records how long it takes to resolve Go
	// module paths from the module cache.
	goModuleCacheResolutionDuration metric.Float64Histogram

	// goModuleCacheResolutionErrorCount tracks the number of failed attempts to
	// resolve paths from the Go module cache.
	goModuleCacheResolutionErrorCount metric.Int64Counter

	// goModuleCacheHitCount tracks the number of cache hits when resolving Go
	// module paths.
	goModuleCacheHitCount metric.Int64Counter

	// goModuleCacheMissCount is a metric counter that tracks Go module cache misses.
	goModuleCacheMissCount metric.Int64Counter
)

func init() {
	var err error
	moduleDetectionCount, err = meter.Int64Counter(
		"resolver.fs.module_detection.count",
		metric.WithDescription("Number of module detection operations by the FS resolver"),
	)
	if err != nil {
		otel.Handle(err)
	}

	moduleDetectionDuration, err = meter.Float64Histogram(
		"resolver.fs.module_detection.duration",
		metric.WithDescription("Duration of module detection operations by the FS resolver"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	moduleDetectionErrorCount, err = meter.Int64Counter(
		"resolver.fs.module_detection.errors",
		metric.WithDescription("Number of module detection errors from the FS resolver"),
	)
	if err != nil {
		otel.Handle(err)
	}

	pathResolutionCount, err = meter.Int64Counter(
		"resolver.fs.path_resolution.count",
		metric.WithDescription("Number of path resolution operations by the FS resolver"),
	)
	if err != nil {
		otel.Handle(err)
	}

	pathResolutionErrorCount, err = meter.Int64Counter(
		"resolver.fs.path_resolution.errors",
		metric.WithDescription("Number of path resolution errors from the FS resolver"),
	)
	if err != nil {
		otel.Handle(err)
	}

	goModuleCacheResolutionCount, err = meter.Int64Counter(
		"resolver.gomodcache.resolution.count",
		metric.WithDescription("Number of resolution operations from Go module cache"),
	)
	if err != nil {
		otel.Handle(err)
	}

	goModuleCacheResolutionDuration, err = meter.Float64Histogram(
		"resolver.gomodcache.resolution.duration",
		metric.WithDescription("Duration of resolution operations from Go module cache"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	goModuleCacheResolutionErrorCount, err = meter.Int64Counter(
		"resolver.gomodcache.resolution.errors",
		metric.WithDescription("Number of resolution errors from Go module cache"),
	)
	if err != nil {
		otel.Handle(err)
	}

	goModuleCacheHitCount, err = meter.Int64Counter(
		"resolver.gomodcache.cache.hits",
		metric.WithDescription("Number of cache hits for Go module directory lookups"),
	)
	if err != nil {
		otel.Handle(err)
	}

	goModuleCacheMissCount, err = meter.Int64Counter(
		"resolver.gomodcache.cache.misses",
		metric.WithDescription("Number of cache misses for Go module directory lookups"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
