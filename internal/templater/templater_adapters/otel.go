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

package templater_adapters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// logFieldPath is the log attribute key for template paths.
	logFieldPath = "path"

	// logFieldCacheKey is the log attribute key for cache keys.
	logFieldCacheKey = "cacheKey"
)

var (
	// log is the package-level logger for the templater_adapters package.
	log = logger_domain.GetLogger("piko/internal/templater/templater_adapters")

	// meter is the OpenTelemetry meter for templater adapters metrics.
	meter = otel.Meter("piko/internal/templater/templater_adapters")

	// InterpretedManifestRunnerRunPageCount tracks the number of RunPage operations in
	// interpreted mode.
	InterpretedManifestRunnerRunPageCount metric.Int64Counter

	// InterpretedManifestRunnerRunPageErrorCount tracks RunPage errors in interpreted
	// mode.
	InterpretedManifestRunnerRunPageErrorCount metric.Int64Counter

	// InterpretedManifestRunnerRunPageDuration measures RunPage operation duration in
	// interpreted mode.
	InterpretedManifestRunnerRunPageDuration metric.Float64Histogram

	// InterpretedManifestRunnerRunPartialCount tracks the number of RunPartial
	// operations in interpreted mode.
	InterpretedManifestRunnerRunPartialCount metric.Int64Counter

	// InterpretedManifestRunnerRunPartialErrorCount tracks RunPartial errors in
	// interpreted mode.
	InterpretedManifestRunnerRunPartialErrorCount metric.Int64Counter

	// InterpretedManifestRunnerRunPartialDuration measures
	// RunPartial operation duration in interpreted mode.
	InterpretedManifestRunnerRunPartialDuration metric.Float64Histogram

	// InterpretedManifestRunnerGetPageEntryCount tracks the number of GetPageEntry
	// operations in interpreted mode.
	InterpretedManifestRunnerGetPageEntryCount metric.Int64Counter

	// InterpretedManifestRunnerGetPageEntryErrorCount tracks GetPageEntry errors in
	// interpreted mode.
	InterpretedManifestRunnerGetPageEntryErrorCount metric.Int64Counter

	// InterpretedManifestRunnerGetPageEntryDuration measures GetPageEntry operation
	// duration in interpreted mode.
	InterpretedManifestRunnerGetPageEntryDuration metric.Float64Histogram

	// InterpretedManifestRunnerCompilationCount tracks the number of template
	// compilations in interpreted mode.
	InterpretedManifestRunnerCompilationCount metric.Int64Counter

	// InterpretedManifestRunnerCompilationErrorCount tracks
	// template compilation errors in interpreted mode.
	InterpretedManifestRunnerCompilationErrorCount metric.Int64Counter

	// InterpretedManifestRunnerCompilationDuration measures template compilation
	// duration in interpreted mode.
	InterpretedManifestRunnerCompilationDuration metric.Float64Histogram

	// CompiledManifestRunnerRunPageCount tracks the number of RunPage operations
	// in compiled mode.
	CompiledManifestRunnerRunPageCount metric.Int64Counter

	// CompiledManifestRunnerRunPageErrorCount tracks RunPage errors in compiled
	// mode.
	CompiledManifestRunnerRunPageErrorCount metric.Int64Counter

	// CompiledManifestRunnerRunPageDuration measures RunPage operation duration in
	// compiled mode.
	CompiledManifestRunnerRunPageDuration metric.Float64Histogram

	// CompiledManifestRunnerRunPartialCount tracks the number of RunPartial
	// operations in compiled mode.
	CompiledManifestRunnerRunPartialCount metric.Int64Counter

	// CompiledManifestRunnerRunPartialErrorCount tracks RunPartial errors in
	// compiled mode.
	CompiledManifestRunnerRunPartialErrorCount metric.Int64Counter

	// CompiledManifestRunnerRunPartialDuration measures RunPartial operation
	// duration in compiled mode.
	CompiledManifestRunnerRunPartialDuration metric.Float64Histogram

	// CompiledManifestRunnerGetPageEntryCount tracks the number of GetPageEntry
	// operations in compiled mode.
	CompiledManifestRunnerGetPageEntryCount metric.Int64Counter

	// CompiledManifestRunnerGetPageEntryErrorCount tracks GetPageEntry errors in
	// compiled mode.
	CompiledManifestRunnerGetPageEntryErrorCount metric.Int64Counter

	// CompiledManifestRunnerGetPageEntryDuration measures GetPageEntry operation
	// duration in compiled mode.
	CompiledManifestRunnerGetPageEntryDuration metric.Float64Histogram
)

