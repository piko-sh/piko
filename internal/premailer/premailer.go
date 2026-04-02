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

package premailer

import (
	"html"
	"net/url"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/logger"
)

// Premailer applies CSS styles inline to a TemplateAST.
// It bridges the TemplateAST and the CSS processing logic.
type Premailer struct {
	// tree holds the parsed template AST for walking and modifying nodes.
	tree *ast_domain.TemplateAST

	// options holds the settings for HTML processing.
	options *Options

	// log stores CSS parsing errors and warnings to report later.
	log logger.Log

	// originalInlineStyles caches the original inline styles before CSS
	// application.
	originalInlineStyles map[*ast_domain.TemplateNode]map[string]bool
}

// Transform processes an email template by inlining CSS and preparing it for
// email clients. It extracts stylesheets, removes comments and scripts, applies
// CSS rules to the AST, validates HTML tags, and appends link query parameters.
//
// Returns *ast_domain.TemplateAST which contains the transformed template.
// Returns error when processing fails.
func (p *Premailer) Transform() (*ast_domain.TemplateAST, error) {
	collectionData := p.collectAndValidate()

	if len(collectionData.nodesToRemove) > 0 {
		p.removeNodes(collectionData.nodesToRemove)
		p.tree.InvalidateQueryContext()
	}

	if collectionData.cssString != "" {
		p.processCSS(collectionData.cssString)
	}

	p.tree.Diagnostics = append(p.tree.Diagnostics, collectionData.diagnostics...)

	p.performCleanup(collectionData.anchorTargets)

	p.tree.InvalidateQueryContext()

	return p.tree, nil
}

// ResolvedProperties contains CSS properties matched to nodes without
// modifying the template AST. Used by the layouter for CSS resolution.
type ResolvedProperties struct {
	// Elements maps each template node to its cascade-resolved CSS properties.
	Elements map[*ast_domain.TemplateNode]map[string]string

	// PseudoElements maps each template node to its pseudo-element properties,
	// keyed by pseudo-element name ("before" or "after").
	PseudoElements map[*ast_domain.TemplateNode]map[string]map[string]string

	// Diagnostics contains any warnings or errors found during processing.
	Diagnostics []*ast_domain.Diagnostic
}

// ResolveProperties resolves the CSS cascade for each node and returns
// property maps without modifying the template AST. This is the non-mutating
// counterpart to Transform(), designed for layout engines that need CSS
// resolution without email-specific transformations.
//
// The method parses CSS from the ExternalCSS option, resolves variables,
// matches selectors, and merges with inline style attributes. It does not
// remove <style> tags, write styles back to the AST, or apply HTML attribute
// mappings.
//
// Returns *ResolvedProperties which contains per-node property maps.
// Returns error when CSS parsing fails.
func (p *Premailer) ResolveProperties() (*ResolvedProperties, error) {
	inlineStyles := p.collectInlineStyles()

	result := &ResolvedProperties{
		Elements:       make(map[*ast_domain.TemplateNode]map[string]string),
		PseudoElements: make(map[*ast_domain.TemplateNode]map[string]map[string]string),
	}

	cssString := p.options.ExternalCSS
	if cssString == "" && len(inlineStyles) == 0 {
		return result, nil
	}

	ruleSet := p.parseRuleSet(cssString, result)

	matched, matchDiags := matchRulesToNodes(p.tree, ruleSet.InlineableRules)
	result.Diagnostics = append(result.Diagnostics, matchDiags...)

	mergeMatchedWithInline(p.tree, matched, inlineStyles, result)
	p.resolvePseudoElementProperties(ruleSet, result)

	return result, nil
}

// collectInlineStyles walks the tree and collects existing inline style
// attributes from element nodes, parsing !important flags. When
// ExpandShorthands is enabled, shorthand properties are expanded to longhands.
//
// Returns map[*ast_domain.TemplateNode]map[string]property which maps nodes
// to their parsed inline style properties with importance tracking.
func (p *Premailer) collectInlineStyles() map[*ast_domain.TemplateNode]map[string]property {
	inlineStyles := make(map[*ast_domain.TemplateNode]map[string]property)
	p.tree.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}
		styleAttr, ok := node.GetAttribute("style")
		if ok && styleAttr != "" {
			parsed := parseInlineStyleWithImportance(styleAttr)
			if p.options.ExpandShorthands {
				parsed = expandInlineShorthandsWithImportance(parsed)
			}
			inlineStyles[node] = parsed
		}
		return true
	})
	return inlineStyles
}

