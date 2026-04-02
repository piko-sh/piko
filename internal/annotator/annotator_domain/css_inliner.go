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

package annotator_domain

// Resolves CSS @import statements by recursively parsing and inlining imported stylesheets into a single AST.
// Detects circular dependencies, caches parsed files, and merges multiple CSS sources whilst preserving layer and media conditions.

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	ast "piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
	"piko.sh/piko/internal/esbuild/css_parser"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/wdk/safeconv"
)

// cssParseCache stores parsed CSS files to avoid parsing the same file twice.
// The map key is the full file path.
type cssParseCache = map[string]*css_ast.AST

// cssInliner holds the state for a single CSS inlining operation.
type cssInliner struct {
	// cp is the CSS processor used to parse stylesheets and resolve imports.
	cp *CSSProcessor

	// fsReader reads CSS files to resolve @import statements.
	fsReader FSReaderPort

	// cache stores parsed CSS stylesheets to avoid parsing the same file twice.
	cache cssParseCache

	// diagnostics collects errors found while processing CSS imports.
	diagnostics []*ast.Diagnostic
}

var cssInlinerPool = sync.Pool{
	New: func() any {
		return &cssInliner{}
	},
}

// InlineAndParse is the main entry point for CSS inlining. It takes the
// initial CSS content and its path, and returns a single, fully inlined AST.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes cssContent (string) which contains the raw CSS to parse and inline.
// Takes containingPath (string) which specifies the file path for resolving
// relative imports.
// Takes startLocation (ast.Location) which marks the source position for
// diagnostics.
//
// Returns *css_ast.AST which is the fully inlined syntax tree, or nil when
// a fatal error such as an import cycle occurs.
// Returns []*ast.Diagnostic which contains any warnings or errors found
// during parsing.
func (ci *cssInliner) InlineAndParse(
	ctx context.Context,
	cssContent string,
	containingPath string,
	startLocation ast.Location,
) (*css_ast.AST, []*ast.Diagnostic) {
	tree, err := ci.parseRecursive(ctx, cssContent, containingPath, startLocation, []string{})
	if err != nil {
		return nil, ci.diagnostics
	}
	return tree, ci.diagnostics
}

// parseRecursive is the core of the custom bundler logic.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes cssContent (string) which is the CSS source to parse.
// Takes containingPath (string) which is the file path for error messages.
// Takes startLocation (ast.Location) which marks where the import appears.
// Takes pathStack ([]string) which tracks paths to detect circular imports.
//
// Returns *css_ast.AST which is the parsed syntax tree with imports resolved.
// Returns error when a circular import is found.
func (ci *cssInliner) parseRecursive(
	ctx context.Context,
	cssContent, containingPath string,
	startLocation ast.Location,
	pathStack []string,
) (*css_ast.AST, error) {
	if err := ci.checkCircularDependency(containingPath, startLocation, pathStack); err != nil {
		return nil, fmt.Errorf("checking circular dependency for %q: %w", containingPath, err)
	}

	if cachedAST, exists := ci.cache[containingPath]; exists {
		return cloneAST(cachedAST), nil
	}

	tree, hasParseErrors := ci.parseCSSContent(cssContent, containingPath, startLocation)
	if hasParseErrors {
		return &tree, nil
	}

	if len(tree.ImportRecords) > 0 {
		processedTree, err := ci.processImports(ctx, tree, containingPath, startLocation, pathStack)
		if err != nil {
			return nil, fmt.Errorf("processing CSS imports for %q: %w", containingPath, err)
		}
		tree = processedTree
	}

	clonedTree := cloneAST(&tree)
	ci.cache[containingPath] = clonedTree
	return clonedTree, nil
}

