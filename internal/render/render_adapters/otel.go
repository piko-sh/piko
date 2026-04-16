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

package render_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the render_adapters package.
	log = logger_domain.GetLogger("piko/internal/render/render_adapters")

	// meter is the OpenTelemetry meter for the render_adapters package.
	meter = otel.Meter("piko/internal/render/render_adapters")

	// componentLoaderBatchCount tracks the number of component loader batch operations.
	componentLoaderBatchCount metric.Int64Counter

	// componentLoaderCacheHitCount tracks the number of component loader cache hits.
	componentLoaderCacheHitCount metric.Int64Counter

	// componentLoaderCacheMissCount tracks the number of component loader cache misses.
	componentLoaderCacheMissCount metric.Int64Counter

	// componentLoaderErrorCount tracks the number of component loader errors.
	componentLoaderErrorCount metric.Int64Counter

	// svgLoaderBatchCount tracks the number of SVG loader batch operations.
	svgLoaderBatchCount metric.Int64Counter

	// svgLoaderCacheHitCount tracks the number of SVG loader cache hits.
	svgLoaderCacheHitCount metric.Int64Counter

	// svgLoaderCacheMissCount tracks the number of SVG loader cache misses.
	svgLoaderCacheMissCount metric.Int64Counter

	// svgLoaderErrorCount tracks the number of SVG loader errors.
	svgLoaderErrorCount metric.Int64Counter

	// svgLoaderItemFailureCount tracks the number of individual SVG items that failed to load.
	svgLoaderItemFailureCount metric.Int64Counter

	// componentLoadDuration records the duration of component load operations.
	componentLoadDuration metric.Float64Histogram

	// svgLoadDuration records the duration of SVG load operations.
	svgLoadDuration metric.Float64Histogram
)

func init() {
	var err error

	componentLoaderBatchCount, err = meter.Int64Counter(
		"render.adapters.component_loader_batch_count",
		metric.WithDescription("Number of component loader batch operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	componentLoaderCacheHitCount, err = meter.Int64Counter(
		"render.adapters.component_loader_cache_hit_count",
		metric.WithDescription("Number of component loader cache hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	componentLoaderCacheMissCount, err = meter.Int64Counter(
		"render.adapters.component_loader_cache_miss_count",
		metric.WithDescription("Number of component loader cache misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	componentLoaderErrorCount, err = meter.Int64Counter(
		"render.adapters.component_loader_error_count",
		metric.WithDescription("Number of component loader errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	svgLoaderBatchCount, err = meter.Int64Counter(
		"render.adapters.svg_loader_batch_count",
		metric.WithDescription("Number of SVG loader batch operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	svgLoaderCacheHitCount, err = meter.Int64Counter(
		"render.adapters.svg_loader_cache_hit_count",
		metric.WithDescription("Number of SVG loader cache hits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	svgLoaderCacheMissCount, err = meter.Int64Counter(
		"render.adapters.svg_loader_cache_miss_count",
		metric.WithDescription("Number of SVG loader cache misses"),
	)
	if err != nil {
		otel.Handle(err)
	}

	svgLoaderErrorCount, err = meter.Int64Counter(
		"render.adapters.svg_loader_error_count",
		metric.WithDescription("Number of SVG loader errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	svgLoaderItemFailureCount, err = meter.Int64Counter(
		"render.adapters.svg_loader_item_failure_count",
		metric.WithDescription("Number of individual SVG items that failed to load"),
	)
	if err != nil {
		otel.Handle(err)
	}

	componentLoadDuration, err = meter.Float64Histogram(
		"render.adapters.component_load_duration",
		metric.WithDescription("Duration of component load operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	svgLoadDuration, err = meter.Float64Histogram(
		"render.adapters.svg_load_duration",
		metric.WithDescription("Duration of SVG load operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
