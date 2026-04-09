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

package generator_domain

import (
	"context"
	"fmt"

	esbuildast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/js_printer"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/esbuild/renamer"
	"piko.sh/piko/internal/jsimport"
)

// TranspileOptions configures the TypeScript to JavaScript transpilation.
type TranspileOptions struct {
	// Filename is the source file path used in error messages and source maps.
	Filename string

	// ModuleName is the Go module name for @/ alias resolution. When set,
	// import paths starting with @/ are rewritten to served asset paths.
	ModuleName string

	// Minify enables shorter syntax and removes whitespace when true.
	Minify bool
}

// TranspileResult contains the transpiled JavaScript code.
type TranspileResult struct {
	// Code contains the transpiled JavaScript output.
	Code string
}

// JSTranspiler transpiles TypeScript to JavaScript using esbuild's parser
// and printer. It strips type annotations while preserving runtime code.
type JSTranspiler struct{}

// NewJSTranspiler creates a new JSTranspiler instance.
//
// Returns *JSTranspiler which is ready to convert TypeScript
// code.
func NewJSTranspiler() *JSTranspiler {
	return &JSTranspiler{}
}

// Transpile converts TypeScript source code to JavaScript.
//
// It parses the TypeScript, removes type annotations, and outputs clean
// JavaScript.
//
// Takes source (string) which is the TypeScript source code to convert.
// Takes opts (TranspileOptions) which controls the transpilation behaviour.
//
// Returns *TranspileResult which contains the generated JavaScript code.
// Returns error when parsing the TypeScript source fails.
func (*JSTranspiler) Transpile(_ context.Context, source string, opts TranspileOptions) (*TranspileResult, error) {
	if source == "" {
		return &TranspileResult{Code: ""}, nil
	}

	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	tree, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: opts.Filename},
			PrettyPaths:    logger.PrettyPaths{Rel: opts.Filename, Abs: opts.Filename},
			Contents:       source,
			IdentifierName: opts.Filename,
		},
		parserOptions(),
	)

	if !ok {
		messages := parseLog.Done()
		if len(messages) > 0 {
			return nil, fmt.Errorf("typescript parsing failed: %s", messages[0].Data.Text)
		}
		return nil, fmt.Errorf("typescript parsing failed for %s", opts.Filename)
	}

	jsimport.RewriteImportRecords(tree.ImportRecords, opts.ModuleName)

	symbols := esbuildast.NewSymbolMap(1)
	symbols.SymbolsForSource[0] = tree.Symbols

	r := renamer.NewNoOpRenamer(symbols)

	printerOpts := js_printer.Options{
		MinifyWhitespace: opts.Minify,
		MinifySyntax:     opts.Minify,
		OutputFormat:     config.FormatESModule,
	}

	result := js_printer.Print(tree, symbols, r, printerOpts)

	return &TranspileResult{
		Code: string(result.JS),
	}, nil
}

// parserOptions creates parser settings for TypeScript files.
//
// Returns js_parser.Options which holds the settings for parsing TypeScript.
func parserOptions() js_parser.Options {
	return js_parser.OptionsFromConfig(&config.Options{
		TS: config.TSOptions{
			Parse: true,
		},
	})
}