// parseInlineStyleWithImportance parses an inline style attribute, extracting
// !important flags from each declaration.
//
// Takes styleAttr (string) which is the style attribute value to parse.
//
// Returns map[string]property which maps property names to their values and
// importance flags.
func parseInlineStyleWithImportance(styleAttr string) map[string]property {
	styles := make(map[string]property)
	if styleAttr == "" {
		return styles
	}

	if strings.Contains(styleAttr, "&") {
		styleAttr = html.UnescapeString(styleAttr)
	}
	for declaration := range strings.SplitSeq(styleAttr, ";") {
		parts := strings.SplitN(declaration, ":", 2)
		if len(parts) != 2 {
			continue
		}
		propName := strings.TrimSpace(parts[0])
		propValue := strings.TrimSpace(parts[1])
		if propName == "" || propValue == "" {
			continue
		}
		important := false
		if idx := strings.Index(propValue, "!important"); idx != -1 {
			important = true
			propValue = strings.TrimSpace(propValue[:idx])
		}
		if propValue != "" {
			styles[propName] = property{value: propValue, important: important}
		}
	}
	return styles
}

// expandInlineShorthandsWithImportance expands shorthand CSS properties in an
// inline style map to their longhand equivalents, preserving importance flags.
//
// Takes styles (map[string]property) which contains the parsed inline styles.
//
// Returns map[string]property with shorthands replaced by their longhands.
func expandInlineShorthandsWithImportance(styles map[string]property) map[string]property {
	expanded := make(map[string]property, len(styles))
	for prop, p := range styles {
		longhands := expandShorthand(prop, p.value)
		if longhands == nil && isCSSWideKeyword(p.value) {
			longhands = expandCSSWideKeyword(prop, p.value)
		}
		if longhands != nil {
			for lh, lv := range longhands {
				expanded[lh] = property{value: lv, important: p.important}
			}
		} else {
			expanded[prop] = p
		}
	}
	return expanded
}

// parseRuleSet parses CSS and produces a RuleSet, appending diagnostics to
// the result.
//
// Takes cssString (string) which is the CSS to parse; empty returns an
// empty RuleSet.
// Takes result (*ResolvedProperties) which receives diagnostics.
//
// Returns *RuleSet which contains the processed rules.
func (p *Premailer) parseRuleSet(cssString string, result *ResolvedProperties) *RuleSet {
	if cssString == "" {
		return &RuleSet{}
	}

	source := logger.Source{Contents: cssString}
	cssAST := css_parser.Parse(p.log, source, css_parser.Options{})
	p.propagateDiagnostics()

	sourcePath := determineSourcePath(p.tree.SourcePath)
	ruleSet := ProcessCSS(cssAST, p.options, &result.Diagnostics, sourcePath)

	if !p.options.SkipEmailValidation {
		validationDiags := validateEmailCompatibility(ruleSet, sourcePath)
		result.Diagnostics = append(result.Diagnostics, validationDiags...)
	}

	return ruleSet
}

// mergeMatchedWithInline walks the tree and merges rule-matched properties
// with inline styles, respecting cascade priorities. The merged results
// are stored in result.Elements.
//
// Takes tree (*ast_domain.TemplateAST) which is the template to walk.
// Takes matched (map) which contains rule-matched properties per node.
// Takes inlineStyles (map) which contains parsed inline style properties.
// Takes result (*ResolvedProperties) which receives the merged properties.
func mergeMatchedWithInline(
	tree *ast_domain.TemplateAST,
	matched map[*ast_domain.TemplateNode]map[string]property,
	inlineStyles map[*ast_domain.TemplateNode]map[string]property,
	result *ResolvedProperties,
) {
	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}

		ruleProps := matched[node]
		nodeInline := inlineStyles[node]

		if ruleProps == nil && nodeInline == nil {
			return true
		}

		result.Elements[node] = mergeRuleAndInlineProperties(ruleProps, nodeInline)
		return true
	})
}

