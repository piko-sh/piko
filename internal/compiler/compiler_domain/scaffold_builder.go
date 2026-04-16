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
	"errors"
	"fmt"
	"html"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	es_logger "piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	_ ScaffoldBuilder = (*scaffoldBuilder)(nil)

	// selfClosingTags maps HTML void element tag names that must not have a closing tag.
	selfClosingTags = map[string]bool{
		"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true,
		"input": true, "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
	}

	// interactivePseudoClasses maps CSS pseudo-classes that
	// require user interaction and are stripped during compilation.
	interactivePseudoClasses = map[string]bool{
		"hover":         true,
		"focus":         true,
		"focus-visible": true,
		"focus-within":  true,
		"active":        true,
		"visited":       true,
		"target":        true,
	}

	// propertiesToStrip lists CSS properties to remove from stylesheets.
	propertiesToStrip = map[string]bool{
		"transition":                 true,
		"transition-property":        true,
		"transition-duration":        true,
		"transition-timing-function": true,
		"transition-delay":           true,
		"animation":                  true,
		"animation-name":             true,
		"animation-duration":         true,
		"animation-timing-function":  true,
		"animation-delay":            true,
		"animation-iteration-count":  true,
		"animation-direction":        true,
		"animation-fill-mode":        true,
		"animation-play-state":       true,
		"cursor":                     true,
		"pointer-events":             true,
		"user-select":                true,
		"outline":                    true,
		"outline-width":              true,
		"outline-style":              true,
		"outline-color":              true,
		"outline-offset":             true,
		"will-change":                true,
	}
)

// scaffoldConfigKey is the context key for ScaffoldBuilderConfig.
type scaffoldConfigKey struct{}

// ScaffoldBuilder builds static HTML scaffolds from template ASTs.
type ScaffoldBuilder interface {
	// BuildStaticScaffold builds the static HTML scaffold from a template AST.
	//
	// Takes tAST (*ast_domain.TemplateAST) which is the parsed template structure.
	// Takes fullCSS (string) which contains the complete CSS styles to embed.
	//
	// Returns string which is the generated HTML scaffold.
	// Returns error when scaffold generation fails.
	BuildStaticScaffold(ctx context.Context, tAST *ast_domain.TemplateAST,
		fullCSS string) (string, error)
}

// ScaffoldBuilderConfig holds settings for the scaffold builder.
type ScaffoldBuilderConfig struct {
	// CSSTreeShakingSafelist lists CSS class names that should never be
	// tree-shaken, even when CSSTreeShaking is enabled. Use for dynamically
	// added classes; specify names without the leading dot.
	CSSTreeShakingSafelist []string

	// CSSTreeShaking enables removal of unused CSS rules based on static HTML
	// analysis. When disabled (default), all CSS is preserved.
	CSSTreeShaking bool
}

// scaffoldBuilder implements the ScaffoldBuilder interface.
type scaffoldBuilder struct {
	// config holds the scaffold builder settings, including CSS tree-shaking options.
	config ScaffoldBuilderConfig
}

// usedSelectors tracks CSS selectors found in static HTML for tree-shaking.
type usedSelectors struct {
	// classes holds CSS class names found in HTML class attributes.
	classes map[string]bool

	// ids maps HTML id attribute values to true for quick lookup.
	ids map[string]bool

	// tags maps lowercase HTML tag names to true when the tag is used.
	tags map[string]bool
}

