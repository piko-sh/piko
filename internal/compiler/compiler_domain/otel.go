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

package compiler_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// jsThis is the JavaScript keyword that refers to the current object.
	jsThis = "this"

	// jsString is the JavaScript String constructor name used to convert values
	// to strings.
	jsString = "String"

	// identifierUnderscore is the underscore character used to replace special
	// characters in JavaScript identifiers.
	identifierUnderscore = "_"

	// typeArray is the type name for array values.
	typeArray = "array"

	// typeObject is the type name for JavaScript object literals.
	typeObject = "object"

	// propTagName is the property key for a custom element's tag name.
	propTagName = "tagName"

	// propClassName is the property name for the custom element class name.
	propClassName = "className"

	// propHref is the HTML attribute name for link URLs.
	propHref = "href"

	// propState is the property name for reactive state in component instances.
	propState = "state"

	// propInitialState is the property name for initial state in generated code.
	propInitialState = "$$initialState"

	// propGetterName is the logging key for getter method names.
	propGetterName = "getterName"

	// logKeyError is the logging key for error details.
	logKeyError = "error"

	// fmtQuotedString is the format string for wrapping a string in quotes.
	fmtQuotedString = "%q"

	// strSpace is a single space character used for whitespace replacement.
	strSpace = " "

	// jsSemicolon is the semicolon character used to end JavaScript statements.
	jsSemicolon = ";"

	// jsCloseBrace is the closing brace character used to find block endings.
	jsCloseBrace = "}"

	// jsOpenBrace is the opening brace character used in JavaScript blocks.
	jsOpenBrace = "{"
)

