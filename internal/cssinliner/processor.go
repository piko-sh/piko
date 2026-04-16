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

package cssinliner

import (
	"context"
	"strings"
	"time"

	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// ProcessorConfig holds the configuration for creating a Processor.
type ProcessorConfig struct {
	// Resolver resolves CSS import paths.
	Resolver resolver_domain.ResolverPort

	// Options controls CSS output formatting (minification, etc.).
	Options *config.Options

	// DiagnosticCode is assigned to diagnostics generated during processing
	// (e.g. "T114" for import errors, "T115" for processing errors).
	DiagnosticCode string

	// Loader configures the CSS parser mode (e.g. LoaderLocalCSS).
	Loader config.Loader
}

// Processor provides CSS @import inlining and optional minification.
type Processor struct {
	// resolver resolves CSS import paths.
	resolver resolver_domain.ResolverPort

	// options controls CSS output formatting.
	options *config.Options

	// diagnosticCode is assigned to generated diagnostics.
	diagnosticCode string

	// parserOptions holds the CSS parser settings.
	parserOptions css_parser.Options
}

// NewProcessor creates a new CSS processor with the given settings.
//
// Takes cfg (ProcessorConfig) which provides the processor configuration.
//
// Returns *Processor which is ready to use.
func NewProcessor(cfg ProcessorConfig) *Processor {
	options := cfg.Options
	if options == nil {
		options = &config.Options{}
	}
	return &Processor{
		resolver:       cfg.Resolver,
		options:        options,
		parserOptions:  css_parser.OptionsFromConfig(cfg.Loader, options),
		diagnosticCode: cfg.DiagnosticCode,
	}
}

// SetResolver updates the resolver used for @import resolution.
// This is used by the LSP to provide per-module resolvers.
//
// Takes resolver (resolver_domain.ResolverPort) which provides path resolution.
func (p *Processor) SetResolver(resolver resolver_domain.ResolverPort) {
	p.resolver = resolver
}

// WithResolver returns a shallow copy of the Processor that uses the given
// resolver. This is safe for concurrent use because the copy does not share
// mutable state with the original.
//
// Takes resolver (resolver_domain.ResolverPort) which provides path resolution.
//
// Returns *Processor which is a new processor with the given resolver.
func (p *Processor) WithResolver(resolver resolver_domain.ResolverPort) *Processor {
	return &Processor{
		resolver:       resolver,
		options:        p.options,
		parserOptions:  p.parserOptions,
		diagnosticCode: p.diagnosticCode,
	}
}

// GetOptions returns the processor's output options.
//
// Returns *config.Options which controls CSS output formatting.
func (p *Processor) GetOptions() *config.Options {
	return p.options
}

// GetParserOptions returns the processor's CSS parser options.
//
// Returns css_parser.Options which configures the CSS parser.
func (p *Processor) GetParserOptions() css_parser.Options {
	return p.parserOptions
}

// GetResolver returns the processor's resolver.
//
// Returns resolver_domain.ResolverPort which resolves CSS import paths.
func (p *Processor) GetResolver() resolver_domain.ResolverPort {
	return p.resolver
}

// Process takes a raw CSS string, resolves all @import rules, cleans the AST,
// and returns the final printed result. This is the string-in/string-out API.
//
// Takes cssBlock (string) which contains the raw CSS to process.
// Takes sourcePath (string) which identifies the source file for diagnostics.
// Takes startLocation (ast_domain.Location) which marks the CSS block origin.
// Takes fsReader (FSReaderPort) which reads imported files from the filesystem.
//
// Returns string which is the processed and bundled CSS output.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors found.
// Returns error when processing fails.
func (p *Processor) Process(
	ctx context.Context,
	cssBlock string,
	sourcePath string,
	startLocation ast_domain.Location,
	fsReader FSReaderPort,
) (string, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CSSProcessor.Process")
	defer span.End()

	ProcessCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ProcessDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	trimmed := strings.TrimSpace(cssBlock)
	if trimmed == "" {
		return "", nil, nil
	}

	inliner := GetInliner(p.resolver, p.parserOptions, fsReader, p.diagnosticCode)
	defer PutInliner(inliner)
	tree, inlinerDiags := inliner.InlineAndParse(ctx, trimmed, sourcePath, startLocation)

	if tree == nil {
		ProcessErrorCount.Add(ctx, 1)
		l.Error("Failed to process CSS with fatal error during import resolution")
		return "", inlinerDiags, nil
	}

	tree.Rules = CleanCSSTree(tree.Rules)

	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	printOpts := css_printer.Options{
		MinifyWhitespace: p.options.MinifyWhitespace,
		ASCIIOnly:        p.options.ASCIIOnly,
	}
	printed := css_printer.Print(*tree, symMap, printOpts)
	result := string(printed.CSS)

	l.Trace("CSS processed and bundled successfully",
		logger_domain.Int("inputSize", len(trimmed)),
		logger_domain.Int("outputSize", len(result)),
	)
	return result, inlinerDiags, nil
}

// InlineToAST resolves all @import rules in a raw CSS string and returns the
// cleaned syntax tree for further manipulation.
//
// Takes cssBlock (string) which contains the raw CSS to process.
// Takes sourcePath (string) which identifies the source file for diagnostics.
// Takes startLocation (ast_domain.Location) which marks the CSS block origin.
// Takes fsReader (FSReaderPort) which reads imported files from the filesystem.
//
// Returns *css_ast.AST which is the inlined and cleaned syntax tree.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors found.
// Returns error when processing fails.
func (p *Processor) InlineToAST(
	ctx context.Context,
	cssBlock string,
	sourcePath string,
	startLocation ast_domain.Location,
	fsReader FSReaderPort,
) (*css_ast.AST, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CSSProcessor.InlineToAST")
	defer span.End()

	ProcessCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ProcessDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	trimmed := strings.TrimSpace(cssBlock)
	if trimmed == "" {
		return nil, nil, nil
	}

	inliner := GetInliner(p.resolver, p.parserOptions, fsReader, p.diagnosticCode)
	defer PutInliner(inliner)
	tree, inlinerDiags := inliner.InlineAndParse(ctx, trimmed, sourcePath, startLocation)

	if tree == nil {
		ProcessErrorCount.Add(ctx, 1)
		l.Error("Failed to process CSS with fatal error during import resolution")
		return nil, inlinerDiags, nil
	}

	tree.Rules = CleanCSSTree(tree.Rules)

	l.Trace("CSS inlined to AST successfully",
		logger_domain.Int("inputSize", len(trimmed)),
		logger_domain.Int("ruleCount", len(tree.Rules)),
	)
	return tree, inlinerDiags, nil
}