// BuildStaticScaffold generates a static HTML scaffold from a template AST,
// removing unused CSS rules when CSSTreeShaking is enabled in config and
// preserving all CSS when disabled (default).
//
// Takes tAST (*ast_domain.TemplateAST) which provides the template structure.
// Takes fullCSS (string) which contains the complete CSS.
//
// Returns string which is the assembled HTML scaffold with CSS.
// Returns error when tAST is nil or static HTML generation fails.
func (builder *scaffoldBuilder) BuildStaticScaffold(ctx context.Context, tAST *ast_domain.TemplateAST, fullCSS string) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "BuildStaticScaffold",
		logger_domain.String("service", "ScaffoldBuilder"),
		logger_domain.Bool("cssTreeShaking", builder.config.CSSTreeShaking),
	)
	defer span.End()

	ScaffoldBuildCount.Add(ctx, 1)

	if tAST == nil {
		err := errors.New("cannot build scaffold from a nil TemplateAST")
		ScaffoldBuildErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to build scaffold")
		return "", err
	}

	if !builder.config.CSSTreeShaking {
		l.Trace("Building static scaffold without CSS tree-shaking (disabled)")

		staticHTML, _, err := buildStaticHTML(ctx, tAST)
		if err != nil {
			ScaffoldBuildErrorCount.Add(ctx, 1)
			l.ReportError(span, err, "Failed to write static nodes")
			return "", fmt.Errorf("building static HTML: %w", err)
		}

		scaffold := assembleFinalScaffold(staticHTML, fullCSS)
		l.Trace("Successfully built final DSD scaffold", logger_domain.Int("totalSize", len(scaffold)))
		return scaffold, nil
	}

	l.Trace("Building static scaffold with CSS tree-shaking enabled")

	staticHTML, selectors, err := buildStaticHTML(ctx, tAST)
	if err != nil {
		ScaffoldBuildErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to write static nodes")
		return "", fmt.Errorf("building static HTML: %w", err)
	}

	for _, cls := range builder.config.CSSTreeShakingSafelist {
		selectors.classes[cls] = true
	}

	l.Trace("Generated static HTML part of scaffold",
		logger_domain.Int("htmlSize", len(staticHTML)),
		logger_domain.Int("usedClasses", len(selectors.classes)),
		logger_domain.Int("usedIDs", len(selectors.ids)),
		logger_domain.Int("usedTags", len(selectors.tags)),
		logger_domain.Int("safelistClasses", len(builder.config.CSSTreeShakingSafelist)),
	)

	treeShakenCSS := treeShakeCSSWithFallback(ctx, fullCSS, selectors)
	scaffold := assembleFinalScaffold(staticHTML, treeShakenCSS)

	l.Trace("Successfully built final DSD scaffold", logger_domain.Int("totalSize", len(scaffold)))
	return scaffold, nil
}

// WithScaffoldConfig returns a new context with the given ScaffoldBuilderConfig,
// passing configuration through the context to the scaffold builder.
//
// Takes config (ScaffoldBuilderConfig) which specifies the
// scaffold builder settings.
//
// Returns context.Context which contains the configuration value.
func WithScaffoldConfig(ctx context.Context, config ScaffoldBuilderConfig) context.Context {
	return context.WithValue(ctx, scaffoldConfigKey{}, config)
}

// GetScaffoldConfig retrieves the scaffold builder settings from the context.
//
// When no config was set in the context, returns an empty config with
// tree-shaking disabled.
//
// Returns ScaffoldBuilderConfig which holds the scaffold builder settings.
func GetScaffoldConfig(ctx context.Context) ScaffoldBuilderConfig {
	if config, ok := ctx.Value(scaffoldConfigKey{}).(ScaffoldBuilderConfig); ok {
		return config
	}
	return ScaffoldBuilderConfig{}
}

// NewScaffoldBuilder creates a new ScaffoldBuilder with the given configuration.
// If no config is provided, tree-shaking is disabled by default.
//
// Takes config (...ScaffoldBuilderConfig) which optionally provides configuration.
// Only the first config is used if multiple are provided.
//
// Returns ScaffoldBuilder which is ready for use.
func NewScaffoldBuilder(config ...ScaffoldBuilderConfig) ScaffoldBuilder {
	var builderConfig ScaffoldBuilderConfig
	if len(config) > 0 {
		builderConfig = config[0]
	}
	return &scaffoldBuilder{config: builderConfig}
}

// newUsedSelectors creates a new empty set for tracking CSS selectors.
//
// Returns *usedSelectors which has empty maps for classes, IDs, and tags.
func newUsedSelectors() *usedSelectors {
	return &usedSelectors{
		classes: make(map[string]bool),
		ids:     make(map[string]bool),
		tags:    make(map[string]bool),
	}
}