var (
	log = logger_domain.GetLogger("piko/internal/compiler/compiler_domain")

	// Meter is the OpenTelemetry meter for the compiler domain package.
	Meter = otel.Meter("piko/internal/compiler/compiler_domain")

	// DecoratorProcessingCount tracks how many times decorators have been
	// processed.
	DecoratorProcessingCount metric.Int64Counter

	// DecoratorProcessingDuration tracks the duration of decorator processing
	// operations.
	DecoratorProcessingDuration metric.Float64Histogram

	// DecoratedVarsCount is a counter that tracks how many variables have
	// been decorated.
	DecoratedVarsCount metric.Int64Counter

	// DecoratedFuncsCount is a counter that tracks how many functions have been
	// decorated.
	DecoratedFuncsCount metric.Int64Counter

	// VDOMBuildCount counts how many times the VDOM has been built.
	VDOMBuildCount metric.Int64Counter

	// VDOMBuildDuration records how long it takes to build the virtual DOM.
	VDOMBuildDuration metric.Float64Histogram

	// VDOMBuildErrorCount counts the number of VDOM build errors.
	VDOMBuildErrorCount metric.Int64Counter

	// NodeBuildCount tracks the number of nodes built during VDOM construction.
	NodeBuildCount metric.Int64Counter

	// NodeBuildErrorCount tracks the number of node building errors during VDOM
	// construction.
	NodeBuildErrorCount metric.Int64Counter

	// ScaffoldBuildCount tracks the number of scaffold build operations.
	ScaffoldBuildCount metric.Int64Counter

	// ScaffoldBuildDuration records how long scaffold building takes.
	ScaffoldBuildDuration metric.Float64Histogram

	// ScaffoldBuildErrorCount counts errors that happen when building scaffolds.
	ScaffoldBuildErrorCount metric.Int64Counter

	// CSSTreeShakingCount counts CSS tree-shaking operations.
	CSSTreeShakingCount metric.Int64Counter

	// CSSTreeShakingDuration records the time spent on CSS tree-shaking.
	CSSTreeShakingDuration metric.Float64Histogram

	// CSSTreeShakingErrorCount counts errors during CSS tree-shaking.
	CSSTreeShakingErrorCount metric.Int64Counter

	// JSParsingCount tracks the number of JavaScript parsing operations.
	JSParsingCount metric.Int64Counter

	// JSParsingDuration tracks the duration of JavaScript parsing operations.
	JSParsingDuration metric.Float64Histogram

	// JSParsingErrorCount tracks the number of JavaScript parsing errors.
	JSParsingErrorCount metric.Int64Counter

	// ImportExtractionCount tracks the number of import extraction operations.
	ImportExtractionCount metric.Int64Counter

	// ImportExtractionDuration tracks the duration of import extraction
	// operations.
	ImportExtractionDuration metric.Float64Histogram

	// ImportExtractionErrorCount counts the number of import extraction errors.
	ImportExtractionErrorCount metric.Int64Counter

	// OrchestratorCompilationCount tracks the number of compilations initiated by
	// the orchestrator.
	OrchestratorCompilationCount metric.Int64Counter

	// OrchestratorCompilationDuration tracks the duration of compilations
	// initiated by the orchestrator.
	OrchestratorCompilationDuration metric.Float64Histogram

	// OrchestratorCompilationErrorCount tracks the number of compilation errors in
	// the orchestrator.
	OrchestratorCompilationErrorCount metric.Int64Counter

	// OrchestratorTransformationCount tracks the number of transformations
	// initiated by the orchestrator.
	OrchestratorTransformationCount metric.Int64Counter

	// OrchestratorTransformationDuration tracks the duration of transformations
	// initiated by the orchestrator.
	OrchestratorTransformationDuration metric.Float64Histogram

	// OrchestratorTransformationErrorCount tracks the number of transformation
	// errors in the orchestrator.
	OrchestratorTransformationErrorCount metric.Int64Counter

	// HTMLExtractionCount tracks how many HTML extraction operations have run.
	HTMLExtractionCount metric.Int64Counter

	// HTMLExtractionDuration tracks the duration of HTML extraction operations.
	HTMLExtractionDuration metric.Float64Histogram

	// HTMLExtractionErrorCount tracks the number of HTML extraction errors.
	HTMLExtractionErrorCount metric.Int64Counter

	// SFCCompilationCount tracks the number of SFC compilations that have
	// happened.
	SFCCompilationCount metric.Int64Counter

	// SFCCompilationDuration records how long SFC compilations take.
	SFCCompilationDuration metric.Float64Histogram

	// SFCCompilationErrorCount tracks the number of single-file component
	// compilation errors.
	SFCCompilationErrorCount metric.Int64Counter

	// ASTTransformationCount counts the number of AST changes made.
	ASTTransformationCount metric.Int64Counter

	// ASTTransformationDuration tracks the duration of AST transformations.
	ASTTransformationDuration metric.Float64Histogram

	// ASTTransformationErrorCount counts errors that occur when transforming the
	// abstract syntax tree.
	ASTTransformationErrorCount metric.Int64Counter

	// MethodInsertionCount counts how many times methods are inserted.
	MethodInsertionCount metric.Int64Counter

	// MethodInsertionDuration tracks the duration of method insertions.
	MethodInsertionDuration metric.Float64Histogram

	// MethodInsertionErrorCount tracks the number of times adding a method fails.
	MethodInsertionErrorCount metric.Int64Counter

	// CSSInsertionCount tracks the number of times CSS has been inserted.
	CSSInsertionCount metric.Int64Counter

	// CSSInsertionDuration records how long CSS insertions take.
	CSSInsertionDuration metric.Float64Histogram

	// CSSInsertionErrorCount tracks the number of CSS insertion errors that have
	// occurred.
	CSSInsertionErrorCount metric.Int64Counter

	// CSSMinificationCount tracks the number of CSS minifications.
	CSSMinificationCount metric.Int64Counter

	// CSSMinificationDuration tracks the duration of CSS minifications.
	CSSMinificationDuration metric.Float64Histogram

	// CSSMinificationErrorCount tracks how many times CSS minification fails.
	CSSMinificationErrorCount metric.Int64Counter

	// ConstructorCreationCount tracks how many constructors have been created.
	ConstructorCreationCount metric.Int64Counter

	// ConstructorStandardisationCount tracks the number of constructors that have
	// been standardised.
	ConstructorStandardisationCount metric.Int64Counter

	// ConnectedCallbackCreationCount tracks the number of connectedCallback
	// methods created.
	ConnectedCallbackCreationCount metric.Int64Counter

	// ConnectedCallbackInjectionCount tracks the number of injections into
	// connectedCallback methods.
	ConnectedCallbackInjectionCount metric.Int64Counter

	// EventBindingCreationCount tracks the number of event bindings created.
	EventBindingCreationCount metric.Int64Counter

	// EventBindingCreationErrorCount tracks the number of errors during event
	// binding creation.
	EventBindingCreationErrorCount metric.Int64Counter

	// EventBindingInjectionCount tracks the number of event bindings injected into
	// constructors.
	EventBindingInjectionCount metric.Int64Counter

	// EventBindingInjectionDuration tracks the duration of event binding injection
	// operations.
	EventBindingInjectionDuration metric.Float64Histogram

	// EventBindingInjectionErrorCount tracks the number of errors during event
	// binding injection.
	EventBindingInjectionErrorCount metric.Int64Counter
)