// mergeRuleAndInlineProperties merges CSS rule properties with inline style
// properties, applying the full CSS cascade priority.
//
// The priority order is:
//   - Inline !important beats stylesheet !important
//   - Stylesheet !important beats inline normal
//   - Inline normal beats stylesheet normal
//
// Takes ruleProps (map[string]property) which contains the rule-matched
// properties with importance flags.
// Takes nodeInline (map[string]property) which contains the inline style
// properties with importance flags.
//
// Returns map[string]string which contains the merged property values.
func mergeRuleAndInlineProperties(
	ruleProps map[string]property,
	nodeInline map[string]property,
) map[string]string {
	merged := make(map[string]string)

	for propName, propVal := range ruleProps {
		merged[propName] = propVal.value
	}

	for propName, propVal := range nodeInline {
		if ruleP, exists := ruleProps[propName]; exists && ruleP.important {
			continue
		}
		merged[propName] = propVal.value
	}

	for propName, propVal := range ruleProps {
		if propVal.important {
			merged[propName] = propVal.value
		}
	}

	for propName, propVal := range nodeInline {
		if propVal.important {
			merged[propName] = propVal.value
		}
	}

	return merged
}

// resolvePseudoElementProperties matches pseudo-element rules and converts
// the results into string property maps, storing them in result.PseudoElements.
//
// Takes ruleSet (*RuleSet) which contains the pseudo-element rules.
// Takes result (*ResolvedProperties) which receives the resolved properties.
func (p *Premailer) resolvePseudoElementProperties(ruleSet *RuleSet, result *ResolvedProperties) {
	if !p.options.ResolvePseudoElements || len(ruleSet.PseudoElementRules) == 0 {
		return
	}

	pseudoMatched, pseudoDiags := matchPseudoRulesToNodes(p.tree, ruleSet.PseudoElementRules)
	result.Diagnostics = append(result.Diagnostics, pseudoDiags...)

	for node, pseudoMap := range pseudoMatched {
		result.PseudoElements[node] = convertPseudoProperties(pseudoMap)
	}
}

// convertPseudoProperties converts a pseudo-element property map with
// importance flags into a plain string property map.
//
// Takes pseudoMap (map[string]map[string]property) which maps pseudo-element
// names to their property maps.
//
// Returns map[string]map[string]string which contains the converted
// properties.
func convertPseudoProperties(
	pseudoMap map[string]map[string]property,
) map[string]map[string]string {
	converted := make(map[string]map[string]string, len(pseudoMap))
	for pseudoName, props := range pseudoMap {
		strProps := make(map[string]string, len(props))
		for propName, propVal := range props {
			strProps[propName] = propVal.value
		}
		converted[pseudoName] = strProps
	}
	return converted
}

// processCSS parses, checks, and applies CSS to the template AST.
//
// Takes cssString (string) which contains the raw CSS to process.
func (p *Premailer) processCSS(cssString string) {
	source := logger.Source{Contents: cssString}
	cssAST := css_parser.Parse(p.log, source, css_parser.Options{})

	p.propagateDiagnostics()

	sourcePath := determineSourcePath(p.tree.SourcePath)

	ruleSet := ProcessCSS(cssAST, p.options, &p.tree.Diagnostics, sourcePath)

	if !p.options.SkipEmailValidation {
		validationDiags := validateEmailCompatibility(ruleSet, sourcePath)
		p.tree.Diagnostics = append(p.tree.Diagnostics, validationDiags...)
	}

	p.applyRuleSet(ruleSet, cssAST)

	p.removeAttributesIfConfigured()
}

// collectionData holds data gathered during the first tree walk.
type collectionData struct {
	// cssString holds the combined CSS text to apply; empty means no CSS rules.
	cssString string

	// nodesToRemove holds nodes to remove after processing, such as style tags,
	// comments, and scripts.
	nodesToRemove []*ast_domain.TemplateNode

	// originalInlineStyles maps nodes to their original inline style properties.
	originalInlineStyles map[*ast_domain.TemplateNode]map[string]bool

	// anchorTargets tracks element IDs that are targets of internal anchor links.
	anchorTargets map[string]bool

	// diagnostics collects validation warnings found during the collection phase.
	diagnostics []*ast_domain.Diagnostic
}

