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

// Resolves CSS @import statements by recursively parsing and inlining imported
// stylesheets into a single AST. Detects circular dependencies, caches parsed files, and
// merges multiple CSS sources whilst preserving layer and media conditions.

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"errors"
	ast "piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
	"piko.sh/piko/internal/esbuild/css_parser"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safeconv"
)

// cssParseCache stores parsed CSS files to avoid parsing the same file twice. The map key
// is the full file path.
type cssParseCache = map[string]*css_ast.AST

// sourceContext locates a block of CSS in its source file, so a rule offset inside that
// block can be reported at the line and column the author wrote.
type sourceContext struct {
	// path is the file the CSS came from.
	path string

	// content is the CSS text that rule offsets are relative to.
	content string

	// startLocation is where content begins in the source file.
	startLocation ast.Location
}

// Inliner holds the state for a single CSS inlining operation.
type Inliner struct {
	// resolver resolves CSS import paths.
	resolver resolver_domain.ResolverPort

	// fsReader reads CSS files to resolve @import statements.
	fsReader FSReaderPort

	// cache stores parsed CSS stylesheets to avoid parsing the same file twice.
	cache cssParseCache

	// parseCache is the shared, content-addressed CSS parse cache; nil disables it.
	parseCache *parseCache

	// merged records every resolved path already inlined in this operation, so a stylesheet
	// reached by two different import chains is merged once rather than duplicated.
	merged map[string]struct{}

	// diagnosticCode is the code assigned to parser diagnostics produced by this inliner.
	diagnosticCode string

	// importDiagnosticCode is the code assigned to diagnostics raised because an @import
	// could not be resolved or read.
	importDiagnosticCode string

	// parserOptions holds the CSS parser settings.
	parserOptions css_parser.Options

	// diagnostics collects errors found while processing CSS imports.
	diagnostics []*ast.Diagnostic

	// limits bounds how far and how much this operation may inline.
	limits Limits

	// inlinedBytes counts the CSS merged so far in this operation, checked against
	// limits.MaxTotalBytes.
	inlinedBytes int
}

var (
	// inlinerPool reuses Inliner instances to reduce allocation pressure during CSS import
	// resolution.
	inlinerPool = sync.Pool{
		New: func() any {
			return &Inliner{}
		},
	}
)

// GetInliner retrieves an Inliner from the pool and prepares it for use.
//
// Takes resolver (resolver_domain.ResolverPort) which resolves CSS import paths.
// Takes parserOptions (css_parser.Options) which configures the CSS parser.
// Takes fsReader (FSReaderPort) which provides file system access.
// Takes diagnosticCode (string) which is assigned to generated diagnostics.
// Takes importDiagnosticCode (string) which is assigned to @import failures.
//
// Returns *Inliner which is ready to use.
func GetInliner(
	resolver resolver_domain.ResolverPort,
	parserOptions css_parser.Options,
	fsReader FSReaderPort,
	diagnosticCode string,
	importDiagnosticCode string,
	sharedParseCache *parseCache,
	limits Limits,
) *Inliner {
	inliner, ok := inlinerPool.Get().(*Inliner)
	if !ok {
		inliner = &Inliner{}
	}
	inliner.resolver = resolver
	inliner.parserOptions = parserOptions
	inliner.fsReader = fsReader
	inliner.diagnosticCode = diagnosticCode
	inliner.importDiagnosticCode = importDiagnosticCode
	inliner.cache = make(cssParseCache)
	inliner.parseCache = sharedParseCache
	inliner.merged = make(map[string]struct{})
	inliner.limits = limits.withDefaults()
	inliner.inlinedBytes = 0
	inliner.diagnostics = nil
	return inliner
}

// PutInliner clears the Inliner fields and returns it to the pool.
//
// Takes inliner (*Inliner) which is the inliner to reset and return.
func PutInliner(inliner *Inliner) {
	inliner.resolver = nil
	inliner.parserOptions = css_parser.Options{}
	inliner.fsReader = nil
	inliner.diagnosticCode = ""
	inliner.importDiagnosticCode = ""
	inliner.cache = nil
	inliner.parseCache = nil
	inliner.merged = nil
	inliner.limits = Limits{}
	inliner.inlinedBytes = 0
	inliner.diagnostics = nil
	inlinerPool.Put(inliner)
}

// InlineAndParse is the main entry point for CSS inlining. It takes the initial CSS
// content and its path, and returns a single, fully inlined AST.
//
// Takes cssContent (string) which contains the raw CSS to parse and inline.
// Takes containingPath (string) which specifies the file path for resolving relative
// imports.
// Takes startLocation (ast.Location) which marks the source position for diagnostics.
//
// Returns *css_ast.AST which is the fully inlined syntax tree, or nil when a fatal error
// such as an import cycle occurs.
// Returns []*ast.Diagnostic which contains any warnings or errors found during parsing.
func (i *Inliner) InlineAndParse(
	ctx context.Context,
	cssContent string,
	containingPath string,
	startLocation ast.Location,
) (*css_ast.AST, []*ast.Diagnostic) {
	tree, err := i.parseRecursive(ctx, cssContent, containingPath, startLocation, []string{})
	if err != nil {
		return nil, i.diagnostics
	}
	return tree, i.diagnostics
}

