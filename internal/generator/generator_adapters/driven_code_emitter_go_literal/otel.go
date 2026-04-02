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

package driven_code_emitter_go_literal

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is a package-level logger instance specific to the go_literal emitter.
	log = logger_domain.GetLogger("piko/internal/generator/generator_emitter/go_literal")

	// Meter is the OpenTelemetry meter for this package, used to create metrics.
	Meter = otel.Meter("piko/internal/generator/generator_emitter/go_literal")

	// CodeEmissionCount counts the number of times EmitCode is called.
	// This helps track the overall workload of the emitter.
	CodeEmissionCount metric.Int64Counter

	// CodeEmissionDuration measures the time taken for each EmitCode operation, in
	// milliseconds. This is a key performance indicator for the code generation
	// stage.
	CodeEmissionDuration metric.Float64Histogram

	// CodeEmissionErrorCount counts the number of fatal, unrecoverable errors that
	// occur during code emission, such as failures to format the final Go code.
	CodeEmissionErrorCount metric.Int64Counter

	// StaticNodeHoistedCount counts the number of individual static AST nodes that
	// are successfully hoisted into the init() function. This provides insight
	// into the effectiveness of the static analysis optimisation.
	StaticNodeHoistedCount metric.Int64Counter

	// PrerenderedNodeCount counts the number of static nodes that are prerendered
	// to HTML bytes at generation time. These nodes skip AST walking at runtime.
	PrerenderedNodeCount metric.Int64Counter
)

func init() {
	var err error

	CodeEmissionCount, err = Meter.Int64Counter(
		"piko.generator.emitter.emission.count",
		metric.WithDescription("Total number of code emission operations initiated."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CodeEmissionDuration, err = Meter.Float64Histogram(
		"piko.generator.emitter.emission.duration",
		metric.WithDescription("The duration of a single code emission operation."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CodeEmissionErrorCount, err = Meter.Int64Counter(
		"piko.generator.emitter.emission.errors",
		metric.WithDescription("Total number of fatal errors during code emission."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	StaticNodeHoistedCount, err = Meter.Int64Counter(
		"piko.generator.emitter.static_nodes_hoisted",
		metric.WithDescription("Total number of static AST nodes hoisted into init() functions for optimisation."),
		metric.WithUnit("{node}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	PrerenderedNodeCount, err = Meter.Int64Counter(
		"piko.generator.emitter.prerendered_nodes",
		metric.WithDescription("Total number of static nodes prerendered to HTML bytes at generation time."),
		metric.WithUnit("{node}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