// buildStaticHTML renders all root nodes to HTML and collects CSS selectors.
//
// Takes tAST (*ast_domain.TemplateAST) which contains the root nodes to render.
//
// Returns string which is the rendered HTML content.
// Returns *usedSelectors which holds the CSS selectors found during rendering.
// Returns error when a static node fails to render.
func buildStaticHTML(_ context.Context, tAST *ast_domain.TemplateAST) (string, *usedSelectors, error) {
	var htmlBuilder strings.Builder
	selectors := newUsedSelectors()

	var err error
	for _, rootNode := range tAST.RootNodes {
		if err = writeStaticNodeAndCollectSelectors(&htmlBuilder, rootNode, selectors); err != nil {
			err = fmt.Errorf("failed to write static node during scaffold build: %w", err)
			break
		}
	}

	if err != nil {
		return "", nil, err
	}
	return htmlBuilder.String(), selectors, nil
}

// treeShakeCSSWithFallback removes unused CSS rules from the given CSS string.
// If removal fails, it returns the original CSS unchanged.
//
// Takes fullCSS (string) which is the complete CSS to process.
// Takes selectors (*usedSelectors) which contains the selectors to keep.
//
// Returns string which is the processed CSS, or the original CSS if processing
// fails.
func treeShakeCSSWithFallback(ctx context.Context, fullCSS string, selectors *usedSelectors) string {
	ctx, l := logger_domain.From(ctx, log)
	if strings.TrimSpace(fullCSS) == "" {
		return ""
	}

	treeShakenCSS, err := shakeCSS(ctx, fullCSS, selectors)
	if err != nil {
		l.Warn("CSS tree-shaking failed, falling back to using full component CSS.",
			logger_domain.String("error", err.Error()),
		)
		return fullCSS
	}
	return treeShakenCSS
}

// assembleFinalScaffold builds the final shadow DOM template with CSS and HTML.
//
// Takes staticHTML (string) which contains the pre-rendered HTML content.
// Takes treeShakenCSS (string) which contains the optimised CSS styles.
//
// Returns string which is the complete shadow DOM template with styles inside.
func assembleFinalScaffold(staticHTML, treeShakenCSS string) string {
	var finalBuilder strings.Builder
	finalBuilder.WriteString(`<template shadowrootmode="open">`)
	if treeShakenCSS != "" {
		finalBuilder.WriteString(`<style>`)
		finalBuilder.WriteString(treeShakenCSS)
		finalBuilder.WriteString(`</style>`)
	}
	finalBuilder.WriteString(staticHTML)
	finalBuilder.WriteString(`</template>`)
	return finalBuilder.String()
}

// writeStaticNodeAndCollectSelectors writes a static template node to the
// builder and collects any CSS selectors used. Nodes with dynamic directives
// (if or for) are skipped.
//
// Takes builder (*strings.Builder) which receives the HTML output.
// Takes node (*ast_domain.TemplateNode) which is the template node to write.
// Takes selectors (*usedSelectors) which collects the CSS selectors found.
//
// Returns error when writing an element node fails.
func writeStaticNodeAndCollectSelectors(builder *strings.Builder, node *ast_domain.TemplateNode, selectors *usedSelectors) error {
	if node.DirIf != nil || node.DirFor != nil {
		return nil
	}

	switch node.NodeType {
	case ast_domain.NodeText:
		builder.WriteString(html.EscapeString(node.TextContent))
	case ast_domain.NodeElement:
		return writeElementNode(builder, node, selectors)
	default:
	}
	return nil
}

// writeElementNode writes an HTML element node to the builder and records its
// selectors.
//
// Takes builder (*strings.Builder) which receives the rendered HTML output.
// Takes node (*ast_domain.TemplateNode) which is the element node to render.
// Takes selectors (*usedSelectors) which collects tag and attribute selectors.
//
// Returns error when child nodes fail to render.
func writeElementNode(builder *strings.Builder, node *ast_domain.TemplateNode, selectors *usedSelectors) error {
	lowerTagName := strings.ToLower(node.TagName)
	selectors.tags[lowerTagName] = true
	builder.WriteString("<" + lowerTagName)

	writeAttributesAndCollect(builder, node.Attributes, selectors)
	for _, arg := range slices.Sorted(maps.Keys(node.TimelineDirectives)) {
		builder.WriteString(` p-timeline-` + arg + `=""`)
	}
	builder.WriteString(">")

	if selfClosingTags[lowerTagName] && len(node.Children) == 0 {
		return nil
	}

	if err := writeChildNodes(builder, node.Children, selectors); err != nil {
		return fmt.Errorf("writing child nodes for <%s>: %w", lowerTagName, err)
	}
	builder.WriteString("</" + lowerTagName + ">")
	return nil
}

