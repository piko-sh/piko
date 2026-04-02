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
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
)

// TypeScriptParser wraps esbuild's TypeScript parser for use in the compiler.
type TypeScriptParser struct {
	// log provides structured logging for the parser.
	log logger.Log
}

// NewTypeScriptParser creates a new TypeScript parser with deferred logging.
//
// Returns *TypeScriptParser which is ready to parse TypeScript code snippets.
func NewTypeScriptParser() *TypeScriptParser {
	return &TypeScriptParser{
		log: logger.NewDeferLog(logger.DeferLogAll, nil),
	}
}

// ParseTypeScript parses TypeScript code using esbuild's parser. This properly
// handles TypeScript syntax including type assertions (as Type).
//
// Deferred log errors (e.g. "super" used outside a class method) are ignored.
// Use ParseTypeScriptStrict for user-authored scripts where such errors should
// surface.
//
// Takes source (string) which contains the TypeScript source code to parse.
// Takes filename (string) which identifies the file for error messages.
//
// Returns *js_ast.AST which is the parsed abstract syntax tree.
// Returns error when parsing fails due to a fatal lexer panic.
func (*TypeScriptParser) ParseTypeScript(source string, filename string) (*js_ast.AST, error) {
	return parseTypeScript(source, filename, false)
}

// ParseTypeScriptStrict parses TypeScript code and treats deferred parser
// errors as failures. Use this for user-authored scripts where errors like
// duplicate variable declarations must be caught at compile time rather than
// silently producing broken output.
//
// Takes source (string) which contains the TypeScript source code to parse.
// Takes filename (string) which identifies the file for error messages.
//
// Returns *js_ast.AST which is the parsed abstract syntax tree.
// Returns error when parsing fails or the parser logged any errors.
func (*TypeScriptParser) ParseTypeScriptStrict(source string, filename string) (*js_ast.AST, error) {
	return parseTypeScript(source, filename, true)
}

// parserOptions creates parser settings for TypeScript parsing.
//
// Returns js_parser.Options which holds the parser settings.
func parserOptions() js_parser.Options {
	return js_parser.OptionsFromConfig(&config.Options{
		TS: config.TSOptions{
			Parse: true,
		},
	})
}

// parseTypeScript is the shared implementation for lenient and strict parsing.
//
// When strict is true, the deferred log is checked for errors even when the
// parser returned ok=true. This catches issues like duplicate declarations
// where esbuild logs an error but does not panic.
//
// Takes source (string) which contains the TypeScript source code to parse.
// Takes filename (string) which identifies the file for error messages.
// Takes strict (bool) which enables strict mode where deferred parser errors
// cause failure.
//
// Returns *js_ast.AST which is the parsed abstract syntax tree.
// Returns error when parsing fails or, in strict mode, when the parser logged
// any errors.
func parseTypeScript(source string, filename string, strict bool) (*js_ast.AST, error) {
	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	result, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: filename},
			PrettyPaths:    logger.PrettyPaths{Rel: filename, Abs: filename},
			Contents:       source,
			IdentifierName: filename,
		},
		parserOptions(),
	)

	if !ok {
		messages := parseLog.Done()
		if len(messages) > 0 {
			return nil, fmt.Errorf("typescript parsing failed: %s", formatParserError(&messages[0]))
		}
		return nil, fmt.Errorf("typescript parsing failed for %s", filename)
	}

	if strict && parseLog.HasErrors() {
		messages := parseLog.Done()
		var errs []string
		for _, message := range messages {
			if message.Kind == logger.Error {
				errs = append(errs, formatParserError(&message))
			}
		}
		if len(errs) > 0 {
			return nil, fmt.Errorf("typescript parsing errors in %s: %s", filename, strings.Join(errs, "; "))
		}
	}

	return &result, nil
}

// formatParserError formats an esbuild parser message into a human-readable
// string that includes location information when available. When the source
// line text is present, a diagnostic snippet with a caret is appended.
//
// Takes message (*logger.Msg) which holds the parser error
// message and optional source location.
//
// Returns string which is the formatted error with location and snippet when
// available.
func formatParserError(message *logger.Msg) string {
	location := message.Data.Location
	if location == nil || location.Line == 0 {
		return message.Data.Text
	}

	header := fmt.Sprintf("%s (line %d, col %d)", message.Data.Text, location.Line, location.Column)

	if location.LineText == "" {
		return header
	}

	lineNum := strconv.Itoa(location.Line)
	gutter := strings.Repeat(" ", len(lineNum))

	return fmt.Sprintf("%s\n    %s | %s\n    %s | %s^",
		header,
		lineNum, location.LineText,
		gutter, strings.Repeat(" ", location.Column))
}
