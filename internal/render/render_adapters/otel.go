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
	log = logger_domain.GetLogger("piko/internal/render/render_adapters")

	meter = otel.Meter("piko/internal/render/render_adapters")

	componentLoaderBatchCount metric.Int64Counter

	componentLoaderCacheHitCount metric.Int64Counter

	componentLoaderCacheMissCount metric.Int64Counter

	componentLoaderErrorCount metric.Int64Counter

	svgLoaderBatchCount metric.Int64Counter

	svgLoaderCacheHitCount metric.Int64Counter

	svgLoaderCacheMissCount metric.Int64Counter

	svgLoaderErrorCount metric.Int64Counter

	svgLoaderItemFailureCount metric.Int64Counter

	componentLoadDuration metric.Float64Histogram

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