// writeAttributesAndCollect writes HTML attributes to a builder and gathers
// class and id selectors.
//
// Takes builder (*strings.Builder) which receives the formatted attribute
// output.
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// write.
// Takes selectors (*usedSelectors) which gathers class and id selectors found
// in the attributes.
func writeAttributesAndCollect(builder *strings.Builder, attrs []ast_domain.HTMLAttribute, selectors *usedSelectors) {
	for attributeIndex := range attrs {
		attr := &attrs[attributeIndex]
		if strings.HasPrefix(attr.Name, "p-") || strings.HasPrefix(attr.Name, ":") {
			continue
		}
		collectAttributeSelectors(attr, selectors)
		builder.WriteString(` ` + attr.Name + `="` + html.EscapeString(attr.Value) + `"`)
	}
}

// collectAttributeSelectors gathers class and id selectors from an HTML
// attribute.
//
// Takes attr (*ast_domain.HTMLAttribute) which is the HTML attribute to extract
// selectors from.
// Takes selectors (*usedSelectors) which is the collection to add found
// selectors to.
func collectAttributeSelectors(attr *ast_domain.HTMLAttribute, selectors *usedSelectors) {
	attributeNameLower := strings.ToLower(attr.Name)
	switch attributeNameLower {
	case "class":
		for cls := range strings.FieldsSeq(attr.Value) {
			selectors.classes[cls] = true
		}
	case "id":
		selectors.ids[attr.Value] = true
	}
}

// writeChildNodes writes each child node to the builder in order.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes children ([]*ast_domain.TemplateNode) which holds the nodes to write.
// Takes selectors (*usedSelectors) which collects CSS selectors found.
//
// Returns error when writing any child node fails.
func writeChildNodes(builder *strings.Builder, children []*ast_domain.TemplateNode, selectors *usedSelectors) error {
	for _, child := range children {
		if err := writeStaticNodeAndCollectSelectors(builder, child, selectors); err != nil {
			return fmt.Errorf("writing static child node: %w", err)
		}
	}
	return nil
}

// shakeCSS removes unused CSS rules based on the provided selectors.
//
// Takes css (string) which is the raw CSS content to process.
// Takes selectors (*usedSelectors) which contains the selectors in use.
//
// Returns string which is the reduced CSS with unused rules removed.
// Returns error when CSS parsing fails or rule filtering fails.
func shakeCSS(ctx context.Context, css string, selectors *usedSelectors) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "shakeCSS")
	defer span.End()

	CSSTreeShakingCount.Add(ctx, 1)

	esLog := es_logger.NewDeferLog(es_logger.DeferLogNoVerboseOrDebug, nil)
	tree := css_parser.Parse(esLog, es_logger.Source{Contents: css}, css_parser.Options{})
	if errs := esLog.Done(); len(errs) > 0 {
		err := fmt.Errorf("css parsing failed: %s", errs[0].Data.Text)
		CSSTreeShakingErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "CSS parsing failed")
		return "", err
	}

	tree.Rules = filterRules(ctx, tree.Rules, selectors, tree.Symbols)

	usedCSSVars := make(map[string]bool)
	collectUsedVariables(tree.Rules, usedCSSVars)

	tree.Rules = stripUnusedDeclarations(tree.Rules, usedCSSVars)

	printOpts := css_printer.Options{MinifyWhitespace: true}
	symMap := es_ast.NewSymbolMap(1)
	symMap.SymbolsForSource[0] = tree.Symbols
	result := css_printer.Print(tree, symMap, printOpts)
	return string(result.CSS), nil
}

// filterRules removes CSS rules that are not used.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to check.
// Takes selectors (*usedSelectors) which tracks which selectors are in use.
// Takes symbols ([]es_ast.Symbol) which provides the JavaScript symbols to
// match against.
//
// Returns []css_ast.Rule which contains only the rules that are needed.
func filterRules(ctx context.Context, rules []css_ast.Rule, selectors *usedSelectors, symbols []es_ast.Symbol) []css_ast.Rule {
	var keptRules []css_ast.Rule
	for _, rule := range rules {
		if shouldKeep := shouldKeepRule(ctx, rule, selectors, symbols); shouldKeep {
			keptRules = append(keptRules, rule)
		}
	}
	return keptRules
}