// parseRecursive is the core of the custom bundler logic.
//
// Takes cssContent (string) which is the CSS source to parse.
// Takes containingPath (string) which is the file path for error messages.
// Takes startLocation (ast.Location) which marks where the import appears.
// Takes pathStack ([]string) which tracks paths to detect circular imports.
//
// Returns *css_ast.AST which is the parsed syntax tree with imports resolved.
// Returns error when a circular import is found.
func (i *Inliner) parseRecursive(
	ctx context.Context,
	cssContent, containingPath string,
	startLocation ast.Location,
	pathStack []string,
) (*css_ast.AST, error) {
	if err := i.checkCircularDependency(containingPath, startLocation, pathStack); err != nil {
		return nil, fmt.Errorf("checking circular dependency for %q: %w", containingPath, err)
	}
	if len(pathStack) > i.limits.MaxDepth {
		return nil, i.reportLimit(startLocation, containingPath, fmt.Sprintf(
			"CSS imports nest deeper than the limit of %d; raise it with WithCSSImportMaxDepth",
			i.limits.MaxDepth))
	}

	if cachedAST, exists := i.cache[containingPath]; exists {
		return CloneAST(cachedAST), nil
	}

	tree, hasParseErrors := i.parseCSSContent(cssContent, containingPath, startLocation)
	if hasParseErrors {
		return &tree, nil
	}

	if len(tree.ImportRecords) > 0 {
		processedTree, err := i.processImports(ctx, tree, containingPath, cssContent, startLocation, pathStack)
		if err != nil {
			return nil, fmt.Errorf("processing CSS imports for %q: %w", containingPath, err)
		}
		tree = processedTree
	}

	i.cache[containingPath] = &tree
	return &tree, nil
}

// checkCircularDependency detects if a CSS import path creates a cycle.
//
// Takes containingPath (string) which is the path being checked for cycles.
// Takes startLocation (ast.Location) which is the source location for error reporting.
// Takes pathStack ([]string) which is the current chain of import paths.
//
// Returns error when containingPath already exists in pathStack, indicating a circular
// dependency.
func (i *Inliner) checkCircularDependency(containingPath string, startLocation ast.Location, pathStack []string) error {
	if !slices.Contains(pathStack, containingPath) {
		return nil
	}
	cyclePath := make([]string, len(pathStack)+1)
	copy(cyclePath, pathStack)
	cyclePath[len(pathStack)] = containingPath
	err := NewCircularDependencyError(cyclePath)
	diagnostic := ast.NewDiagnosticWithCode(ast.Error, err.Error(), "", i.diagnosticCode, startLocation, pathStack[0])
	i.diagnostics = append(i.diagnostics, diagnostic)
	return fmt.Errorf("circular CSS import dependency detected: %w", err)
}

// parseCSSContent parses raw CSS text and returns the resulting AST.
//
// Takes cssContent (string) which is the CSS source text to parse.
// Takes containingPath (string) which identifies the source file for error messages.
// Takes startLocation (ast.Location) which is the offset for error positions.
//
// Returns css_ast.AST which is the parsed stylesheet tree.
// Returns bool which is true when parsing errors occurred.
func (i *Inliner) parseCSSContent(cssContent, containingPath string, startLocation ast.Location) (css_ast.AST, bool) {
	tree, messages := i.parseOrLoadCSS(cssContent, containingPath)

	diagnostics := ConvertESBuildMessagesToDiagnostics(messages, containingPath, startLocation, i.diagnosticCode)
	if len(diagnostics) > 0 {
		i.diagnostics = append(i.diagnostics, diagnostics...)
	}
	return tree, ast.HasErrors(diagnostics)
}

// parseOrLoadCSS parses CSS, returning a private clone on a cache hit.
//
// The shared cache's master AST is never handed out: every consumer gets its own
// CloneAST, since downstream mutates the tree in place. The raw parse messages are
// returned so the caller can re-offset them to its own location, keeping cache-hit
// diagnostics identical.
//
// Takes cssContent (string) which is the CSS source text to parse.
// Takes containingPath (string) which identifies the source file for diagnostics.
//
// Returns css_ast.AST which is a private parsed stylesheet tree.
// Returns []es_logger.Msg which are the raw esbuild parse messages.
func (i *Inliner) parseOrLoadCSS(cssContent, containingPath string) (css_ast.AST, []es_logger.Msg) {
	if i.parseCache != nil {
		if entry := i.parseCache.get(cssContent); entry != nil {
			return *CloneAST(entry.master), entry.diags
		}
	}

	esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
	source := es_logger.Source{
		KeyPath:  es_logger.Path{Text: containingPath},
		Contents: cssContent,
	}
	tree := css_parser.Parse(esLog, source, i.parserOptions)
	messages := esLog.Done()

	if i.parseCache == nil {
		return tree, messages
	}

	winner := i.parseCache.put(cssContent, &parseCacheEntry{master: &tree, diags: messages, content: cssContent})
	return *CloneAST(winner.master), winner.diags
}

