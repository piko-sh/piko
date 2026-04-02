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
	"strings"
	"time"

	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
)

// minifyCSS reduces the size of a CSS block using the given parser and printer
// settings.
//
// Takes cssBlock (string) which is the raw CSS content to make smaller.
// Takes parserOptions (css_parser.Options) which controls how CSS is parsed.
// Takes options (css_printer.Options) which controls how CSS output is written.
//
// Returns string which is the smaller CSS, or empty if input was empty.
// Returns error when CSS minification fails.
func minifyCSS(ctx context.Context, cssBlock string, parserOptions css_parser.Options, options css_printer.Options) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "minifyCSS",
		logger_domain.Int("cssLength", len(cssBlock)),
	)
	defer span.End()

	trimmed := strings.TrimSpace(cssBlock)
	if trimmed == "" {
		l.Trace("Empty CSS block, returning empty string")
		return "", nil
	}

	CSSMinificationCount.Add(ctx, 1)
	startTime := time.Now()

	result, err := executeCSSMinification(ctx, trimmed, parserOptions, options)

	minificationDuration := time.Since(startTime)
	CSSMinificationDuration.Record(ctx, float64(minificationDuration.Milliseconds()))

	if err != nil {
		l.ReportError(span, err, "CSS minification failed")
		CSSMinificationErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("minifying CSS: %w", err)
	}

	l.Trace("CSS minification successful",
		logger_domain.Int("originalLength", len(trimmed)),
		logger_domain.Int("minifiedLength", len(result)),
		logger_domain.Int64("durationMs", minificationDuration.Milliseconds()),
	)
	return result, nil
}

// executeCSSMinification parses and minifies CSS content.
//
// Takes css (string) which is the CSS content to minify.
// Takes parserOptions (css_parser.Options) which sets up the CSS parser.
// Takes options (css_printer.Options) which sets up the output format.
//
// Returns string which is the minified CSS content.
// Returns error when parsing fails.
func executeCSSMinification(_ context.Context, css string, parserOptions css_parser.Options, options css_printer.Options) (string, error) {
	src := createCSSSource(css)
	esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
	tree := css_parser.Parse(esLog, src, parserOptions)

	if err := checkCSSParseErrors(esLog); err != nil {
		return "", fmt.Errorf("parsing CSS: %w", err)
	}

	return printCSS(tree, options), nil
}

// createCSSSource creates an esbuild source for CSS parsing.
//
// Takes css (string) which contains the CSS content to wrap.
//
// Returns es_logger.Source which is ready for esbuild parsing.
func createCSSSource(css string) es_logger.Source {
	const sourceIndex uint32 = 0
	return es_logger.Source{
		Index:    sourceIndex,
		KeyPath:  es_logger.Path{Text: "inline-style.css"},
		Contents: css,
	}
}

// checkCSSParseErrors checks for CSS parse errors and returns an error if any
// are found.
//
// Takes esLog (es_logger.Log) which holds the log messages from esbuild.
//
// Returns error when CSS parse errors are found in the log.
func checkCSSParseErrors(esLog es_logger.Log) error {
	messages := esLog.Done()
	if len(messages) == 0 {
		return nil
	}

	var errs []string
	for _, m := range messages {
		if m.Kind == es_logger.Error {
			errs = append(errs, m.String(es_logger.OutputOptions{}, es_logger.TerminalInfo{}))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("esbuild CSS parsing failed: %s", strings.Join(errs, "\n"))
	}
	return nil
}

// printCSS renders a CSS syntax tree to a string.
//
// Takes tree (css_ast.AST) which is the parsed CSS syntax tree.
// Takes options (css_printer.Options) which controls the output formatting.
//
// Returns string which is the rendered CSS text.
func printCSS(tree css_ast.AST, options css_printer.Options) string {
	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	printed := css_printer.Print(tree, symMap, options)
	return string(printed.CSS)
}
