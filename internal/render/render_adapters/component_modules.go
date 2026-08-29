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

package render_adapters

import (
	"context"
	"strings"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// maxComponentJSBytes caps the compiled component source this will parse. The parser is
	// recursive descent with no depth limit of its own, so an unbounded input can exhaust
	// the goroutine stack, which no recover can catch.
	maxComponentJSBytes = 4 << 20

	// componentModuleComponent names this work in panic recovery reports.
	componentModuleComponent = "render_adapters.extractRequiredModules"
)

// extractRequiredModules lists the application library modules a compiled component
// imports statically.
//
// Takes componentJS (string) which is the compiled component module.
//
// Returns []string which contains the module URLs in source order, or nil when the source
// cannot be parsed.
func extractRequiredModules(ctx context.Context, componentJS string) []string {
	if componentJS == "" {
		return nil
	}

	if len(componentJS) > maxComponentJSBytes {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Component JavaScript is too large to scan for module preloads",
			logger_domain.Int("bytes", len(componentJS)),
			logger_domain.Int("limit", maxComponentJSBytes))

		return nil
	}

	tree, ok := parseComponentJS(ctx, componentJS)
	if !ok {
		return nil
	}

	prefix := assetpath.DefaultServePath + "/"

	modules := make([]string, 0, len(tree.ImportRecords))
	seen := make(map[string]struct{}, len(tree.ImportRecords))

	for _, record := range tree.ImportRecords {
		if record.Kind != ast.ImportStmt {
			continue
		}
		if !strings.HasPrefix(record.Path.Text, prefix) {
			continue
		}
		if _, duplicate := seen[record.Path.Text]; duplicate {
			continue
		}
		seen[record.Path.Text] = struct{}{}
		modules = append(modules, record.Path.Text)
	}

	if len(modules) == 0 {
		return nil
	}

	return modules
}

// parseComponentJS parses compiled component source without letting a parser panic
// escape.
//
// Takes componentJS (string) which is the compiled component module.
//
// Returns *js_ast.AST which is the parsed tree.
// Returns bool which is false when the source could not be parsed.
func parseComponentJS(ctx context.Context, componentJS string) (*js_ast.AST, bool) {
	type parseResult struct {
		tree *js_ast.AST
		ok   bool
	}

	result := goroutine.SafeCallValue(ctx, componentModuleComponent, func() parseResult {
		parseLog := logger.NewDeferLog(logger.DeferLogNoVerboseOrDebug, nil)
		tree, ok := js_parser.Parse(
			parseLog,
			logger.Source{
				Index:    0,
				KeyPath:  logger.Path{Text: "component.js"},
				Contents: componentJS,
			},
			js_parser.OptionsFromConfig(&config.Options{}),
		)

		return parseResult{tree: &tree, ok: ok}
	})
	if !result.ok || result.tree == nil {
		return nil, false
	}

	return result.tree, true
}