// processImports resolves and inlines CSS @import rules into the AST.
//
// Takes tree (css_ast.AST) which is the parsed CSS to process.
// Takes containingPath (string) which is the file path of the CSS being processed.
// Takes startLocation (ast.Location) which marks where processing began.
// Takes pathStack ([]string) which tracks visited paths to detect cycles.
//
// Returns css_ast.AST which is the tree with imports merged in reverse order.
// Returns error when an imported file cannot be collected or processed.
func (i *Inliner) processImports(
	ctx context.Context,
	tree css_ast.AST,
	containingPath string,
	cssContent string,
	startLocation ast.Location,
	pathStack []string,
) (css_ast.AST, error) {
	newPathStack := slices.Concat(pathStack, []string{containingPath})
	rulesToKeep, importsToMerge, err := i.collectImportedASTs(ctx, tree, containingPath, cssContent, startLocation, newPathStack)
	if err != nil {
		return css_ast.AST{}, err
	}

	newTree := css_ast.AST{
		CharFreq:             tree.CharFreq,
		ApproximateLineCount: tree.ApproximateLineCount,
		SourceMapComment:     tree.SourceMapComment,
		Rules:                rulesToKeep,
		Symbols:              tree.Symbols,
		ImportRecords:        tree.ImportRecords,
		LayersPreImport:      tree.LayersPreImport,
		LayersPostImport:     tree.LayersPostImport,
	}

	for _, importToMerge := range slices.Backward(importsToMerge) {
		MergeASTs(&newTree, importToMerge)
	}

	newTree.Rules = hoistImportRules(newTree.Rules)

	return newTree, nil
}

// hoistImportRules reorders a rule list into the prologue order the CSS specification
// requires: a leading @charset (if present) first, then every @import rule, then all
// other rules.
//
// Takes rules ([]css_ast.Rule) which is the rule list to reorder.
//
// Returns []css_ast.Rule ordered @charset first, then @import, then the remaining rules.
func hoistImportRules(rules []css_ast.Rule) []css_ast.Rule {
	if !importRulesMisplaced(rules) {
		return rules
	}

	charsets := make([]css_ast.Rule, 0, len(rules))
	imports := make([]css_ast.Rule, 0, len(rules))
	others := make([]css_ast.Rule, 0, len(rules))
	for _, rule := range rules {
		switch rule.Data.(type) {
		case *css_ast.RAtCharset:
			charsets = append(charsets, rule)
		case *css_ast.RAtImport:
			imports = append(imports, rule)
		default:
			others = append(others, rule)
		}
	}

	ordered := make([]css_ast.Rule, 0, len(rules))
	ordered = append(ordered, charsets...)
	ordered = append(ordered, imports...)
	ordered = append(ordered, others...)
	return ordered
}

// importRulesMisplaced reports whether a rule list deviates from the CSS prologue order
// of @charset, then @import, then everything else.
//
// Takes rules ([]css_ast.Rule) which is the rule list to inspect.
//
// Returns bool which is true when at least one @charset or @import rule is out of place.
func importRulesMisplaced(rules []css_ast.Rule) bool {
	seenImport := false
	seenOther := false
	for i := range rules {
		switch rules[i].Data.(type) {
		case *css_ast.RAtCharset:
			if seenImport || seenOther {
				return true
			}
		case *css_ast.RAtImport:
			if seenOther {
				return true
			}
			seenImport = true
		default:
			seenOther = true
		}
	}
	return false
}