// shouldKeepRule checks whether a CSS rule should be kept after tree-shaking.
//
// Takes rule (css_ast.Rule) which is the CSS rule to check.
// Takes selectors (*usedSelectors) which tracks which selectors are in use.
// Takes symbols ([]es_ast.Symbol) which contains the JavaScript symbols.
//
// Returns bool which is true if the rule should be kept, false otherwise.
func shouldKeepRule(ctx context.Context, rule css_ast.Rule, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	switch r := rule.Data.(type) {
	case *css_ast.RSelector:
		return filterSelectorRule(ctx, r, selectors, symbols)
	case *css_ast.RKnownAt:
		return filterAtRule(ctx, r, selectors, symbols)
	default:
		return true
	}
}

// filterSelectorRule checks a selector rule and returns whether to keep it.
//
// Takes r (*css_ast.RSelector) which is the selector rule to check.
// Takes selectors (*usedSelectors) which holds the set of used selectors.
// Takes symbols ([]es_ast.Symbol) which provides symbol data for matching.
//
// Returns bool which is true if the rule has matching selectors and should be
// kept.
func filterSelectorRule(ctx context.Context, r *css_ast.RSelector, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	keptSelectors := filterMatchingSelectors(r.Selectors, selectors, symbols)
	if len(keptSelectors) == 0 {
		return false
	}
	r.Selectors = keptSelectors
	r.Rules = filterRules(ctx, r.Rules, selectors, symbols)
	return true
}

// filterMatchingSelectors returns only the selectors that match static content.
//
// Takes selectors ([]css_ast.ComplexSelector) which contains the CSS selectors
// to filter.
// Takes usedSel (*usedSelectors) which tracks selectors used in the content.
// Takes symbols ([]es_ast.Symbol) which provides the symbol table for matching.
//
// Returns []css_ast.ComplexSelector which contains the selectors that matched.
func filterMatchingSelectors(selectors []css_ast.ComplexSelector, usedSel *usedSelectors, symbols []es_ast.Symbol) []css_ast.ComplexSelector {
	var kept []css_ast.ComplexSelector
	for _, selector := range selectors {
		if selectorMatchesStaticContent(selector, usedSel, symbols) {
			kept = append(kept, selector)
		}
	}
	return kept
}

// filterAtRule checks whether an at-rule should be kept after filtering.
//
// Takes r (*css_ast.RKnownAt) which is the at-rule to check.
// Takes selectors (*usedSelectors) which tracks selectors in use.
// Takes symbols ([]es_ast.Symbol) which provides the symbol table.
//
// Returns bool which is true if the at-rule has rules remaining after
// filtering.
func filterAtRule(ctx context.Context, r *css_ast.RKnownAt, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	if r.Rules == nil {
		return false
	}
	r.Rules = filterRules(ctx, r.Rules, selectors, symbols)
	return len(r.Rules) > 0
}

// collectUsedVariables scans CSS rules and records any CSS variables that are
// used. It handles different rule types and calls itself for nested rules.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to scan.
// Takes usedVars (map[string]bool) which stores the variable names found.
func collectUsedVariables(rules []css_ast.Rule, usedVars map[string]bool) {
	for _, rule := range rules {
		switch r := rule.Data.(type) {
		case *css_ast.RSelector:
			collectUsedVariables(r.Rules, usedVars)
		case *css_ast.RKnownAt:
			collectUsedVariables(r.Rules, usedVars)
		case *css_ast.RDeclaration:
			collectVariablesFromTokens(r.Value, usedVars)
		}
	}
}

// collectVariablesFromTokens finds CSS variable references within tokens.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to scan.
// Takes usedVars (map[string]bool) which collects the variable names found.
func collectVariablesFromTokens(tokens []css_ast.Token, usedVars map[string]bool) {
	for _, token := range tokens {
		if !isVarFunction(token) {
			continue
		}
		if varName := extractVarName(token.Children); varName != "" {
			usedVars[varName] = true
		}
	}
}

// isVarFunction checks whether a token is a CSS var() function call.
//
// Takes token (css_ast.Token) which is the CSS token to check.
//
// Returns bool which is true when the token is a var() function.
func isVarFunction(token css_ast.Token) bool {
	return token.Kind == css_lexer.TFunction &&
		strings.EqualFold(token.Text, "var") &&
		token.Children != nil
}