// collectAndValidate performs a unified tree walk that collects CSS, nodes to
// remove, inline styles, anchor targets, and validates HTML structure. This
// consolidates what were previously 6+ separate tree walks into a single pass.
//
// Returns *collectionData which contains all collected information.
func (p *Premailer) collectAndValidate() *collectionData {
	data := p.initializeCollectionData()
	cssBuilder := p.initializeCSSBuilder(data)
	sourcePath := determineSourcePath(p.tree.SourcePath)

	p.tree.Walk(p.createCollectionWalker(data, &cssBuilder, sourcePath))

	if cssBuilder.Len() > 0 {
		data.cssString = cssBuilder.String()
	}

	p.originalInlineStyles = data.originalInlineStyles
	return data
}

// initializeCollectionData creates and initialises the collection data
// structure.
//
// Returns *collectionData which holds the state for CSS inlining operations.
func (*Premailer) initializeCollectionData() *collectionData {
	return &collectionData{
		cssString:            "",
		nodesToRemove:        make([]*ast_domain.TemplateNode, 0),
		originalInlineStyles: make(map[*ast_domain.TemplateNode]map[string]bool),
		anchorTargets:        make(map[string]bool),
		diagnostics:          make([]*ast_domain.Diagnostic, 0),
	}
}

// initializeCSSBuilder creates a CSS builder with external CSS if provided.
//
// Takes data (*collectionData) which holds the collected CSS string to append.
//
// Returns strings.Builder which contains the combined external and collected
// CSS.
func (p *Premailer) initializeCSSBuilder(data *collectionData) strings.Builder {
	var cssBuilder strings.Builder

	if p.options.ExternalCSS != "" {
		cssBuilder.WriteString(p.options.ExternalCSS)
		cssBuilder.WriteString(literalNewline)
		data.cssString = cssBuilder.String()
	}

	if data.cssString != "" {
		cssBuilder.WriteString(data.cssString)
	}

	return cssBuilder
}

// createCollectionWalker returns a walker function that processes each node.
//
// Takes data (*collectionData) which stores collected styles and nodes.
// Takes cssBuilder (*strings.Builder) which accumulates extracted CSS.
// Takes sourcePath (string) which identifies the template being processed.
//
// Returns func(*ast_domain.TemplateNode) bool which visits each node,
// collecting styles and validation data, and always returns true to continue.
func (p *Premailer) createCollectionWalker(data *collectionData, cssBuilder *strings.Builder, sourcePath string) func(*ast_domain.TemplateNode) bool {
	return func(node *ast_domain.TemplateNode) bool {
		if !p.options.SkipStyleExtraction {
			p.processStyleNode(node, cssBuilder, data)
			p.collectNodesToRemove(node, data)
		}
		p.captureInlineStyles(node, data)
		if !p.options.SkipEmailValidation {
			p.validateHTMLTag(node, data, sourcePath)
		}
		p.captureAnchorTargets(node, data)
		return true
	}
}

// processStyleNode collects CSS from style nodes and marks them for removal.
//
// Takes node (*ast_domain.TemplateNode) which is the style element to process.
// Takes cssBuilder (*strings.Builder) which accumulates the extracted CSS.
// Takes data (*collectionData) which tracks nodes to remove.
func (*Premailer) processStyleNode(node *ast_domain.TemplateNode, cssBuilder *strings.Builder, data *collectionData) {
	if node.NodeType != ast_domain.NodeElement || node.TagName != "style" {
		return
	}

	if shouldIgnoreStyleNode(node) || shouldSkipStyleNodeMedia(node) {
		return
	}

	content := node.InnerHTML
	if content == "" && len(node.Children) > 0 {
		for _, child := range node.Children {
			if child.NodeType == ast_domain.NodeText {
				content += child.TextContent
			}
		}
	}

	if content != "" {
		cssBuilder.WriteString(content)
		cssBuilder.WriteString(literalNewline)
	}

	data.nodesToRemove = append(data.nodesToRemove, node)
}

