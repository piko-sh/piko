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

package render_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/render/render_domain")

	// meter is the OpenTelemetry meter for the render domain package.
	meter = otel.Meter("piko/internal/render/render_domain")

	// RenderASTDuration measures the time taken to render an AST to HTML.
	RenderASTDuration metric.Float64Histogram

	// BuildThemeCSSCount tracks the number of theme CSS build operations.
	BuildThemeCSSCount metric.Int64Counter

	// BuildThemeCSSErrorCount counts errors that happen when building theme CSS.
	BuildThemeCSSErrorCount metric.Int64Counter

	// CollectMetadataCount tracks the number of times metadata has been collected.
	CollectMetadataCount metric.Int64Counter

	// CollectMetadataErrorCount tracks the number of errors that happen when
	// gathering metadata.
	CollectMetadataErrorCount metric.Int64Counter

	// RenderASTCount tracks the number of AST render operations.
	RenderASTCount metric.Int64Counter

	// RenderASTErrorCount tracks the number of errors that happen during AST
	// rendering.
	RenderASTErrorCount metric.Int64Counter

	// BuildSvgSpriteSheetCount tracks the number of SVG sprite sheet build
	// operations.
	BuildSvgSpriteSheetCount metric.Int64Counter

	// BuildSvgSpriteSheetErrorCount tracks errors during SVG sprite sheet builds.
	BuildSvgSpriteSheetErrorCount metric.Int64Counter

	// SpriteSheetCacheHitCount tracks cache hits for assembled sprite sheets.
	SpriteSheetCacheHitCount metric.Int64Counter

	// ComponentRenderCount tracks individual component render operations.
	ComponentRenderCount metric.Int64Counter

	// SVGSymbolCount tracks the number of SVG symbols processed.
	SVGSymbolCount metric.Int64Counter

	// SVGTransformCount counts SVG transform operations.
	SVGTransformCount metric.Int64Counter

	// SVGTransformErrorCount tracks errors that occur during SVG transformations.
	SVGTransformErrorCount metric.Int64Counter

	// LinkTransformCount counts link transformation operations.
	LinkTransformCount metric.Int64Counter

	// ImgTransformCount tracks image transformation operations.
	ImgTransformCount metric.Int64Counter

	// PictureTransformCount tracks picture element transformation operations.
	PictureTransformCount metric.Int64Counter

	// VideoTransformCount tracks video transformation operations.
	VideoTransformCount metric.Int64Counter

	// VideoTransformErrorCount tracks errors during video transformations.
	VideoTransformErrorCount metric.Int64Counter
)

func init() {
	var err error

	RenderASTDuration, err = meter.Float64Histogram(
		"render.domain.render_ast_duration",
		metric.WithDescription("Duration of rendering AST operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuildThemeCSSCount, err = meter.Int64Counter(
		"render.domain.build_theme_css_count",
		metric.WithDescription("Number of theme CSS build operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuildThemeCSSErrorCount, err = meter.Int64Counter(
		"render.domain.build_theme_css_error_count",
		metric.WithDescription("Number of theme CSS build errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectMetadataCount, err = meter.Int64Counter(
		"render.domain.collect_metadata_count",
		metric.WithDescription("Number of metadata collection operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CollectMetadataErrorCount, err = meter.Int64Counter(
		"render.domain.collect_metadata_error_count",
		metric.WithDescription("Number of metadata collection errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	RenderASTCount, err = meter.Int64Counter(
		"render.domain.render_ast_count",
		metric.WithDescription("Number of AST render operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	RenderASTErrorCount, err = meter.Int64Counter(
		"render.domain.render_ast_error_count",
		metric.WithDescription("Number of AST render errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuildSvgSpriteSheetCount, err = meter.Int64Counter(
		"render.domain.build_svg_sprite_sheet_count",
		metric.WithDescription("Number of SVG sprite sheet build operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	BuildSvgSpriteSheetErrorCount, err = meter.Int64Counter(
		"render.domain.build_svg_sprite_sheet_error_count",
		metric.WithDescription("Number of SVG sprite sheet build errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SpriteSheetCacheHitCount, err = meter.Int64Counter(
		"render.domain.sprite_sheet_cache_hit_count",
		metric.WithDescription("Number of cache hits for assembled sprite sheets"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ComponentRenderCount, err = meter.Int64Counter(
		"render.domain.component_render_count",
		metric.WithDescription("Number of component render operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SVGSymbolCount, err = meter.Int64Counter(
		"render.domain.svg_symbol_count",
		metric.WithDescription("Number of SVG symbols processed"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SVGTransformCount, err = meter.Int64Counter(
		"render.domain.svg_transform_count",
		metric.WithDescription("Number of SVG transform operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SVGTransformErrorCount, err = meter.Int64Counter(
		"render.domain.svg_transform_error_count",
		metric.WithDescription("Number of SVG transform errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	LinkTransformCount, err = meter.Int64Counter(
		"render.domain.link_transform_count",
		metric.WithDescription("Number of link transform operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ImgTransformCount, err = meter.Int64Counter(
		"render.domain.img_transform_count",
		metric.WithDescription("Number of image transform operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PictureTransformCount, err = meter.Int64Counter(
		"render.domain.picture_transform_count",
		metric.WithDescription("Number of picture transform operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VideoTransformCount, err = meter.Int64Counter(
		"render.domain.video_transform_count",
		metric.WithDescription("Number of video transform operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VideoTransformErrorCount, err = meter.Int64Counter(
		"render.domain.video_transform_error_count",
		metric.WithDescription("Number of video transform errors"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