// extractVarName gets the variable name from a CSS var() function.
//
// Takes children (*[]css_ast.Token) which holds the tokens from a CSS var()
// function call.
//
// Returns string which is the variable name (starting with "--"), or an empty
// string if no valid variable name is found.
func extractVarName(children *[]css_ast.Token) string {
	if children == nil {
		return ""
	}
	for _, child := range *children {
		if child.Kind == css_lexer.TWhitespace {
			continue
		}
		if strings.HasPrefix(child.Text, "--") {
			return child.Text
		}
		break
	}
	return ""
}

// stripUnusedDeclarations removes unused CSS custom property declarations and
// strips properties listed in propertiesToStrip.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to filter.
// Takes usedVars (map[string]bool) which tracks used CSS custom properties.
//
// Returns []css_ast.Rule which contains the filtered rules with unused
// declarations removed.
func stripUnusedDeclarations(rules []css_ast.Rule, usedVars map[string]bool) []css_ast.Rule {
	var cleanedRules []css_ast.Rule
	for _, rule := range rules {
		switch r := rule.Data.(type) {
		case *css_ast.RSelector:
			r.Rules = stripUnusedDeclarations(r.Rules, usedVars)
			cleanedRules = append(cleanedRules, rule)
		case *css_ast.RKnownAt:
			r.Rules = stripUnusedDeclarations(r.Rules, usedVars)
			cleanedRules = append(cleanedRules, rule)
		case *css_ast.RDeclaration:
			key := strings.ToLower(r.KeyText)
			if propertiesToStrip[key] {
				continue
			}
			if strings.HasPrefix(key, "--") && !usedVars[r.KeyText] {
				continue
			}
			cleanedRules = append(cleanedRules, rule)
		default:
			cleanedRules = append(cleanedRules, rule)
		}
	}
	return cleanedRules
}

// selectorMatchesStaticContent checks whether all compound selectors in a
// complex selector could match the static content.
//
// Takes selector (css_ast.ComplexSelector) which is the selector to check.
// Takes selectors (*usedSelectors) which tracks selectors used in content.
// Takes symbols ([]es_ast.Symbol) which provides symbol information.
//
// Returns bool which is true if all parts of the selector could match.
func selectorMatchesStaticContent(selector css_ast.ComplexSelector, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	for _, component := range selector.Selectors {
		if !compoundSelectorIsPlausible(component, selectors, symbols) {
			return false
		}
	}
	return true
}

// compoundSelectorIsPlausible checks whether a compound CSS selector could
// match any element in the static content.
//
// Takes component (css_ast.CompoundSelector) which is the selector to check.
// Takes selectors (*usedSelectors) which holds the tags, classes, and IDs
// found in the content.
// Takes symbols ([]es_ast.Symbol) which provides symbol data for working out
// class and ID references.
//
// Returns bool which is true if the selector might match content, or false if
// it cannot match.
func compoundSelectorIsPlausible(component css_ast.CompoundSelector, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	if hasInteractivePseudo(component) {
		return false
	}
	if isHostSelector(component) {
		return true
	}
	if !typeMatchesUsedTags(component, selectors) {
		return false
	}
	if !classMatchesUsedClasses(component, selectors, symbols) {
		return false
	}
	return idMatchesUsedIDs(component, selectors, symbols)
}

// hasInteractivePseudo checks whether a compound selector contains any
// interactive pseudo-class such as :hover, :focus, or :active.
//
// Takes component (css_ast.CompoundSelector) which is the selector to check.
//
// Returns bool which is true if an interactive pseudo-class is found.
func hasInteractivePseudo(component css_ast.CompoundSelector) bool {
	for _, subclass := range component.SubclassSelectors {
		if pseudo, ok := subclass.Data.(*css_ast.SSPseudoClass); ok {
			if interactivePseudoClasses[pseudo.Name] {
				return true
			}
		}
	}
	return false
}

