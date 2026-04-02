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

	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/logger/logger_domain"
)

// ParseJSResult holds the output from parsing a JavaScript or TypeScript script.
type ParseJSResult struct {
	// AST holds the parsed JavaScript abstract syntax tree.
	AST *js_ast.AST

	// TypeAssertions maps type names to their assertion definitions.
	TypeAssertions map[string]TypeAssertion

	// Imports lists the import statements found in the parsed JavaScript.
	Imports []string
}

// ParseUserScript parses a TypeScript or JavaScript script and extracts type
// assertions.
//
// When the script code is empty, returns an empty AST with no assertions.
//
// Takes scriptCode (string) which is the TypeScript or JavaScript source code.
// Takes filename (string) which identifies the source file for error messages.
//
// Returns *ParseJSResult which contains the parsed AST and type assertions.
// Returns error when the TypeScript parsing fails.
func ParseUserScript(
	ctx context.Context,
	scriptCode string,
	filename string,
) (*ParseJSResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ParseUserScript",
		logger_domain.Int("scriptLength", len(scriptCode)),
		logger_domain.String("filename", filename),
	)
	defer span.End()

	if scriptCode == "" {
		l.Trace("Empty script code, returning empty AST")
		return &ParseJSResult{
			AST:            &js_ast.AST{},
			TypeAssertions: nil,
			Imports:        nil,
		}, nil
	}

	typeAssertions := extractTypeAssertionsWithSpan(ctx, scriptCode)

	ast, err := parseTypeScriptWithMetrics(ctx, scriptCode, filename)

	if err != nil {
		return nil, fmt.Errorf("typescript parse error: %w", err)
	}

	l.Trace("TypeScript parsing completed successfully",
		logger_domain.Int("typeAssertions", len(typeAssertions)),
	)

	return &ParseJSResult{
		AST:            ast,
		Imports:        nil,
		TypeAssertions: typeAssertions,
	}, nil
}

// extractTypeAssertionsWithSpan extracts type assertions within a tracing span.
//
// Takes script (string) which contains the JavaScript source to parse.
//
// Returns map[string]TypeAssertion which maps assertion names to their
// definitions.
func extractTypeAssertionsWithSpan(ctx context.Context, script string) map[string]TypeAssertion {
	ctx, l := logger_domain.From(ctx, log)
	_ = ctx
	typeAssertions := ExtractTypeAssertions(script)
	l.Trace("Type assertions extracted", logger_domain.Int("count", len(typeAssertions)))
	return typeAssertions
}

// parseTypeScriptWithMetrics parses TypeScript code and records metrics.
//
// Takes script (string) which contains the TypeScript source code to parse.
// Takes filename (string) which identifies the source file for error messages.
//
// Returns *js_ast.AST which is the parsed abstract syntax tree.
// Returns error when the TypeScript parsing fails.
func parseTypeScriptWithMetrics(ctx context.Context, script, filename string) (*js_ast.AST, error) {
	ctx, l := logger_domain.From(ctx, log)
	JSParsingCount.Add(ctx, 1)
	startTime := time.Now()
	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScriptStrict(script, filename)
	JSParsingDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))

	if err != nil {
		l.Trace("TypeScript parsing failed", logger_domain.Error(err), logger_domain.Int("codeLength", len(script)))
		JSParsingErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("parsing TypeScript: %w", err)
	}
	return parsedAST, nil
}
