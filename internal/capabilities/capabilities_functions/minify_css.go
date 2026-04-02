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

package capabilities_functions

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/compat"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
)

// MinifyCSS returns a capability function that minifies CSS content using
// esbuild with support for CSS nesting lowering for broader browser
// compatibility.
//
// Returns capabilities_domain.CapabilityFunc which processes CSS input and
// returns minified output.
func MinifyCSS() capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, "MinifyCSS",
			logger_domain.String(logger_domain.KeyReference, "capability"),
		)
		defer span.End()

		l.Trace("Minifying CSS with esbuild (including nesting lowering)")

		if err := ctx.Err(); err != nil {
			l.Warn("Context cancelled during execution setup", logger_domain.Error(err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Context cancelled")
			return nil, fmt.Errorf("context cancelled before CSS minification: %w", err)
		}

		minifiedBytes, duration, err := executeCSSMinification(ctx, inputData)
		if err != nil {
			l.ReportError(span, err, "Failed to minify CSS")
			return nil, fmt.Errorf("executing CSS minification: %w", err)
		}

		recordCSSMinificationSuccess(ctx, span, minifiedBytes, duration)
		return bytes.NewReader(minifiedBytes), nil
	}
}

// executeCSSMinification parses and minifies CSS content within a tracing span.
//
// Takes inputData (io.Reader) which supplies the CSS content to minify.
//
// Returns []byte which contains the minified CSS output.
// Returns time.Duration which shows how long the minification took.
// Returns error when the input cannot be read.
func executeCSSMinification(ctx context.Context, inputData io.Reader) ([]byte, time.Duration, error) {
	ctx, l := logger_domain.From(ctx, log)
	var minifiedBytes []byte
	var duration time.Duration

	err := l.RunInSpan(ctx, "MinifyCSS", func(_ context.Context, _ logger_domain.Logger) error {
		startTime := time.Now()

		inputBytes, err := io.ReadAll(inputData)
		if err != nil {
			return fmt.Errorf("reading CSS input data: %w", err)
		}

		minifiedBytes = parseAndMinifyCSS(inputBytes)
		duration = time.Since(startTime)
		return nil
	})

	return minifiedBytes, duration, err
}

// parseAndMinifyCSS parses and minifies CSS using esbuild with nesting
// lowering.
//
// Takes inputBytes ([]byte) which contains the raw CSS to parse and minify.
//
// Returns []byte which contains the minified CSS output.
func parseAndMinifyCSS(inputBytes []byte) []byte {
	esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
	source := createCSSSource(inputBytes)

	tree := parseCSSTree(esLog, source)
	return printMinifiedCSS(tree)
}

// createCSSSource creates an esbuild Source from the input bytes.
//
// Takes inputBytes ([]byte) which contains the raw CSS content.
//
// Returns es_logger.Source which wraps the CSS content for esbuild to process.
func createCSSSource(inputBytes []byte) es_logger.Source {
	return es_logger.Source{
		KeyPath: es_logger.Path{
			Text:             "asset.css",
			Namespace:        "",
			IgnoredSuffix:    "",
			ImportAttributes: es_logger.ImportAttributes{},
			Flags:            0,
		},
		Contents: string(inputBytes),
	}
}

// parseCSSTree parses the CSS source with nesting lowering enabled.
//
// Takes esLog (es_logger.Log) which receives parser messages and warnings.
// Takes source (es_logger.Source) which provides the CSS content to parse.
//
// Returns css_ast.AST which is the parsed and minified CSS syntax tree.
func parseCSSTree(esLog es_logger.Log, source es_logger.Source) css_ast.AST {
	cssConfig := config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}
	parserOpts := css_parser.OptionsFromConfig(config.LoaderCSS, &cssConfig)
	return css_parser.Parse(esLog, source, parserOpts)
}

// printMinifiedCSS outputs the CSS tree as minified bytes.
//
// Takes tree (css_ast.AST) which is the parsed CSS syntax tree to minify.
//
// Returns []byte which contains the minified CSS output.
func printMinifiedCSS(tree css_ast.AST) []byte {
	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	printOpts := css_printer.Options{
		MinifyWhitespace: true,
		ASCIIOnly:        true,
	}
	printed := css_printer.Print(tree, symMap, printOpts)
	return printed.CSS
}

// recordCSSMinificationSuccess records metrics and span attributes for a
// successful CSS minification.
//
// Takes span (trace.Span) which receives duration and status attributes.
// Takes minifiedBytes ([]byte) which provides the output size for metrics.
// Takes duration (time.Duration) which records how long minification took.
func recordCSSMinificationSuccess(ctx context.Context, span trace.Span, minifiedBytes []byte, duration time.Duration) {
	ctx, l := logger_domain.From(ctx, log)
	minificationDuration.Record(ctx, float64(duration.Milliseconds()))
	span.SetAttributes(
		attribute.Int64("durationMs", duration.Milliseconds()),
		attribute.String("status", "minified"),
		attribute.Int("outputSize", len(minifiedBytes)),
	)
	span.SetStatus(codes.Ok, "CSS minified successfully")
	l.Trace("CSS minified successfully with nesting lowering", logger_domain.Int("outputSize", len(minifiedBytes)))
}