// collectNodesToRemove marks comments and scripts for removal.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check for removal.
// Takes data (*collectionData) which accumulates nodes marked for removal.
func (*Premailer) collectNodesToRemove(node *ast_domain.TemplateNode, data *collectionData) {
	if node.NodeType == ast_domain.NodeComment {
		data.nodesToRemove = append(data.nodesToRemove, node)
		return
	}

	if node.NodeType == ast_domain.NodeElement && node.TagName == "script" {
		data.nodesToRemove = append(data.nodesToRemove, node)
	}
}

// captureInlineStyles stores original inline style properties for each element.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes data (*collectionData) which holds the collection of original styles.
func (*Premailer) captureInlineStyles(node *ast_domain.TemplateNode, data *collectionData) {
	if node.NodeType != ast_domain.NodeElement {
		return
	}

	styleAttr, ok := node.GetAttribute("style")
	if !ok || styleAttr == "" {
		return
	}

	parsed := parseInlineStyle(styleAttr)
	originalProps := make(map[string]bool, len(parsed))
	for prop := range parsed {
		originalProps[prop] = true
	}
	data.originalInlineStyles[node] = originalProps
}

// validateHTMLTag validates element tags and collects diagnostics.
//
// Takes node (*ast_domain.TemplateNode) which is the node to validate.
// Takes data (*collectionData) which collects the resulting diagnostics.
// Takes sourcePath (string) which identifies the source file for error
// messages.
func (*Premailer) validateHTMLTag(node *ast_domain.TemplateNode, data *collectionData, sourcePath string) {
	if node.NodeType != ast_domain.NodeElement {
		return
	}

	if diagnostic := createHTMLTagDiagnostic(node, sourcePath); diagnostic != nil {
		data.diagnostics = append(data.diagnostics, diagnostic)
	}
}

// captureAnchorTargets collects IDs referenced by anchor href attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the node to inspect.
// Takes data (*collectionData) which stores the collected anchor targets.
func (p *Premailer) captureAnchorTargets(node *ast_domain.TemplateNode, data *collectionData) {
	if !p.options.RemoveIDs {
		return
	}

	if node.NodeType != ast_domain.NodeElement || node.TagName != "a" {
		return
	}

	href, ok := node.GetAttribute("href")
	if !ok || !strings.HasPrefix(href, "#") {
		return
	}

	targetID := strings.TrimPrefix(href, "#")
	if targetID != "" {
		data.anchorTargets[targetID] = true
	}
}

// performCleanup runs the final cleanup tasks in a single pass.
// This combines what were previously three or more separate tree walks.
//
// Takes anchorTargets (map[string]bool) which contains IDs that anchors
// point to.
func (p *Premailer) performCleanup(anchorTargets map[string]bool) {
	if !p.shouldPerformCleanup() {
		return
	}

	p.tree.Walk(p.createCleanupWalker(anchorTargets))
}

// shouldPerformCleanup checks if any cleanup operations are configured.
//
// Returns bool which is true when class removal, ID removal, or link query
// parameters are configured.
func (p *Premailer) shouldPerformCleanup() bool {
	return p.options.RemoveClasses || p.options.RemoveIDs || len(p.options.LinkQueryParams) > 0
}

// createCleanupWalker returns a walker function that performs cleanup on each
// element.
//
// Takes anchorTargets (map[string]bool) which identifies anchor IDs to
// preserve.
//
// Returns func(*ast_domain.TemplateNode) bool which walks nodes and applies
// cleanup operations.
func (p *Premailer) createCleanupWalker(anchorTargets map[string]bool) func(*ast_domain.TemplateNode) bool {
	return func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}

		p.removeClassIfConfigured(node)
		p.removeIDIfConfigured(node, anchorTargets)
		p.appendLinkParamsIfConfigured(node)

		return true
	}
}

