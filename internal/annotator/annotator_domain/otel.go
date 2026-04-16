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

package annotator_domain

// Defines OpenTelemetry metrics for monitoring compilation performance and
// tracking operation counts. Provides counters, histograms, and gauges for
// observing annotator behaviour including parse times, CSS processing, and
// error rates.

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the annotator_domain package.
	log = logger_domain.GetLogger("piko/internal/annotator/annotator_domain")

	// Meter is the OpenTelemetry meter for the annotator domain package.
	Meter = otel.Meter("piko/internal/annotator/annotator_domain")

	// CSSProcessCount tracks the number of CSS processing tasks that have run.
	CSSProcessCount metric.Int64Counter

	// CSSProcessDuration tracks how long CSS processing tasks take.
	CSSProcessDuration metric.Float64Histogram

	// CSSProcessErrorCount tracks the number of errors that occur when processing
	// CSS files.
	CSSProcessErrorCount metric.Int64Counter

	// CSSScopeCount tracks the number of CSS scoping operations that have
	// happened.
	CSSScopeCount metric.Int64Counter

	// CSSScopeDuration tracks how long CSS scoping operations take.
	CSSScopeDuration metric.Float64Histogram

	// CSSScopeErrorCount is a counter metric that tracks CSS scoping errors.
	CSSScopeErrorCount metric.Int64Counter

	// TransformASTCount tracks the number of AST transformation operations.
	TransformASTCount metric.Int64Counter

	// TransformASTDuration records the time taken to apply AST changes.
	TransformASTDuration metric.Float64Histogram

	// TransformASTErrorCount counts errors that occur during AST changes.
	TransformASTErrorCount metric.Int64Counter

	// WalkASTNodeCount tracks the number of AST node walk operations.
	WalkASTNodeCount metric.Int64Counter

	// WalkASTNodeDuration tracks how long it takes to walk through AST nodes.
	WalkASTNodeDuration metric.Float64Histogram

	// PartialExpandCount is a counter that tracks the number of partial expand
	// operations.
	PartialExpandCount metric.Int64Counter

	// PartialExpandDuration tracks the duration of partial expand operations.
	PartialExpandDuration metric.Float64Histogram

	// PartialExpandErrorCount tracks the number of partial expand errors.
	PartialExpandErrorCount metric.Int64Counter

	// PartialRecursiveExpandCount tracks the number of recursive partial expand
	// operations.
	PartialRecursiveExpandCount metric.Int64Counter

	// PartialRecursiveExpandDuration tracks the duration of recursive partial
	// expand operations.
	PartialRecursiveExpandDuration metric.Float64Histogram

	// PartialHandleExpansionCount tracks the number of partial handle expansion
	// operations.
	PartialHandleExpansionCount metric.Int64Counter

	// PartialHandleExpansionDuration tracks the duration of partial handle
	// expansion operations.
	PartialHandleExpansionDuration metric.Float64Histogram

	// PartialFillSlotsCount counts the number of partial fill slots operations.
	PartialFillSlotsCount metric.Int64Counter

	// PartialFillSlotsDuration tracks the duration of partial fill slots
	// operations.
	PartialFillSlotsDuration metric.Float64Histogram

	// PartialAnnotateTypesCount tracks the number of partial annotate types
	// operations.
	PartialAnnotateTypesCount metric.Int64Counter

	// PartialAnnotateTypesDuration tracks the duration of partial annotate types
	// operations.
	PartialAnnotateTypesDuration metric.Float64Histogram
)

func init() {
	var err error

	CSSProcessCount, err = Meter.Int64Counter(
		"generator.css_process_count",
		metric.WithDescription("Number of CSS processing operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSProcessDuration, err = Meter.Float64Histogram(
		"generator.css_process_duration",
		metric.WithDescription("Duration of CSS processing operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSProcessErrorCount, err = Meter.Int64Counter(
		"generator.css_process_error_count",
		metric.WithDescription("Number of CSS processing errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSScopeCount, err = Meter.Int64Counter(
		"generator.css_scope_count",
		metric.WithDescription("Number of CSS scoping operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSScopeDuration, err = Meter.Float64Histogram(
		"generator.css_scope_duration",
		metric.WithDescription("Duration of CSS scoping operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSScopeErrorCount, err = Meter.Int64Counter(
		"generator.css_scope_error_count",
		metric.WithDescription("Number of CSS scoping errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialExpandCount, err = Meter.Int64Counter(
		"generator.partial_expand_count",
		metric.WithDescription("Number of partial expand operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialExpandDuration, err = Meter.Float64Histogram(
		"generator.partial_expand_duration",
		metric.WithDescription("Duration of partial expand operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialExpandErrorCount, err = Meter.Int64Counter(
		"generator.partial_expand_error_count",
		metric.WithDescription("Number of partial expand errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialRecursiveExpandCount, err = Meter.Int64Counter(
		"generator.partial_recursive_expand_count",
		metric.WithDescription("Number of recursive partial expand operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialRecursiveExpandDuration, err = Meter.Float64Histogram(
		"generator.partial_recursive_expand_duration",
		metric.WithDescription("Duration of recursive partial expand operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialHandleExpansionCount, err = Meter.Int64Counter(
		"generator.partial_handle_expansion_count",
		metric.WithDescription("Number of partial handle expansion operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialHandleExpansionDuration, err = Meter.Float64Histogram(
		"generator.partial_handle_expansion_duration",
		metric.WithDescription("Duration of partial handle expansion operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialFillSlotsCount, err = Meter.Int64Counter(
		"generator.partial_fill_slots_count",
		metric.WithDescription("Number of partial fill slots operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialFillSlotsDuration, err = Meter.Float64Histogram(
		"generator.partial_fill_slots_duration",
		metric.WithDescription("Duration of partial fill slots operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialAnnotateTypesCount, err = Meter.Int64Counter(
		"generator.partial_annotate_types_count",
		metric.WithDescription("Number of partial annotate types operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PartialAnnotateTypesDuration, err = Meter.Float64Histogram(
		"generator.partial_annotate_types_duration",
		metric.WithDescription("Duration of partial annotate types operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TransformASTCount, err = Meter.Int64Counter(
		"generator.transform_ast_count",
		metric.WithDescription("Number of AST transformation operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TransformASTDuration, err = Meter.Float64Histogram(
		"generator.transform_ast_duration",
		metric.WithDescription("Duration of AST transformation operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	TransformASTErrorCount, err = Meter.Int64Counter(
		"generator.transform_ast_error_count",
		metric.WithDescription("Number of AST transformation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WalkASTNodeCount, err = Meter.Int64Counter(
		"generator.walk_ast_node_count",
		metric.WithDescription("Number of AST node walk operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WalkASTNodeDuration, err = Meter.Float64Histogram(
		"generator.walk_ast_node_duration",
		metric.WithDescription("Duration of AST node walk operations"),
		metric.WithUnit("ms"),
	)

	if err != nil {
		otel.Handle(err)
	}
}