// isExternalImportPath reports whether a CSS @import target refers to an external
// resource that must be fetched by the browser at runtime rather than inlined at build
// time.
//
// Takes path (string) which is the raw @import target text.
//
// Returns bool which is true when the target is external and should be kept verbatim.
func isExternalImportPath(path string) bool {
	if strings.HasPrefix(path, "//") {
		return true
	}

	colon := strings.IndexByte(path, ':')
	if colon <= 0 {
		return false
	}
	for i := range colon {
		c := path[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9', c == '+', c == '-', c == '.':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// collectImportedASTs processes a CSS AST to separate import rules from other rules and
// resolve the imported stylesheets.
//
// Takes tree (css_ast.AST) which is the CSS abstract syntax tree to process.
// Takes containingPath (string) which is the file path of the stylesheet.
// Takes startLocation (ast.Location) which marks the import's source position.
// Takes pathStack ([]string) which tracks visited paths to detect cycles.
//
// Returns []css_ast.Rule which contains the rules kept as-is: ordinary rules plus any
// external @import rules that are left verbatim for the browser to resolve at runtime.
// Returns []*css_ast.AST which contains the parsed ASTs of inlined local imports.
// Returns error when a circular import is found or parsing fails.
func (i *Inliner) collectImportedASTs(
	ctx context.Context,
	tree css_ast.AST,
	containingPath string,
	cssContent string,
	startLocation ast.Location,
	pathStack []string,
) ([]css_ast.Rule, []*css_ast.AST, error) {
	var rulesToKeep []css_ast.Rule
	var importsToMerge []*css_ast.AST

	for _, rule := range tree.Rules {
		imp, isImport := rule.Data.(*css_ast.RAtImport)
		if !isImport {
			rulesToKeep = append(rulesToKeep, rule)
			continue
		}

		if isExternalImportPath(tree.ImportRecords[imp.ImportRecordIndex].Path.Text) {
			rulesToKeep = append(rulesToKeep, rule)
			continue
		}

		importedAST, err := i.resolveAndParseImport(ctx, tree, imp, rule, sourceContext{path: containingPath, content: cssContent, startLocation: startLocation}, pathStack)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving CSS import in %q: %w", containingPath, err)
		}
		if importedAST != nil {
			importsToMerge = append(importsToMerge, importedAST)
		}
	}

	return rulesToKeep, importsToMerge, nil
}

// resolveAndParseImport resolves an @import path and parses the imported CSS.
//
// Takes tree (css_ast.AST) which provides the import records for resolution.
// Takes imp (*css_ast.RAtImport) which specifies the import rule to process.
// Takes rule (css_ast.Rule) which provides the original rule location.
// Takes source (sourceContext) which locates the CSS block that holds the import.
// Takes pathStack ([]string) which tracks visited paths to find cycles.
//
// Returns *css_ast.AST which is the parsed and condition-wrapped imported CSS, or nil if
// the import could not be resolved or parsed.
// Returns error when a circular import is found.
func (i *Inliner) resolveAndParseImport(
	ctx context.Context,
	tree css_ast.AST,
	imp *css_ast.RAtImport,
	rule css_ast.Rule,
	source sourceContext,
	pathStack []string,
) (*css_ast.AST, error) {
	containingPath, cssContent, startLocation := source.path, source.content, source.startLocation
	importRecord := tree.ImportRecords[imp.ImportRecordIndex]
	importPath := importRecord.Path.Text
	ruleLocation := LocationForOffset(cssContent, int(rule.Loc.Start), startLocation)

	resolvedPath, err := i.resolver.ResolveCSSPath(ctx, importPath, filepath.Dir(containingPath))
	if err != nil {
		diagnostic := ast.NewDiagnosticWithCode(ast.Error, err.Error(), importPath, i.importDiagnosticCode, ruleLocation, containingPath)
		i.diagnostics = append(i.diagnostics, diagnostic)
		return nil, nil
	}

	if _, alreadyMerged := i.merged[resolvedPath]; alreadyMerged && !slices.Contains(pathStack, resolvedPath) {
		return nil, nil
	}

	importedContent, err := i.fsReader.ReadFile(ctx, resolvedPath)
	if err != nil {
		diagnostic := ast.NewDiagnosticWithCode(ast.Error, err.Error(), importPath, i.importDiagnosticCode, ruleLocation, containingPath)
		i.diagnostics = append(i.diagnostics, diagnostic)
		return nil, nil
	}

	i.inlinedBytes += len(importedContent)
	if i.inlinedBytes > i.limits.MaxTotalBytes {
		return nil, i.reportLimit(ruleLocation, containingPath,
			fmt.Sprintf("CSS imports total more than the limit of %d bytes; raise it with WithCSSImportMaxBytes", i.limits.MaxTotalBytes))
	}
	i.merged[resolvedPath] = struct{}{}

	importedAST, err := i.parseRecursive(ctx, string(importedContent), resolvedPath, ast.Location{Line: 1, Column: 1, Offset: 0}, pathStack)
	if err != nil {
		return nil, fmt.Errorf("parsing imported CSS %q: %w", resolvedPath, err)
	}
	if importedAST == nil {
		return nil, nil
	}

	return WrapImportedASTWithConditions(importedAST, imp.ImportConditions, rule.Loc), nil
}

// MergeASTs joins a child AST into a parent AST by updating all token indexes. Child
// rules are added before parent rules so that imports appear first in the CSS output.
//
// Takes parent (*css_ast.AST) which receives the merged result.
// Takes child (*css_ast.AST) which provides the rules to add.
func MergeASTs(parent *css_ast.AST, child *css_ast.AST) {
	symbolOffset := safeconv.IntToUint32(len(parent.Symbols))
	importRecordOffset := safeconv.IntToUint32(len(parent.ImportRecords))

	parent.Symbols = append(parent.Symbols, child.Symbols...)
	parent.ImportRecords = append(parent.ImportRecords, child.ImportRecords...)

	childRulesCopy := make([]css_ast.Rule, len(child.Rules), len(child.Rules)+len(parent.Rules))
	copy(childRulesCopy, child.Rules)
	for i := range childRulesCopy {
		reIndexRule(&childRulesCopy[i], symbolOffset, importRecordOffset)
	}

	parent.Rules = append(childRulesCopy, parent.Rules...)
}

// reIndexRule adjusts index values in a CSS rule and its child rules.
//
// Takes rule (*css_ast.Rule) which is the rule to update.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexRule(rule *css_ast.Rule, symbolOffset, importRecordOffset uint32) {
	switch r := rule.Data.(type) {
	case *css_ast.RSelector:
		reIndexSelectorRule(r, symbolOffset, importRecordOffset)
	case *css_ast.RKnownAt:
		reIndexAtRule(r.Prelude, r.Rules, symbolOffset, importRecordOffset)
	case *css_ast.RDeclaration:
		reIndexTokens(r.Value, symbolOffset, importRecordOffset)
	case *css_ast.RAtLayer:
		reIndexRuleList(r.Rules, symbolOffset, importRecordOffset)
	case *css_ast.RAtKeyframes:
		reIndexKeyframesRule(r, symbolOffset, importRecordOffset)
	case *css_ast.RAtMedia:
		reIndexRuleList(r.Rules, symbolOffset, importRecordOffset)
	case *css_ast.RQualified:
		reIndexAtRule(r.Prelude, r.Rules, symbolOffset, importRecordOffset)
	case *css_ast.RBadDeclaration:
		reIndexTokens(r.Tokens, symbolOffset, importRecordOffset)
	case *css_ast.RAtImport:
		r.ImportRecordIndex += importRecordOffset
		if r.ImportConditions != nil {
			reIndexTokens(r.ImportConditions.Layers, symbolOffset, importRecordOffset)
			reIndexTokens(r.ImportConditions.Supports, symbolOffset, importRecordOffset)
		}
	case *css_ast.RUnknownAt:
		reIndexTokens(r.Prelude, symbolOffset, importRecordOffset)
		reIndexTokens(r.Block, symbolOffset, importRecordOffset)
	}
}

// reIndexSelectorRule updates symbol and import record indices for a selector rule and
// its nested rules.
//
// Takes r (*css_ast.RSelector) which is the selector rule to re-index.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexSelectorRule(r *css_ast.RSelector, symbolOffset, importRecordOffset uint32) {
	for i := range r.Selectors {
		reIndexSelector(&r.Selectors[i], symbolOffset, importRecordOffset)
	}
	reIndexRuleList(r.Rules, symbolOffset, importRecordOffset)
}