// removeClassIfConfigured removes the class attribute if configured.
//
// Takes node (*ast_domain.TemplateNode) which is the node to modify.
func (p *Premailer) removeClassIfConfigured(node *ast_domain.TemplateNode) {
	if p.options.RemoveClasses {
		node.RemoveAttribute("class")
	}
}

// removeIDIfConfigured removes the ID attribute from a node if configured,
// unless the ID is an anchor target.
//
// Takes node (*ast_domain.TemplateNode) which is the node to process.
// Takes anchorTargets (map[string]bool) which contains IDs that are anchor
// targets and should be preserved.
func (p *Premailer) removeIDIfConfigured(node *ast_domain.TemplateNode, anchorTargets map[string]bool) {
	if !p.options.RemoveIDs {
		return
	}

	id, ok := node.GetAttribute("id")
	if !ok {
		return
	}

	if !anchorTargets[id] {
		node.RemoveAttribute("id")
	}
}

// appendLinkParamsIfConfigured appends query parameters to anchor links if
// configured.
//
// Takes node (*ast_domain.TemplateNode) which is the HTML element to process.
func (p *Premailer) appendLinkParamsIfConfigured(node *ast_domain.TemplateNode) {
	if len(p.options.LinkQueryParams) == 0 || node.TagName != "a" {
		return
	}

	href, ok := node.GetAttribute("href")
	if !ok || shouldSkipLink(href) {
		return
	}

	parsedURL, err := url.Parse(href)
	if err != nil {
		return
	}

	query := parsedURL.Query()
	for key, value := range p.options.LinkQueryParams {
		query.Set(key, value)
	}
	parsedURL.RawQuery = query.Encode()
	node.SetAttribute("href", parsedURL.String())
}

// New creates a Premailer for a given template AST with optional
// configuration.
//
// Use functional options to configure behaviour:
// premailer.New(tree, premailer.WithKeepBangImportant(true))
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template
// to process.
// Takes opts (...Option) which are functional options to configure
// behaviour.
//
// Returns *Premailer which is ready to inline CSS styles.
func New(tree *ast_domain.TemplateAST, opts ...Option) *Premailer {
	return &Premailer{
		tree:    tree,
		log:     logger.NewDeferLog(logger.DeferLogAll, nil),
		options: applyOptions(opts...),
	}
}

// parseInlineStyle parses an inline style attribute into a map of properties.
//
// Takes styleAttr (string) which is the style attribute value to parse.
//
// Returns map[string]string which maps property names to their values.
func parseInlineStyle(styleAttr string) map[string]string {
	styles := make(map[string]string)
	if styleAttr == "" {
		return styles
	}
	for declaration := range strings.SplitSeq(styleAttr, ";") {
		parts := strings.SplitN(declaration, ":", 2)
		if len(parts) == 2 {
			property := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if property != "" && value != "" {
				styles[property] = value
			}
		}
	}
	return styles
}

// determineSourcePath returns the source path for diagnostic reporting.
//
// Takes sourcePath (*string) which points to a custom path, or nil to use the
// default.
//
// Returns string which is the source path. Defaults to "inline-styles" when
// sourcePath is nil.
func determineSourcePath(sourcePath *string) string {
	if sourcePath != nil {
		return *sourcePath
	}
	return "inline-styles"
}

// shouldIgnoreStyleNode checks if a style node has the data-premailer="ignore"
// attribute set.
//
// Takes node (*ast_domain.TemplateNode) which is the style node to check.
//
// Returns bool which is true if the node should be skipped during processing.
func shouldIgnoreStyleNode(node *ast_domain.TemplateNode) bool {
	value, exists := node.GetAttribute("data-premailer")
	return exists && value == "ignore"
}

// shouldSkipStyleNodeMedia checks if a style node should be skipped based on
// its media attribute.
//
// Only inline styles for "all", "screen", or no media attribute are processed.
// Style nodes with other media types (such as "print") are skipped.
//
// Takes node (*ast_domain.TemplateNode) which is the style node to check.
//
// Returns bool which is true if the node should be skipped.
func shouldSkipStyleNodeMedia(node *ast_domain.TemplateNode) bool {
	media, exists := node.GetAttribute("media")
	return exists && media != "all" && media != "screen"
}
