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

package db_provider_seagull

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"seagull.sh/seagull/sdk/observe"
)

const (
	// instrumentationName identifies Piko's SQL tracing and metrics instrumentation.
	instrumentationName = "piko/internal/querier/querier_domain"

	// unitCall is the metric unit for operation counters.
	unitCall = "{call}"

	// unitError is the metric unit for error counters.
	unitError = "{error}"

	// unitMilliseconds is the metric unit for duration histograms.
	unitMilliseconds = "ms"
)

var (
	// spanNames maps Seagull stages to Piko tracing span names.
	spanNames = map[observe.Stage]string{
		observe.StageGenerate:           "QuerierService.GenerateDatabase",
		observe.StageCatalogue:          "QuerierService.buildCatalogue",
		observe.StageAnalyse:            "QuerierService.analyseQueries",
		observe.StageAnalyseQuery:       "QueryAnalyser.AnalyseQuery",
		observe.StageResolveColumns:     "TypeResolver.ResolveOutputColumns",
		observe.StageResolveParameters:  "TypeResolver.ResolveParameters",
		observe.StageCatalogueMigration: "CatalogueBuilder.ApplyMigration",
		observe.StageMigrateUp:          "MigrationService.Up",
		observe.StageMigrateDown:        "MigrationService.Down",
		observe.StageMigrateStatus:      "MigrationService.Status",
		observe.StageMigrateValidate:    "MigrationService.Validate",
		observe.StageMigrateForce:       "MigrationService.Force",
		observe.StageMigrateRepairPlan:  "MigrationService.PlanRepair",
		observe.StageMigrateRepair:      "MigrationService.Repair",
		observe.StageVet:                "QuerierService.Vet",
		observe.StageSeedApply:          "SeedService.Apply",
		observe.StageSeedStatus:         "SeedService.Status",
		observe.StageSeedReseed:         "SeedService.Reseed",

		// Generation's planning, emitting and formatting had no span of their own in Piko, so
		// they keep Seagull's stage names.
		observe.StagePlan:   string(observe.StagePlan),
		observe.StageEmit:   string(observe.StageEmit),
		observe.StageFormat: string(observe.StageFormat),
	}
)

// stageMetrics holds the counter and histogram recorded for one stage.
type stageMetrics struct {
	// count counts operations.
	count metric.Int64Counter

	// duration records operation durations in milliseconds.
	duration metric.Float64Histogram
}

// otelObserver reports Seagull's stages as Piko's OpenTelemetry spans and metrics.
type otelObserver struct {
	// tracer starts the spans.
	tracer trace.Tracer

	// catalogue holds the catalogue build metrics.
	catalogue stageMetrics

	// analyse holds the query analysis metrics.
	analyse stageMetrics

	// generate holds the full generation pipeline metrics.
	generate stageMetrics

	// generationErrors counts failed generation runs.
	generationErrors metric.Int64Counter
}

// newOTelObserver builds the observer from the global OpenTelemetry providers.
//
// Instrument creation errors are reported through otel.Handle and leave the affected
// instrument as a no-op, so observability never blocks generation or migrations.
//
// Returns *otelObserver which is ready to use.
func newOTelObserver() *otelObserver {
	meter := otel.Meter(instrumentationName)
	generationErrors, err := meter.Int64Counter("querier.service.generation.errors",
		metric.WithDescription("Number of errors during generation operations."),
		metric.WithUnit(unitError),
	)
	if err != nil {
		otel.Handle(err)
	}
	return &otelObserver{
		tracer:           otel.Tracer(instrumentationName),
		catalogue:        newStageMetrics(meter, "querier.service.catalogue_build", "catalogue build"),
		analyse:          newStageMetrics(meter, "querier.service.query_analysis", "query analysis"),
		generate:         newStageMetrics(meter, "querier.service.generation", "full generation pipeline"),
		generationErrors: generationErrors,
	}
}

// newStageMetrics creates the counter and histogram for one stage.
//
// Takes meter (metric.Meter) which creates the instruments.
// Takes prefix (string) which is the metric name prefix.
// Takes subject (string) which names the operation in descriptions.
//
// Returns stageMetrics which holds both instruments.
func newStageMetrics(meter metric.Meter, prefix, subject string) stageMetrics {
	count, err := meter.Int64Counter(prefix+".count",
		metric.WithDescription("Number of "+subject+" operations."),
		metric.WithUnit(unitCall),
	)
	if err != nil {
		otel.Handle(err)
	}
	duration, err := meter.Float64Histogram(prefix+".duration",
		metric.WithDescription("Duration of "+subject+" operations."),
		metric.WithUnit(unitMilliseconds),
	)
	if err != nil {
		otel.Handle(err)
	}
	return stageMetrics{count: count, duration: duration}
}

// Start opens a span for stage and returns the function that closes it.
//
// Takes stage (observe.Stage) which names the operation.
// Takes attrs (...slog.Attr) which become span attributes.
//
// Returns context.Context which carries the span.
// Returns func(error) which records the outcome and ends the span.
func (o *otelObserver) Start(ctx context.Context, stage observe.Stage, attrs ...slog.Attr) (context.Context, func(error)) {
	name, known := spanNames[stage]
	if !known {
		name = string(stage)
	}
	ctx, span := o.tracer.Start(ctx, name, trace.WithAttributes(toAttributes(attrs)...))
	started := time.Now()
	metrics, measured := o.metricsFor(stage)
	if measured && metrics.count != nil {
		metrics.count.Add(ctx, 1)
	}

	return ctx, func(err error) {
		if measured && metrics.duration != nil {
			metrics.duration.Record(ctx, float64(time.Since(started).Milliseconds()))
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if stage == observe.StageGenerate && o.generationErrors != nil {
				o.generationErrors.Add(ctx, 1)
			}
		}
		span.End()
	}
}

// toAttributes converts slog attributes into OpenTelemetry attributes.
//
// Takes attrs ([]slog.Attr) which are converted.
//
// Returns []attribute.KeyValue which holds one attribute per input.
func toAttributes(attrs []slog.Attr) []attribute.KeyValue {
	result := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		value := attr.Value.Resolve()
		switch value.Kind() {
		case slog.KindInt64:
			result = append(result, attribute.Int64(attr.Key, value.Int64()))
		case slog.KindBool:
			result = append(result, attribute.Bool(attr.Key, value.Bool()))
		case slog.KindFloat64:
			result = append(result, attribute.Float64(attr.Key, value.Float64()))
		default:
			result = append(result, attribute.String(attr.Key, value.String()))
		}
	}
	return result
}

// metricsFor returns the metrics for the stages Piko has always measured.
//
// Takes stage (observe.Stage) which names the operation.
//
// Returns stageMetrics which holds the stage's instruments.
// Returns bool which is false for stages Piko does not measure.
func (o *otelObserver) metricsFor(stage observe.Stage) (stageMetrics, bool) {
	switch stage {
	case observe.StageCatalogue:
		return o.catalogue, true
	case observe.StageAnalyse:
		return o.analyse, true
	case observe.StageGenerate:
		return o.generate, true
	default:
		return stageMetrics{}, false
	}
}