// checkCircularDependency detects if a CSS import path creates a cycle.
//
// Takes containingPath (string) which is the path being checked for cycles.
// Takes startLocation (ast.Location) which is the source location for error
// reporting.
// Takes pathStack ([]string) which is the current chain of import paths.
//
// Returns error when containingPath already exists in pathStack, indicating a
// circular dependency.
func (ci *cssInliner) checkCircularDependency(containingPath string, startLocation ast.Location, pathStack []string) error {
	if !slices.Contains(pathStack, containingPath) {
		return nil
	}
	cyclePath := make([]string, len(pathStack)+1)
	copy(cyclePath, pathStack)
	cyclePath[len(pathStack)] = containingPath
	err := NewCircularDependencyError(cyclePath)
	diagnostic := ast.NewDiagnosticWithCode(ast.Error, err.Error(), "", annotator_dto.CodeCSSImportError, startLocation, pathStack[0])
	ci.diagnostics = append(ci.diagnostics, diagnostic)
	return fmt.Errorf("circular CSS import dependency detected: %w", err)
}

// parseCSSContent parses raw CSS text and returns the resulting AST.
//
// Takes cssContent (string) which is the CSS source text to parse.
// Takes containingPath (string) which identifies the source file for error
// messages.
// Takes startLocation (ast.Location) which is the offset for error positions.
//
// Returns css_ast.AST which is the parsed stylesheet tree.
// Returns bool which is true when parsing errors occurred.
func (ci *cssInliner) parseCSSContent(cssContent, containingPath string, startLocation ast.Location) (css_ast.AST, bool) {
	esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
	source := es_logger.Source{
		KeyPath:  es_logger.Path{Text: containingPath},
		Contents: cssContent,
	}
	tree := css_parser.Parse(esLog, source, ci.cp.parseOpts)

	diagnostics := convertESBuildMessagesToDiagnostics(esLog.Done(), containingPath, startLocation)
	if len(diagnostics) > 0 {
		ci.diagnostics = append(ci.diagnostics, diagnostics...)
	}
	return tree, ast.HasErrors(diagnostics)
}

// processImports resolves and inlines CSS @import rules into the AST.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes tree (css_ast.AST) which is the parsed CSS to process.
// Takes containingPath (string) which is the file path of the CSS being
// processed.
// Takes startLocation (ast.Location) which marks where processing began.
// Takes pathStack ([]string) which tracks visited paths to detect cycles.
//
// Returns css_ast.AST which is the tree with imports merged in reverse order.
// Returns error when an imported file cannot be collected or processed.
func (ci *cssInliner) processImports(ctx context.Context, tree css_ast.AST, containingPath string, startLocation ast.Location, pathStack []string) (css_ast.AST, error) {
	newPathStack := slices.Concat(pathStack, []string{containingPath})
	rulesToKeep, importsToMerge, err := ci.collectImportedASTs(ctx, tree, containingPath, startLocation, newPathStack)
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

	for i := len(importsToMerge) - 1; i >= 0; i-- {
		mergeASTs(&newTree, importsToMerge[i])
	}

	return newTree, nil
}

