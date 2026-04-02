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
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/esbuild/compat"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/logger/logger_domain"
)

// InsertStaticCSS inserts minified CSS as a static getter on the component
// class.
//
// Takes fullSyntaxTree (*js_ast.AST) which is the parsed JavaScript AST to
// modify.
// Takes cascadingStyleSheet (string) which is the CSS content to minify and
// insert.
// Takes targetClassName (string) which identifies the class to add the getter
// to.
//
// Returns error when CSS minification fails, getter creation fails, or the
// target class cannot be found.
func InsertStaticCSS(
	executionContext context.Context,
	fullSyntaxTree *js_ast.AST,
	cascadingStyleSheet string,
	targetClassName string,
) error {
	executionContext, l := logger_domain.From(executionContext, log)
	ctx, span, l := l.Span(executionContext, "InsertStaticCSS",
		logger_domain.String("targetClassName", targetClassName),
		logger_domain.Int("cssLength", len(cascadingStyleSheet)),
	)
	defer span.End()

	if cascadingStyleSheet == "" {
		l.Trace("Empty CSS, skipping insertion")
		return nil
	}

	CSSInsertionCount.Add(ctx, 1)
	startTime := time.Now()

	minifiedStyles, err := minifyCSSWithMetrics(ctx, span, cascadingStyleSheet)
	if err != nil {
		CSSInsertionErrorCount.Add(ctx, 1)
		return fmt.Errorf("minifying CSS: %w", err)
	}

	staticGetterFunction, err := createCSSGetterWithMetrics(ctx, span, minifiedStyles)
	if err != nil {
		CSSInsertionErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating CSS getter: %w", err)
	}

	if err := insertGetterIntoClass(ctx, span, fullSyntaxTree, targetClassName, staticGetterFunction); err != nil {
		CSSInsertionErrorCount.Add(ctx, 1)
		return fmt.Errorf("inserting getter into class: %w", err)
	}

	recordCSSInsertionMetrics(ctx, span, startTime, targetClassName, len(minifiedStyles))
	return nil
}

// minifyCSSWithMetrics minifies CSS and records timing metrics.
//
// Takes span (trace.Span) which receives timing attributes for the operation.
// Takes css (string) which contains the raw CSS to minify.
//
// Returns string which is the minified CSS output.
// Returns error when CSS minification fails.
func minifyCSSWithMetrics(ctx context.Context, span trace.Span, css string) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	CSSMinificationCount.Add(ctx, 1)
	minifyStartTime := time.Now()

	minifiedStyles, err := minifyCSS(ctx, css, css_parser.OptionsFromConfig(config.LoaderCSS, &config.Options{
		UnsupportedCSSFeatures: compat.Nesting,
	}), css_printer.Options{
		MinifyWhitespace: true,
		ASCIIOnly:        true,
	})

	minifyDuration := time.Since(minifyStartTime)
	CSSMinificationDuration.Record(ctx, float64(minifyDuration.Milliseconds()))
	span.SetAttributes(attribute.Int64("minifyDurationMs", minifyDuration.Milliseconds()))

	if err != nil {
		CSSMinificationErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "CSS minification failed")
		return "", fmt.Errorf("minifying CSS content: %w", err)
	}

	return minifiedStyles, nil
}

// createCSSGetterWithMetrics creates a getter property for static CSS content.
//
// Takes span (trace.Span) which provides tracing context.
// Takes minifiedCSS (string) which contains the CSS content to embed.
//
// Returns *js_ast.Property which is the getter property for the CSS.
// Returns error when the static getter function cannot be created.
func createCSSGetterWithMetrics(ctx context.Context, span trace.Span, minifiedCSS string) (*js_ast.Property, error) {
	ctx, l := logger_domain.From(ctx, log)
	staticGetterFunction, err := createStaticGetterFunction("css", minifiedCSS)
	if err != nil {
		l.ReportError(span, err, "Failed to create static getter for CSS")
		return nil, fmt.Errorf("unable to create static getter for CSS: %w", err)
	}
	return staticGetterFunction, nil
}

// insertGetterIntoClass finds the target class and adds the getter property.
//
// Takes span (trace.Span) which provides tracing for the operation.
// Takes ast (*js_ast.AST) which is the JavaScript AST to modify.
// Takes className (string) which is the name of the target class.
// Takes getter (*js_ast.Property) which is the getter property to add.
//
// Returns error when the named class cannot be found in the AST.
func insertGetterIntoClass(ctx context.Context, span trace.Span, ast *js_ast.AST, className string, getter *js_ast.Property) error {
	ctx, l := logger_domain.From(ctx, log)
	targetClassDeclaration := findClassDeclarationByName(ast, className)
	if targetClassDeclaration == nil {
		err := fmt.Errorf("class named %q not found for static CSS insertion", className)
		l.ReportError(span, err, "Target class not found")
		return err
	}
	targetClassDeclaration.Properties = append(targetClassDeclaration.Properties, *getter)
	return nil
}

// recordCSSInsertionMetrics records metrics after CSS insertion and logs the
// result.
//
// Takes span (trace.Span) which receives duration and length attributes.
// Takes startTime (time.Time) which marks when the insertion started.
// Takes className (string) which is the component class that
// received the CSS insertion.
// Takes cssLength (int) which gives the length of the minified CSS.
func recordCSSInsertionMetrics(ctx context.Context, span trace.Span, startTime time.Time, className string, cssLength int) {
	ctx, l := logger_domain.From(ctx, log)
	insertionDuration := time.Since(startTime)
	CSSInsertionDuration.Record(ctx, float64(insertionDuration.Milliseconds()))
	span.SetAttributes(
		attribute.Int64("insertionDurationMs", insertionDuration.Milliseconds()),
		attribute.Int("minifiedCssLength", cssLength),
	)
	l.Trace("Inserted static CSS getter",
		logger_domain.String("className", className),
		logger_domain.Int("cssLength", cssLength),
	)
	span.SetStatus(codes.Ok, "CSS insertion successful")
}