// reIndexKeyframesRule updates symbol references in a keyframes rule and its nested
// blocks.
//
// Takes r (*css_ast.RAtKeyframes) which is the keyframes rule to re-index.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexKeyframesRule(r *css_ast.RAtKeyframes, symbolOffset, importRecordOffset uint32) {
	if symbolOffset > 0 {
		r.Name.Ref.InnerIndex += symbolOffset
		r.Name.Ref.SourceIndex = 0
	}
	for i := range r.Blocks {
		for j := range r.Blocks[i].Rules {
			reIndexRule(&r.Blocks[i].Rules[j], symbolOffset, importRecordOffset)
		}
	}
}

// reIndexAtRule updates index values in an at-rule's prelude and nested rules.
//
// Takes prelude ([]css_ast.Token) which contains the at-rule's prelude tokens.
// Takes rules ([]css_ast.Rule) which contains the at-rule's nested rules.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexAtRule(prelude []css_ast.Token, rules []css_ast.Rule, symbolOffset, importRecordOffset uint32) {
	reIndexTokens(prelude, symbolOffset, importRecordOffset)
	reIndexRuleList(rules, symbolOffset, importRecordOffset)
}

// reIndexRuleList updates symbol and import record indices for a list of CSS rules.
//
// Takes rules ([]css_ast.Rule) which contains the rules to re-index.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexRuleList(rules []css_ast.Rule, symbolOffset, importRecordOffset uint32) {
	for i := range rules {
		reIndexRule(&rules[i], symbolOffset, importRecordOffset)
	}
}

// reIndexSelector updates symbol references in a CSS complex selector.
//
// Takes selector (*css_ast.ComplexSelector) which is the selector to re-index.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexSelector(selector *css_ast.ComplexSelector, symbolOffset, importRecordOffset uint32) {
	for i := range selector.Selectors {
		reIndexCompoundSelector(&selector.Selectors[i], symbolOffset, importRecordOffset)
	}
}

