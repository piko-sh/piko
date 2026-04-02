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

package wasm_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	// meter is the package-level meter for WASM domain metrics.
	meter = otel.Meter("piko/internal/wasm/wasm_domain")

	// analyseCount tracks the number of times the analyse operation has been run.
	analyseCount metric.Int64Counter

	// analyseDuration tracks the duration of analyse operations in milliseconds.
	analyseDuration metric.Float64Histogram

	// analyseErrorCount counts the number of errors found during analysis.
	analyseErrorCount metric.Int64Counter

	// completionCount tracks the number of completion requests.
	completionCount metric.Int64Counter

	// completionDuration records how long each completion operation takes.
	completionDuration metric.Float64Histogram

	// hoverCount tracks how many hover requests have been made.
	hoverCount metric.Int64Counter

	// parseTemplateCount tracks the number of template parse requests.
	parseTemplateCount metric.Int64Counter

	// renderPreviewCount tracks the number of render preview requests.
	renderPreviewCount metric.Int64Counter

	// sourceSizeBytes tracks the size of source code that has been analysed.
	sourceSizeBytes metric.Int64Histogram

	// generateCount tracks the number of code generation requests.
	generateCount metric.Int64Counter

	// generateDuration tracks the duration of generation operations in
	// milliseconds.
	generateDuration metric.Float64Histogram

	// generateErrorCount counts the number of generation errors.
	generateErrorCount metric.Int64Counter

	// renderCount tracks the number of render requests.
	renderCount metric.Int64Counter

	// renderDuration tracks the duration of render operations in milliseconds.
	renderDuration metric.Float64Histogram

	// renderErrorCount counts the number of render errors.
	renderErrorCount metric.Int64Counter

	// dynamicRenderCount tracks the number of dynamic render requests.
	dynamicRenderCount metric.Int64Counter

	// dynamicRenderDuration tracks the duration of dynamic render operations in
	// milliseconds.
	dynamicRenderDuration metric.Float64Histogram

	// dynamicRenderErrorCount counts the number of dynamic render errors.
	dynamicRenderErrorCount metric.Int64Counter
)

func init() {
	var err error

	analyseCount, err = meter.Int64Counter(
		"wasm.domain.analyse.count",
		metric.WithDescription("Number of analyse requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	analyseDuration, err = meter.Float64Histogram(
		"wasm.domain.analyse.duration",
		metric.WithDescription("Duration of analyse operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	analyseErrorCount, err = meter.Int64Counter(
		"wasm.domain.analyse.error_count",
		metric.WithDescription("Number of analyse errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completionCount, err = meter.Int64Counter(
		"wasm.domain.completion.count",
		metric.WithDescription("Number of completion requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completionDuration, err = meter.Float64Histogram(
		"wasm.domain.completion.duration",
		metric.WithDescription("Duration of completion operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	hoverCount, err = meter.Int64Counter(
		"wasm.domain.hover.count",
		metric.WithDescription("Number of hover requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	parseTemplateCount, err = meter.Int64Counter(
		"wasm.domain.parse_template.count",
		metric.WithDescription("Number of template parse requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	renderPreviewCount, err = meter.Int64Counter(
		"wasm.domain.render_preview.count",
		metric.WithDescription("Number of render preview requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	sourceSizeBytes, err = meter.Int64Histogram(
		"wasm.domain.source_size.bytes",
		metric.WithDescription("Size of analysed source code"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateCount, err = meter.Int64Counter(
		"wasm.domain.generate.count",
		metric.WithDescription("Number of code generation requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateDuration, err = meter.Float64Histogram(
		"wasm.domain.generate.duration",
		metric.WithDescription("Duration of generation operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	generateErrorCount, err = meter.Int64Counter(
		"wasm.domain.generate.error_count",
		metric.WithDescription("Number of generation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	renderCount, err = meter.Int64Counter(
		"wasm.domain.render.count",
		metric.WithDescription("Number of render requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	renderDuration, err = meter.Float64Histogram(
		"wasm.domain.render.duration",
		metric.WithDescription("Duration of render operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	renderErrorCount, err = meter.Int64Counter(
		"wasm.domain.render.error_count",
		metric.WithDescription("Number of render errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dynamicRenderCount, err = meter.Int64Counter(
		"wasm.domain.dynamic_render.count",
		metric.WithDescription("Number of dynamic render requests"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dynamicRenderDuration, err = meter.Float64Histogram(
		"wasm.domain.dynamic_render.duration",
		metric.WithDescription("Duration of dynamic render operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	dynamicRenderErrorCount, err = meter.Int64Counter(
		"wasm.domain.dynamic_render.error_count",
		metric.WithDescription("Number of dynamic render errors"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