func init() {
	var err error

	InterpretedManifestRunnerRunPageCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_run_page_count",
		metric.WithDescription("Number of RunPage operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerRunPageErrorCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_run_page_error_count",
		metric.WithDescription("Number of RunPage errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerRunPageDuration, err = meter.Float64Histogram(
		"templater.adapters.interpreted_manifest_runner_run_page_duration",
		metric.WithDescription("Duration of RunPage operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerRunPartialCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_run_partial_count",
		metric.WithDescription("Number of RunPartial operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerRunPartialErrorCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_run_partial_error_count",
		metric.WithDescription("Number of RunPartial errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerRunPartialDuration, err = meter.Float64Histogram(
		"templater.adapters.interpreted_manifest_runner_run_partial_duration",
		metric.WithDescription("Duration of RunPartial operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerGetPageEntryCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_get_page_entry_count",
		metric.WithDescription("Number of GetPageEntry operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerGetPageEntryErrorCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_get_page_entry_error_count",
		metric.WithDescription("Number of GetPageEntry errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerGetPageEntryDuration, err = meter.Float64Histogram(
		"templater.adapters.interpreted_manifest_runner_get_page_entry_duration",
		metric.WithDescription("Duration of GetPageEntry operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerCompilationCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_compilation_count",
		metric.WithDescription("Number of template compilations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerCompilationErrorCount, err = meter.Int64Counter(
		"templater.adapters.interpreted_manifest_runner_compilation_error_count",
		metric.WithDescription("Number of template compilation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	InterpretedManifestRunnerCompilationDuration, err = meter.Float64Histogram(
		"templater.adapters.interpreted_manifest_runner_compilation_duration",
		metric.WithDescription("Duration of template compilations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerRunPageCount, err = meter.Int64Counter(
		"templater.adapters.compiled_manifest_runner_run_page_count",
		metric.WithDescription("Number of RunPage operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerRunPageErrorCount, err = meter.Int64Counter(
		"templater.adapters.compiled_manifest_runner_run_page_error_count",
		metric.WithDescription("Number of RunPage errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerRunPageDuration, err = meter.Float64Histogram(
		"templater.adapters.compiled_manifest_runner_run_page_duration",
		metric.WithDescription("Duration of RunPage operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerRunPartialCount, err = meter.Int64Counter(
		"templater.adapters.compiled_manifest_runner_run_partial_count",
		metric.WithDescription("Number of RunPartial operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerRunPartialErrorCount, err = meter.Int64Counter(
		"templater.adapters.compiled_manifest_runner_run_partial_error_count",
		metric.WithDescription("Number of RunPartial errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerRunPartialDuration, err = meter.Float64Histogram(
		"templater.adapters.compiled_manifest_runner_run_partial_duration",
		metric.WithDescription("Duration of RunPartial operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerGetPageEntryCount, err = meter.Int64Counter(
		"templater.adapters.compiled_manifest_runner_get_page_entry_count",
		metric.WithDescription("Number of GetPageEntry operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerGetPageEntryErrorCount, err = meter.Int64Counter(
		"templater.adapters.compiled_manifest_runner_get_page_entry_error_count",
		metric.WithDescription("Number of GetPageEntry errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CompiledManifestRunnerGetPageEntryDuration, err = meter.Float64Histogram(
		"templater.adapters.compiled_manifest_runner_get_page_entry_duration",
		metric.WithDescription("Duration of GetPageEntry operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