// typeMatchesUsedTags checks if a type selector matches any known used tags.
//
// Takes component (css_ast.CompoundSelector) which is the selector to check.
// Takes selectors (*usedSelectors) which holds the set of used tag names.
//
// Returns bool which is true if there is no type selector, or if the tag name
// is in the used selectors set.
func typeMatchesUsedTags(component css_ast.CompoundSelector, selectors *usedSelectors) bool {
	if component.TypeSelector == nil {
		return true
	}
	return selectors.tags[strings.ToLower(component.TypeSelector.Name.Text)]
}

// classMatchesUsedClasses checks if any class selector matches used classes.
//
// Takes component (css_ast.CompoundSelector) which is the selector to check.
// Takes selectors (*usedSelectors) which contains the set of used class names.
// Takes symbols ([]es_ast.Symbol) which provides symbol information for lookup.
//
// Returns bool which is true if the component has no class selectors or if any
// class selector matches a used class.
func classMatchesUsedClasses(component css_ast.CompoundSelector, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	classSelectors := getClassesFromComponent(component, symbols)
	if len(classSelectors) == 0 {
		return true
	}
	return anyMatches(classSelectors, selectors.classes)
}

// idMatchesUsedIDs checks if any ID selector in a component matches a used ID.
//
// Takes component (css_ast.CompoundSelector) which is the CSS selector to
// check.
// Takes selectors (*usedSelectors) which holds the set of used IDs.
// Takes symbols ([]es_ast.Symbol) which provides symbol data for matching.
//
// Returns bool which is true if the component has no ID selectors, or if any
// ID selector matches a used ID.
func idMatchesUsedIDs(component css_ast.CompoundSelector, selectors *usedSelectors, symbols []es_ast.Symbol) bool {
	idSelectors := getIDsFromComponent(component, symbols)
	if len(idSelectors) == 0 {
		return true
	}
	return anyMatches(idSelectors, selectors.ids)
}

// anyMatches checks if any item in the slice exists in the map.
//
// Takes items ([]string) which is the slice of strings to check.
// Takes matches (map[string]bool) which contains the set of valid matches.
//
// Returns bool which is true if any item exists in matches.
func anyMatches(items []string, matches map[string]bool) bool {
	for _, item := range items {
		if matches[item] {
			return true
		}
	}
	return false
}

// getClassesFromComponent extracts CSS class names from a compound selector.
//
// Takes component (css_ast.CompoundSelector) which is the selector to extract
// classes from.
// Takes symbols ([]es_ast.Symbol) which provides the symbol table for finding
// original class names.
//
// Returns []string which contains the original names of all class selectors
// found in the component.
func getClassesFromComponent(component css_ast.CompoundSelector, symbols []es_ast.Symbol) []string {
	var classes []string
	for _, sub := range component.SubclassSelectors {
		if ssClass, ok := sub.Data.(*css_ast.SSClass); ok {
			if int(ssClass.Name.Ref.InnerIndex) < len(symbols) {
				classes = append(classes, symbols[ssClass.Name.Ref.InnerIndex].OriginalName)
			}
		}
	}
	return classes
}

// getIDsFromComponent extracts CSS ID selector names from a compound selector.
//
// Takes component (css_ast.CompoundSelector) which contains the CSS selector
// to check.
// Takes symbols ([]es_ast.Symbol) which provides the symbol table for name
// lookup.
//
// Returns []string which contains the original names of any ID selectors found
// in the component.
func getIDsFromComponent(component css_ast.CompoundSelector, symbols []es_ast.Symbol) []string {
	var ids []string
	for _, sub := range component.SubclassSelectors {
		if ssHash, ok := sub.Data.(*css_ast.SSHash); ok {
			if int(ssHash.Name.Ref.InnerIndex) < len(symbols) {
				ids = append(ids, symbols[ssHash.Name.Ref.InnerIndex].OriginalName)
			}
		}
	}
	return ids
}

// isHostSelector checks whether the selector contains a :host or
// :host-context pseudo-class.
//
// Takes selector (css_ast.CompoundSelector) which is the selector to check.
//
// Returns bool which is true if the selector contains a host pseudo-class.
func isHostSelector(selector css_ast.CompoundSelector) bool {
	for _, sub := range selector.SubclassSelectors {
		if pseudo, ok := sub.Data.(*css_ast.SSPseudoClass); ok {
			if pseudo.Name == "host" || pseudo.Name == "host-context" {
				return true
			}
		}
	}
	return false
}