// reIndexCompoundSelector updates symbol and import record references in a compound
// selector.
//
// Takes cs (*css_ast.CompoundSelector) which is the compound selector to re-index.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexCompoundSelector(cs *css_ast.CompoundSelector, symbolOffset, importRecordOffset uint32) {
	for i := range cs.SubclassSelectors {
		switch data := cs.SubclassSelectors[i].Data.(type) {
		case *css_ast.SSHash:
			if symbolOffset > 0 {
				data.Name.Ref.InnerIndex += symbolOffset
				data.Name.Ref.SourceIndex = 0
			}
		case *css_ast.SSClass:
			if symbolOffset > 0 {
				data.Name.Ref.InnerIndex += symbolOffset
				data.Name.Ref.SourceIndex = 0
			}
		case *css_ast.SSPseudoClassWithSelectorList:
			for j := range data.Selectors {
				reIndexSelector(&data.Selectors[j], symbolOffset, importRecordOffset)
			}
		case *css_ast.SSPseudoClass:
			if len(data.Args) > 0 {
				reIndexTokens(data.Args, symbolOffset, importRecordOffset)
			}
		}
	}
}

// reIndexTokens updates payload indices in CSS tokens when merging symbol tables. URL
// tokens receive the import record offset and symbol tokens receive the symbol offset.
//
// Takes tokens ([]css_ast.Token) which contains the tokens to re-index.
// Takes symbolOffset (uint32) which is the offset to add to symbol indices.
// Takes importRecordOffset (uint32) which is the offset to add to import record indices.
func reIndexTokens(tokens []css_ast.Token, symbolOffset, importRecordOffset uint32) {
	for i := range tokens {
		t := &tokens[i]
		switch t.Kind {
		case css_lexer.TURL:
			t.PayloadIndex += importRecordOffset
		case css_lexer.TSymbol:
			t.PayloadIndex += symbolOffset
		default:
		}
		if t.Children != nil {
			reIndexTokens(*t.Children, symbolOffset, importRecordOffset)
		}
	}
}

// CloneAST creates a deep copy of an AST for use in caching.
//
// It copies only the fields the inliner and CSS printer read downstream:
// SourceMapComment, ApproximateLineCount, Symbols, ImportRecords, and Rules. The other
// css_ast.AST fields (CharFreq, the scope and symbol maps, Composes, and the Layers
// slices) are intentionally NOT copied: nothing after inlining consumes them, and
// dropping them avoids aliasing the cached master's maps/slices. Because the parse cache
// makes CloneAST the sole path for every parsed tree, a future consumer that reads those
// fields (e.g. CSS Modules via Composes / LocalScope) must extend this clone.
//
// When original is nil, returns nil.
//
// Takes original (*css_ast.AST) which is the AST to copy.
//
// Returns *css_ast.AST which is a new copy of the original AST.
func CloneAST(original *css_ast.AST) *css_ast.AST {
	if original == nil {
		return nil
	}
	clone := &css_ast.AST{
		SourceMapComment:     original.SourceMapComment,
		ApproximateLineCount: original.ApproximateLineCount,
	}

	if original.Symbols != nil {
		clone.Symbols = make([]es_ast.Symbol, len(original.Symbols))
		copy(clone.Symbols, original.Symbols)
	}
	if original.ImportRecords != nil {
		clone.ImportRecords = make([]es_ast.ImportRecord, len(original.ImportRecords))
		copy(clone.ImportRecords, original.ImportRecords)
	}
	if original.Rules != nil {
		clone.Rules = make([]css_ast.Rule, len(original.Rules))
		for i, rule := range original.Rules {
			clone.Rules[i] = cloneRule(rule)
		}
	}

	return clone
}

// cloneRule creates a deep copy of a CSS rule, including its data and location.
//
// Takes original (css_ast.Rule) which is the rule to copy.
//
// Returns css_ast.Rule which is a new independent copy.
func cloneRule(original css_ast.Rule) css_ast.Rule {
	return css_ast.Rule{
		Data: cloneR(original.Data),
		Loc:  original.Loc,
	}
}

// cloneR creates a deep copy of a CSS rule node, dispatching to the appropriate
// type-specific clone function based on the concrete type.
//
// Takes original (css_ast.R) which is the rule data to copy.
//
// Returns css_ast.R which is a new deep copy, or nil if the original is nil or an
// unrecognised type.
func cloneR(original css_ast.R) css_ast.R {
	if original == nil {
		return nil
	}
	switch r := original.(type) {
	case *css_ast.RAtCharset:
		return new(*r)
	case *css_ast.RAtImport:
		return cloneRAtImport(r)
	case *css_ast.RAtKeyframes:
		return cloneRAtKeyframes(r)
	case *css_ast.RKnownAt:
		return cloneRKnownAt(r)
	case *css_ast.RUnknownAt:
		return cloneRUnknownAt(r)
	case *css_ast.RSelector:
		return cloneRSelector(r)
	case *css_ast.RDeclaration:
		clone := *r
		clone.Value = cloneTokens(r.Value)
		return &clone
	case *css_ast.RBadDeclaration:
		clone := *r
		clone.Tokens = cloneTokens(r.Tokens)
		return &clone
	case *css_ast.RComment:
		return new(*r)
	case *css_ast.RAtLayer:
		return cloneRAtLayer(r)
	case *css_ast.RAtMedia:
		return cloneRAtMedia(r)
	default:
		return nil
	}
}