// collectImportedASTs processes a CSS AST to separate import rules from other
// rules and resolve the imported stylesheets.
//
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes tree (css_ast.AST) which is the CSS abstract syntax tree to process.
// Takes containingPath (string) which is the file path of the stylesheet.
// Takes startLocation (ast.Location) which marks the import's source position.
// Takes pathStack ([]string) which tracks visited paths to detect cycles.
//
// Returns []css_ast.Rule which contains all non-import rules from the tree.
// Returns []*css_ast.AST which contains the parsed ASTs of imported files.
// Returns error when a circular import is found or parsing fails.
func (ci *cssInliner) collectImportedASTs(ctx context.Context, tree css_ast.AST, containingPath string, startLocation ast.Location, pathStack []string) ([]css_ast.Rule, []*css_ast.AST, error) {
	var rulesToKeep []css_ast.Rule
	var importsToMerge []*css_ast.AST

	for _, rule := range tree.Rules {
		imp, isImport := rule.Data.(*css_ast.RAtImport)
		if !isImport {
			rulesToKeep = append(rulesToKeep, rule)
			continue
		}

		importedAST, err := ci.resolveAndParseImport(ctx, tree, imp, rule, containingPath, startLocation, pathStack)
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
// Takes ctx (context.Context) which controls cancellation and timeout.
// Takes tree (css_ast.AST) which provides the import records for resolution.
// Takes imp (*css_ast.RAtImport) which specifies the import rule to process.
// Takes rule (css_ast.Rule) which provides the original rule location.
// Takes containingPath (string) which is the path of the file that contains
// the import.
// Takes startLocation (ast.Location) which marks where the import appears.
// Takes pathStack ([]string) which tracks visited paths to find cycles.
//
// Returns *css_ast.AST which is the parsed and condition-wrapped imported CSS,
// or nil if the import could not be resolved or parsed.
// Returns error when a circular import is found.
func (ci *cssInliner) resolveAndParseImport(
	ctx context.Context,
	tree css_ast.AST,
	imp *css_ast.RAtImport,
	rule css_ast.Rule,
	containingPath string,
	startLocation ast.Location,
	pathStack []string,
) (*css_ast.AST, error) {
	importRecord := tree.ImportRecords[imp.ImportRecordIndex]
	importPath := importRecord.Path.Text

	resolvedPath, err := ci.cp.resolver.ResolveCSSPath(ctx, importPath, filepath.Dir(containingPath))
	if err != nil {
		diagnostic := ast.NewDiagnosticWithCode(ast.Error, err.Error(), importPath, annotator_dto.CodeCSSImportError, startLocation, containingPath)
		ci.diagnostics = append(ci.diagnostics, diagnostic)
		return nil, nil
	}

	importedContent, err := ci.fsReader.ReadFile(ctx, resolvedPath)
	if err != nil {
		diagnostic := ast.NewDiagnosticWithCode(ast.Error, err.Error(), importPath, annotator_dto.CodeCSSImportError, startLocation, containingPath)
		ci.diagnostics = append(ci.diagnostics, diagnostic)
		return nil, nil
	}

	importedAST, err := ci.parseRecursive(ctx, string(importedContent), resolvedPath, ast.Location{Line: 1, Column: 1, Offset: 0}, pathStack)
	if err != nil {
		return nil, fmt.Errorf("parsing imported CSS %q: %w", resolvedPath, err)
	}
	if importedAST == nil {
		return nil, nil
	}

	return wrapImportedASTWithConditions(importedAST, imp.ImportConditions, rule.Loc), nil
}

// getCSSInliner retrieves a cssInliner from the pool and prepares it for use.
//
// Takes cp (*CSSProcessor) which provides CSS processing.
// Takes fsReader (FSReaderPort) which provides file system access.
//
// Returns *cssInliner which is ready to use.
func getCSSInliner(cp *CSSProcessor, fsReader FSReaderPort) *cssInliner {
	ci, ok := cssInlinerPool.Get().(*cssInliner)
	if !ok {
		ci = &cssInliner{}
	}
	ci.cp = cp
	ci.fsReader = fsReader
	ci.cache = make(cssParseCache)
	ci.diagnostics = nil
	return ci
}

// putCSSInliner clears the cssInliner fields and returns it to the pool.
//
// Takes ci (*cssInliner) which is the inliner to reset and return.
func putCSSInliner(ci *cssInliner) {
	ci.cp = nil
	ci.fsReader = nil
	ci.cache = nil
	ci.diagnostics = nil
	cssInlinerPool.Put(ci)
}

// newCSSInliner creates a new inliner instance for a single top-level task.
//
// Takes cp (*CSSProcessor) which provides CSS processing features.
// Takes fsReader (FSReaderPort) which reads files from the file system.
//
// Returns *cssInliner which is the configured inliner ready for use.
func newCSSInliner(cp *CSSProcessor, fsReader FSReaderPort) *cssInliner {
	return &cssInliner{
		cp:          cp,
		fsReader:    fsReader,
		cache:       make(cssParseCache),
		diagnostics: nil,
	}
}

// mergeASTs joins a child AST into a parent AST by updating all token
// indexes. Child rules are added before parent rules so that imports appear
// first in the CSS output.
//
// Takes parent (*css_ast.AST) which receives the merged result.
// Takes child (*css_ast.AST) which provides the rules to add.
func mergeASTs(parent *css_ast.AST, child *css_ast.AST) {
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
// Takes rule (*css_ast.Rule) which is the CSS rule to adjust.
// Takes symbolOffset (uint32) which is added to symbol index values.
// Takes importRecordOffset (uint32) which is added to import record index
// values.
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
	case *css_ast.RUnknownAt:
		reIndexTokens(r.Prelude, symbolOffset, importRecordOffset)
		reIndexTokens(r.Block, symbolOffset, importRecordOffset)
	}
}

// reIndexSelectorRule updates symbol and import record indices for a selector
// rule and its nested rules.
//
// Takes r (*css_ast.RSelector) which is the selector rule to update.
// Takes symbolOffset (uint32) which is the offset to add to each symbol index.
// Takes importRecordOffset (uint32) which is the offset to add to each import
// record index.
func reIndexSelectorRule(r *css_ast.RSelector, symbolOffset, importRecordOffset uint32) {
	for i := range r.Selectors {
		reIndexSelector(&r.Selectors[i], symbolOffset, importRecordOffset)
	}
	reIndexRuleList(r.Rules, symbolOffset, importRecordOffset)
}

// reIndexKeyframesRule updates symbol references in a keyframes rule.
// It changes the keyframe name symbol and processes all nested rules within
// the keyframe blocks.
//
// Takes r (*css_ast.RAtKeyframes) which is the keyframes rule to update.
// Takes symbolOffset (uint32) which is the value to add to symbol references.
// Takes importRecordOffset (uint32) which is the offset for import records.
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
// Takes rules ([]css_ast.Rule) which contains the nested rules to update.
// Takes symbolOffset (uint32) which is added to symbol indices.
// Takes importRecordOffset (uint32) which is added to import record indices.
func reIndexAtRule(prelude []css_ast.Token, rules []css_ast.Rule, symbolOffset, importRecordOffset uint32) {
	reIndexTokens(prelude, symbolOffset, importRecordOffset)
	reIndexRuleList(rules, symbolOffset, importRecordOffset)
}

// reIndexRuleList updates symbol and import record indices for a list of CSS
// rules by adding the given offsets to each rule.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to update.
// Takes symbolOffset (uint32) which is the value added to each symbol index.
// Takes importRecordOffset (uint32) which is the value added to each import
// record index.
func reIndexRuleList(rules []css_ast.Rule, symbolOffset, importRecordOffset uint32) {
	for i := range rules {
		reIndexRule(&rules[i], symbolOffset, importRecordOffset)
	}
}

// reIndexSelector updates symbol references in a CSS complex selector.
//
// Takes selector (*css_ast.ComplexSelector) which is the selector to update.
// Takes symbolOffset (uint32) which is added to each symbol index.
// Takes importRecordOffset (uint32) which is added to each import record
// index.
func reIndexSelector(selector *css_ast.ComplexSelector, symbolOffset, importRecordOffset uint32) {
	for i := range selector.Selectors {
		reIndexCompoundSelector(&selector.Selectors[i], symbolOffset, importRecordOffset)
	}
}

// reIndexCompoundSelector updates symbol and import record references in a
// compound selector.
//
// Takes cs (*css_ast.CompoundSelector) which is the compound selector to
// update.
// Takes symbolOffset (uint32) which is the amount to add to each symbol index.
// Takes importRecordOffset (uint32) which is the amount to add to each import
// record index.
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

// reIndexTokens updates payload indices in CSS tokens when merging symbol and
// import record tables from different sources.
//
// Takes tokens ([]css_ast.Token) which is the slice of tokens to update.
// Takes symbolOffset (uint32) which is added to symbol payload indices.
// Takes importRecordOffset (uint32) which is added to URL payload indices.
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

// cloneAST creates a deep copy of an AST for use in caching.
//
// When original is nil, returns nil.
//
// Takes original (*css_ast.AST) which is the AST to copy.
//
// Returns *css_ast.AST which is a new copy of the original AST.
func cloneAST(original *css_ast.AST) *css_ast.AST {
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

// cloneRule creates a deep copy of a CSS rule.
//
// Takes original (css_ast.Rule) which is the rule to copy.
//
// Returns css_ast.Rule which is a new rule with the same data and source
// location. All nested structures are copied rather than shared.
func cloneRule(original css_ast.Rule) css_ast.Rule {
	return css_ast.Rule{
		Data: cloneR(original.Data),
		Loc:  original.Loc,
	}
}

// cloneR creates a deep copy of a CSS rule node.
//
// Takes original (css_ast.R) which is the CSS rule node to copy.
//
// Returns css_ast.R which is a deep copy of the original. Returns nil if the
// original is nil or if the rule type is not recognised.
func cloneR(original css_ast.R) css_ast.R {
	if original == nil {
		return nil
	}
	switch r := original.(type) {
	case *css_ast.RAtCharset:
		return cloneRAtCharset(r)
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
		return cloneRDeclaration(r)
	case *css_ast.RBadDeclaration:
		return cloneRBadDeclaration(r)
	case *css_ast.RComment:
		return cloneRComment(r)
	case *css_ast.RAtLayer:
		return cloneRAtLayer(r)
	case *css_ast.RAtMedia:
		return cloneRAtMedia(r)
	default:
		return nil
	}
}

// cloneRAtCharset creates a shallow copy of an @charset rule.
//
// Takes r (*css_ast.RAtCharset) which is the rule to copy.
//
// Returns *css_ast.RAtCharset which is a new instance with the same values.
func cloneRAtCharset(r *css_ast.RAtCharset) *css_ast.RAtCharset {
	return new(*r)
}

// cloneRAtImport creates a deep copy of a CSS @import rule.
//
// Takes r (*css_ast.RAtImport) which is the rule to copy.
//
// Returns *css_ast.RAtImport which is a new copy that can be modified without
// affecting the original.
func cloneRAtImport(r *css_ast.RAtImport) *css_ast.RAtImport {
	clone := *r
	if r.ImportConditions != nil {
		conditionsClone, _ := r.ImportConditions.CloneWithImportRecords(nil, nil)
		clone.ImportConditions = &conditionsClone
	}
	return &clone
}

// cloneRAtKeyframes creates a deep copy of a CSS @keyframes at-rule.
//
// Takes r (*css_ast.RAtKeyframes) which is the keyframes rule to copy.
//
// Returns *css_ast.RAtKeyframes which is a new copy that can be modified
// without affecting the original.
func cloneRAtKeyframes(r *css_ast.RAtKeyframes) *css_ast.RAtKeyframes {
	clone := *r
	clone.Blocks = make([]css_ast.KeyframeBlock, len(r.Blocks))
	for i, block := range r.Blocks {
		clone.Blocks[i] = cloneKeyframeBlock(block)
	}
	return &clone
}

// cloneRKnownAt creates a deep copy of a CSS at-rule.
//
// Takes r (*css_ast.RKnownAt) which is the at-rule to copy.
//
// Returns *css_ast.RKnownAt which is a new copy that does not share memory
// with the original.
func cloneRKnownAt(r *css_ast.RKnownAt) *css_ast.RKnownAt {
	clone := *r
	clone.Prelude = cloneTokens(r.Prelude)
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRUnknownAt creates a deep copy of an unknown at-rule.
//
// Takes r (*css_ast.RUnknownAt) which is the at-rule to copy.
//
// Returns *css_ast.RUnknownAt which is a new copy with cloned prelude and
// block tokens.
func cloneRUnknownAt(r *css_ast.RUnknownAt) *css_ast.RUnknownAt {
	clone := *r
	clone.Prelude = cloneTokens(r.Prelude)
	clone.Block = cloneTokens(r.Block)
	return &clone
}

// cloneRSelector creates a deep copy of a CSS selector rule.
//
// Takes r (*css_ast.RSelector) which is the selector rule to copy.
//
// Returns *css_ast.RSelector which is a new copy that does not share memory
// with the original.
func cloneRSelector(r *css_ast.RSelector) *css_ast.RSelector {
	clone := *r
	clone.Selectors = make([]css_ast.ComplexSelector, len(r.Selectors))
	for i, selector := range r.Selectors {
		clone.Selectors[i] = selector.Clone()
	}
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRDeclaration creates a deep copy of a CSS declaration rule.
//
// Takes r (*css_ast.RDeclaration) which is the declaration to copy.
//
// Returns *css_ast.RDeclaration which is a new copy that can be changed
// without affecting the original.
func cloneRDeclaration(r *css_ast.RDeclaration) *css_ast.RDeclaration {
	clone := *r
	clone.Value = cloneTokens(r.Value)
	return &clone
}

// cloneRBadDeclaration creates a copy of a bad declaration rule.
//
// Takes r (*css_ast.RBadDeclaration) which is the rule to copy.
//
// Returns *css_ast.RBadDeclaration which is a copy of the rule.
func cloneRBadDeclaration(r *css_ast.RBadDeclaration) *css_ast.RBadDeclaration {
	clone := *r
	clone.Tokens = cloneTokens(r.Tokens)
	return &clone
}

// cloneRComment creates a shallow copy of a CSS comment rule.
//
// Takes r (*css_ast.RComment) which is the comment rule to copy.
//
// Returns *css_ast.RComment which is a new copy of the rule.
func cloneRComment(r *css_ast.RComment) *css_ast.RComment {
	return new(*r)
}

// cloneRAtLayer creates a deep copy of a CSS @layer at-rule.
//
// Takes r (*css_ast.RAtLayer) which is the at-rule to copy.
//
// Returns *css_ast.RAtLayer which is a new copy that shares no memory with
// the original.
func cloneRAtLayer(r *css_ast.RAtLayer) *css_ast.RAtLayer {
	clone := *r
	clone.Names = make([][]string, len(r.Names))
	for i, name := range r.Names {
		clone.Names[i] = append([]string(nil), name...)
	}
	clone.Rules = cloneRules(r.Rules)
	return &clone
}

// cloneRAtMedia creates a full copy of a CSS @media rule.
//
// Takes r (*css_ast.RAtMedia) which is the media rule to copy.
//
// Returns *css_ast.RAtMedia which is a new copy that can be changed without
// affecting the original.
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
// Returns []css_ast.Rule which is a new slice with deep copies of all rules.
func cloneRules(rules []css_ast.Rule) []css_ast.Rule {
	cloned := make([]css_ast.Rule, len(rules))
	for i, rule := range rules {
		cloned[i] = cloneRule(rule)
	}
	return cloned
}

// cloneKeyframeBlock creates a deep copy of a CSS keyframe block.
//
// Takes original (css_ast.KeyframeBlock) which is the keyframe block to copy.
//
// Returns css_ast.KeyframeBlock which is a new copy with its own selectors
// and rules.
func cloneKeyframeBlock(original css_ast.KeyframeBlock) css_ast.KeyframeBlock {
	clone := original
	clone.Selectors = append([]string(nil), original.Selectors...)
	clone.Rules = make([]css_ast.Rule, len(original.Rules))
	for i, rule := range original.Rules {
		clone.Rules[i] = cloneRule(rule)
	}
	return clone
}

// cloneTokens creates a deep copy of a CSS token slice.
//
// Takes original ([]css_ast.Token) which is the token slice to copy.
//
// Returns []css_ast.Token which is a new slice with copied tokens. Child
// tokens are cloned using recursion.
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

// wrapImportedASTWithConditions wraps the imported AST's rules with
// @layer, @supports, and/or @media rules based on the
// ImportConditions from the @import statement.
//
// Takes importedAST (*css_ast.AST) which is the parsed CSS to wrap.
// Takes conditions (*css_ast.ImportConditions) which specifies the
// wrapping rules from the @import statement.
// Takes loc (es_logger.Loc) which provides the source location for
// the generated wrapper rules.
//
// Returns *css_ast.AST which is a new AST with the rules wrapped
// according to the conditions, or the original AST if conditions is
// nil.
func wrapImportedASTWithConditions(importedAST *css_ast.AST, conditions *css_ast.ImportConditions, loc es_logger.Loc) *css_ast.AST {
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

	wrapped := cloneAST(importedAST)
	wrapped.Rules = rules
	return wrapped
}

// extractLayerNamesFromTokens parses layer names from CSS import tokens.
// It handles forms such as "layer", "layer(name)", or "layer(foo.bar)".
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to parse.
//
// Returns [][]string which contains the parsed layer name parts. Returns a
// slice with one empty slice when no layer function is found.
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

// parseLayerNameFromChildren extracts layer names from CSS token children.
// It handles dotted names like "foo.bar" which become []string{"foo", "bar"}.
//
// Takes children ([]css_ast.Token) which contains the tokens to parse.
//
// Returns [][]string which contains the extracted layer name parts. Returns a
// slice with an empty inner slice if no identifier tokens are found.
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