func init() {
	var err error

	ScaffoldBuildCount, err = Meter.Int64Counter(
		"compiler.scaffold_build_count",
		metric.WithDescription("Number of scaffold building operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ScaffoldBuildDuration, err = Meter.Float64Histogram(
		"compiler.scaffold_build_duration",
		metric.WithDescription("Duration of scaffold building operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ScaffoldBuildErrorCount, err = Meter.Int64Counter(
		"compiler.scaffold_build_error_count",
		metric.WithDescription("Number of scaffold building errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSTreeShakingCount, err = Meter.Int64Counter(
		"compiler.css_tree_shaking_count",
		metric.WithDescription("Number of CSS tree-shaking operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSTreeShakingDuration, err = Meter.Float64Histogram(
		"compiler.css_tree_shaking_duration",
		metric.WithDescription("Duration of CSS tree-shaking operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSTreeShakingErrorCount, err = Meter.Int64Counter(
		"compiler.css_tree_shaking_error_count",
		metric.WithDescription("Number of CSS tree-shaking errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DecoratorProcessingCount, err = Meter.Int64Counter(
		"compiler.decorator_processing_count",
		metric.WithDescription("Number of decorator processing operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DecoratorProcessingDuration, err = Meter.Float64Histogram(
		"compiler.decorator_processing_duration",
		metric.WithDescription("Duration of decorator processing operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DecoratedVarsCount, err = Meter.Int64Counter(
		"compiler.decorated_vars_count",
		metric.WithDescription("Number of decorated variables"),
	)
	if err != nil {
		otel.Handle(err)
	}

	DecoratedFuncsCount, err = Meter.Int64Counter(
		"compiler.decorated_funcs_count",
		metric.WithDescription("Number of decorated functions"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VDOMBuildCount, err = Meter.Int64Counter(
		"compiler.vdom_build_count",
		metric.WithDescription("Number of VDOM building operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VDOMBuildDuration, err = Meter.Float64Histogram(
		"compiler.vdom_build_duration",
		metric.WithDescription("Duration of VDOM building operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	VDOMBuildErrorCount, err = Meter.Int64Counter(
		"compiler.vdom_build_error_count",
		metric.WithDescription("Number of VDOM building errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	NodeBuildCount, err = Meter.Int64Counter(
		"compiler.node_build_count",
		metric.WithDescription("Number of nodes built during VDOM construction"),
	)
	if err != nil {
		otel.Handle(err)
	}

	NodeBuildErrorCount, err = Meter.Int64Counter(
		"compiler.node_build_error_count",
		metric.WithDescription("Number of node building errors during VDOM construction"),
	)
	if err != nil {
		otel.Handle(err)
	}

	JSParsingCount, err = Meter.Int64Counter(
		"compiler.js_parsing_count",
		metric.WithDescription("Number of JavaScript parsing operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	JSParsingDuration, err = Meter.Float64Histogram(
		"compiler.js_parsing_duration",
		metric.WithDescription("Duration of JavaScript parsing operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	JSParsingErrorCount, err = Meter.Int64Counter(
		"compiler.js_parsing_error_count",
		metric.WithDescription("Number of JavaScript parsing errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ImportExtractionCount, err = Meter.Int64Counter(
		"compiler.import_extraction_count",
		metric.WithDescription("Number of import extraction operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ImportExtractionDuration, err = Meter.Float64Histogram(
		"compiler.import_extraction_duration",
		metric.WithDescription("Duration of import extraction operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ImportExtractionErrorCount, err = Meter.Int64Counter(
		"compiler.import_extraction_error_count",
		metric.WithDescription("Number of import extraction errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OrchestratorCompilationCount, err = Meter.Int64Counter(
		"compiler.orchestrator_compilation_count",
		metric.WithDescription("Number of compilations initiated by the orchestrator"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OrchestratorCompilationDuration, err = Meter.Float64Histogram(
		"compiler.orchestrator_compilation_duration",
		metric.WithDescription("Duration of compilations initiated by the orchestrator"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OrchestratorCompilationErrorCount, err = Meter.Int64Counter(
		"compiler.orchestrator_compilation_error_count",
		metric.WithDescription("Number of compilation errors in the orchestrator"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OrchestratorTransformationCount, err = Meter.Int64Counter(
		"compiler.orchestrator_transformation_count",
		metric.WithDescription("Number of transformations initiated by the orchestrator"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OrchestratorTransformationDuration, err = Meter.Float64Histogram(
		"compiler.orchestrator_transformation_duration",
		metric.WithDescription("Duration of transformations initiated by the orchestrator"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	OrchestratorTransformationErrorCount, err = Meter.Int64Counter(
		"compiler.orchestrator_transformation_error_count",
		metric.WithDescription("Number of transformation errors in the orchestrator"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HTMLExtractionCount, err = Meter.Int64Counter(
		"compiler.html_extraction_count",
		metric.WithDescription("Number of HTML extraction operations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HTMLExtractionDuration, err = Meter.Float64Histogram(
		"compiler.html_extraction_duration",
		metric.WithDescription("Duration of HTML extraction operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	HTMLExtractionErrorCount, err = Meter.Int64Counter(
		"compiler.html_extraction_error_count",
		metric.WithDescription("Number of HTML extraction errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SFCCompilationCount, err = Meter.Int64Counter(
		"compiler.sfc_compilation_count",
		metric.WithDescription("Number of SFC compilations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SFCCompilationDuration, err = Meter.Float64Histogram(
		"compiler.sfc_compilation_duration",
		metric.WithDescription("Duration of SFC compilations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SFCCompilationErrorCount, err = Meter.Int64Counter(
		"compiler.sfc_compilation_error_count",
		metric.WithDescription("Number of SFC compilation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ASTTransformationCount, err = Meter.Int64Counter(
		"compiler.ast_transformation_count",
		metric.WithDescription("Number of AST transformations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ASTTransformationDuration, err = Meter.Float64Histogram(
		"compiler.ast_transformation_duration",
		metric.WithDescription("Duration of AST transformations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ASTTransformationErrorCount, err = Meter.Int64Counter(
		"compiler.ast_transformation_error_count",
		metric.WithDescription("Number of AST transformation errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	MethodInsertionCount, err = Meter.Int64Counter(
		"compiler.method_insertion_count",
		metric.WithDescription("Number of method insertions"),
	)
	if err != nil {
		otel.Handle(err)
	}

	MethodInsertionDuration, err = Meter.Float64Histogram(
		"compiler.method_insertion_duration",
		metric.WithDescription("Duration of method insertions"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	MethodInsertionErrorCount, err = Meter.Int64Counter(
		"compiler.method_insertion_error_count",
		metric.WithDescription("Number of method insertion errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ConstructorCreationCount, err = Meter.Int64Counter(
		"compiler.constructor_creation_count",
		metric.WithDescription("Number of constructors created"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ConstructorStandardisationCount, err = Meter.Int64Counter(
		"compiler.constructor_standardisation_count",
		metric.WithDescription("Number of constructors standardised"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ConnectedCallbackCreationCount, err = Meter.Int64Counter(
		"compiler.connected_callback_creation_count",
		metric.WithDescription("Number of connectedCallback methods created"),
	)
	if err != nil {
		otel.Handle(err)
	}

	ConnectedCallbackInjectionCount, err = Meter.Int64Counter(
		"compiler.connected_callback_injection_count",
		metric.WithDescription("Number of injections into connectedCallback methods"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSInsertionCount, err = Meter.Int64Counter(
		"compiler.css_insertion_count",
		metric.WithDescription("Number of CSS insertions"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSInsertionDuration, err = Meter.Float64Histogram(
		"compiler.css_insertion_duration",
		metric.WithDescription("Duration of CSS insertions"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSInsertionErrorCount, err = Meter.Int64Counter(
		"compiler.css_insertion_error_count",
		metric.WithDescription("Number of CSS insertion errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSMinificationCount, err = Meter.Int64Counter(
		"compiler.css_minification_count",
		metric.WithDescription("Number of CSS minifications"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSMinificationDuration, err = Meter.Float64Histogram(
		"compiler.css_minification_duration",
		metric.WithDescription("Duration of CSS minifications"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	CSSMinificationErrorCount, err = Meter.Int64Counter(
		"compiler.css_minification_error_count",
		metric.WithDescription("Number of CSS minification errors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	EventBindingCreationCount, err = Meter.Int64Counter(
		"compiler.event_binding_creation_count",
		metric.WithDescription("Number of event bindings created"),
	)
	if err != nil {
		otel.Handle(err)
	}

	EventBindingCreationErrorCount, err = Meter.Int64Counter(
		"compiler.event_binding_creation_error_count",
		metric.WithDescription("Number of errors during event binding creation"),
	)
	if err != nil {
		otel.Handle(err)
	}

	EventBindingInjectionCount, err = Meter.Int64Counter(
		"compiler.event_binding_injection_count",
		metric.WithDescription("Number of event bindings injected into constructors"),
	)
	if err != nil {
		otel.Handle(err)
	}

	EventBindingInjectionDuration, err = Meter.Float64Histogram(
		"compiler.event_binding_injection_duration",
		metric.WithDescription("Duration of event binding injection operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	EventBindingInjectionErrorCount, err = Meter.Int64Counter(
		"compiler.event_binding_injection_error_count",
		metric.WithDescription("Number of errors during event binding injection"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