// cloneRAtImport creates a deep copy of a CSS @import rule, including its import
// conditions.
//
// Takes r (*css_ast.RAtImport) which is the rule to copy.
//
// Returns *css_ast.RAtImport which is a new copy.
func cloneRAtImport(r *css_ast.RAtImport) *css_ast.RAtImport {
	clone := *r
	if r.ImportConditions != nil {
		clone.ImportConditions = cloneImportConditions(r.ImportConditions)
	}
	return &clone
}

// cloneImportConditions deep-copies the layer, supports, and media conditions of a CSS
// @import rule, preserving each url() token's import-record index verbatim.
//
// Takes conditions (*css_ast.ImportConditions) which holds the conditions to copy.
//
// Returns *css_ast.ImportConditions which is an independent copy of the conditions.
func cloneImportConditions(conditions *css_ast.ImportConditions) *css_ast.ImportConditions {
	clone := *conditions
	clone.Layers = cloneTokens(conditions.Layers)
	clone.Supports = cloneTokens(conditions.Supports)
	if conditions.Queries != nil {
		clone.Queries = make([]css_ast.MediaQuery, len(conditions.Queries))
		copy(clone.Queries, conditions.Queries)
	}
	return &clone
}

// cloneRAtKeyframes creates a deep copy of a CSS @keyframes rule with all its blocks
// cloned.
//
// Takes r (*css_ast.RAtKeyframes) which is the keyframes rule to copy.
//
// Returns *css_ast.RAtKeyframes which is a new copy.
func cloneRAtKeyframes(r *css_ast.RAtKeyframes) *css_ast.RAtKeyframes {
	clone := *r
	clone.Blocks = make([]css_ast.KeyframeBlock, len(r.Blocks))
	for i, block := range r.Blocks {
		clone.Blocks[i] = cloneKeyframeBlock(block)
	}
	return &clone
}

// cloneRKnownAt creates a deep copy of a known CSS at-rule with its prelude and nested
// rules cloned.
//
// Takes r (*css_ast.RKnownAt) which is the at-rule to copy.
//
// Returns *css_ast.RKnownAt which is a new copy.
func cloneRKnownAt(r *css_ast.RKnownAt) *css_ast.RKnownAt {
	clone := *r
	clone.Prelude = cloneTokens(r.Prelude)
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRUnknownAt creates a deep copy of an unknown CSS at-rule with its prelude and
// block tokens cloned.
//
// Takes r (*css_ast.RUnknownAt) which is the at-rule to copy.
//
// Returns *css_ast.RUnknownAt which is a new copy.
func cloneRUnknownAt(r *css_ast.RUnknownAt) *css_ast.RUnknownAt {
	clone := *r
	clone.Prelude = cloneTokens(r.Prelude)
	clone.Block = cloneTokens(r.Block)
	return &clone
}

// cloneRSelector creates a deep copy of a CSS selector rule with all its selectors and
// nested rules cloned.
//
// Takes r (*css_ast.RSelector) which is the selector rule to copy.
//
// Returns *css_ast.RSelector which is a new copy.
func cloneRSelector(r *css_ast.RSelector) *css_ast.RSelector {
	clone := *r
	clone.Selectors = make([]css_ast.ComplexSelector, len(r.Selectors))
	for i, selector := range r.Selectors {
		clone.Selectors[i] = selector.Clone()
	}
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRAtLayer creates a deep copy of a CSS @layer rule with its layer names and nested
// rules cloned.
//
// Takes r (*css_ast.RAtLayer) which is the @layer rule to copy.
//
// Returns *css_ast.RAtLayer which is a new copy.
func cloneRAtLayer(r *css_ast.RAtLayer) *css_ast.RAtLayer {
	clone := *r
	clone.Names = make([][]string, len(r.Names))
	for i, name := range r.Names {
		clone.Names[i] = append([]string(nil), name...)
	}
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRAtMedia creates a deep copy of a CSS @media rule with its queries and nested
// rules cloned.
//
// Takes r (*css_ast.RAtMedia) which is the @media rule to copy.
//
// Returns *css_ast.RAtMedia which is a new copy.
func cloneRAtMedia(r *css_ast.RAtMedia) *css_ast.RAtMedia {
	clone := *r
	clone.Queries = make([]css_ast.MediaQuery, len(r.Queries))
	copy(clone.Queries, r.Queries)
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRules creates a deep copy of a slice of CSS rules.
//
// Takes rules ([]css_ast.Rule) which contains the rules to copy.
//
// Returns []css_ast.Rule which is a new slice with each rule deep-copied.
func cloneRules(rules []css_ast.Rule) []css_ast.Rule {
	cloned := make([]css_ast.Rule, len(rules))
	for i, rule := range rules {
		cloned[i] = cloneRule(rule)
	}
	return cloned
}

// cloneKeyframeBlock creates a deep copy of a CSS keyframe block, including its selectors
// and nested rules.
//
// Takes original (css_ast.KeyframeBlock) which is the keyframe block to copy.
//
// Returns css_ast.KeyframeBlock which is a new independent copy.
func cloneKeyframeBlock(original css_ast.KeyframeBlock) css_ast.KeyframeBlock {
	clone := original
	clone.Selectors = append([]string(nil), original.Selectors...)
	clone.Rules = make([]css_ast.Rule, len(original.Rules))
	for i, rule := range original.Rules {
		clone.Rules[i] = cloneRule(rule)
	}
	return clone
}

// cloneTokens creates a deep copy of a CSS token slice, recursively cloning any child
// token slices.
//
// Takes original ([]css_ast.Token) which contains the tokens to copy.
//
// Returns []css_ast.Token which is a new slice with each token deep-copied, or nil if the
// original is nil.
func cloneTokens(original []css_ast.Token) []css_ast.Token {
	if original == nil {
		return nil
	}
	clone := make([]css_ast.Token, len(original))
	for i, t := range original {
		clone[i] = t
		if t.Children != nil {
			clone[i].Children = new(cloneTokens(*t.Children))
		}
	}
	return clone
}

// WrapImportedASTWithConditions wraps the imported AST's rules with @layer, @supports,
// and/or @media rules based on the ImportConditions from the @import statement.
//
// Takes importedAST (*css_ast.AST) which is the parsed CSS to wrap.
// Takes conditions (*css_ast.ImportConditions) which specifies the wrapping rules from
// the @import statement.
// Takes loc (es_logger.Loc) which provides the source location for the generated wrapper
// rules.
//
// Returns *css_ast.AST which is a new AST with the rules wrapped according to the
// conditions, or the original AST if conditions is nil.
func WrapImportedASTWithConditions(importedAST *css_ast.AST, conditions *css_ast.ImportConditions, loc es_logger.Loc) *css_ast.AST {
	if conditions == nil {
		return importedAST
	}

	rules := importedAST.Rules

	if len(conditions.Layers) > 0 {
		layerNames := extractLayerNamesFromTokens(conditions.Layers)
		rules = []css_ast.Rule{
			{
				Loc: loc,
				Data: &css_ast.RAtLayer{
					Names: layerNames,
					Rules: rules,
				},
			},
		}
	}

	if len(conditions.Supports) > 0 {
		rules = []css_ast.Rule{
			{
				Loc: loc,
				Data: &css_ast.RKnownAt{
					AtToken: "@supports",
					Prelude: conditions.Supports,
					Rules:   rules,
				},
			},
		}
	}

	if len(conditions.Queries) > 0 {
		rules = []css_ast.Rule{
			{
				Loc: loc,
				Data: &css_ast.RAtMedia{
					Queries: conditions.Queries,
					Rules:   rules,
				},
			},
		}
	}

	wrapped := CloneAST(importedAST)
	wrapped.Rules = rules
	return wrapped
}

// extractLayerNamesFromTokens parses layer names from CSS import tokens by finding a
// "layer" function token and extracting its children.
//
// Takes tokens ([]css_ast.Token) which contains the import condition tokens.
//
// Returns [][]string which contains the parsed layer name parts.
func extractLayerNamesFromTokens(tokens []css_ast.Token) [][]string {
	if len(tokens) == 0 {
		return [][]string{{}}
	}

	for _, token := range tokens {
		if token.Kind == css_lexer.TFunction {
			if strings.EqualFold(token.Text, "layer") && token.Children != nil {
				return parseLayerNameFromChildren(*token.Children)
			}
		}
	}

	return [][]string{{}}
}

// parseLayerNameFromChildren extracts layer names from the child tokens of a CSS layer()
// function, collecting identifier tokens as name parts.
//
// Takes children ([]css_ast.Token) which contains the function's child tokens.
//
// Returns [][]string which contains the parsed layer name parts.
func parseLayerNameFromChildren(children []css_ast.Token) [][]string {
	if len(children) == 0 {
		return [][]string{{}}
	}

	var parts []string
	for _, child := range children {
		if child.Kind == css_lexer.TIdent {
			parts = append(parts, child.Text)
		}
	}

	if len(parts) == 0 {
		return [][]string{{}}
	}

	return [][]string{parts}
}

// reportLimit records a hard limit being reached and returns the matching error.
//
// A limit is reported as a diagnostic as well as an error so the failure reaches the
// build output through the same channel as every other CSS problem, rather than
// truncating the stylesheet quietly.
//
// Takes startLocation (ast.Location) which is where to attribute the failure.
// Takes sourcePath (string) which is the file the limit was reached in.
// Takes message (string) which describes the limit and how to raise it.
//
// Returns error which reports the limit.
func (i *Inliner) reportLimit(startLocation ast.Location, sourcePath, message string) error {
	i.diagnostics = append(i.diagnostics,
		ast.NewDiagnosticWithCode(ast.Error, message, "", CodeImportLimitExceeded, startLocation, sourcePath))
	return errors.New(message)
}
