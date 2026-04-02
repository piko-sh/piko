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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_printer"
	"piko.sh/piko/internal/esbuild/logger"
)

// applyRuleSet applies both inlineable and leftover rules from the given rule
// set. This method can be tested by passing a manually created RuleSet.
//
// Takes ruleSet (*RuleSet) which contains the sorted rules to apply.
// Takes cssAST (css_ast.AST) which provides context for reinserting leftover
// rules.
func (p *Premailer) applyRuleSet(ruleSet *RuleSet, cssAST css_ast.AST) {
	p.applyInlineableRules(ruleSet.InlineableRules)
	p.reinsertLeftoverRules(ruleSet.LeftoverRules, cssAST)
}

// applyInlineableRules finds nodes using the AST query engine and merges
// styles into them. Uses cached original inline styles to avoid walking the
// tree again.
//
// Takes rules ([]styleRule) which contains the CSS rules to apply inline.
func (p *Premailer) applyInlineableRules(rules []styleRule) {
	originalInlineStyles := p.originalInlineStyles
	if originalInlineStyles == nil {
		originalInlineStyles = make(map[*ast_domain.TemplateNode]map[string]bool)
	}

	matched, diagnostics := matchRulesToNodes(p.tree, rules)
	p.tree.Diagnostics = append(p.tree.Diagnostics, diagnostics...)

	for node, nodeProps := range matched {
		originalProps := originalInlineStyles[node]
		applyMatchedPropertiesToNode(node, nodeProps, originalProps, p.options)
	}
}

// matchRulesToNodes matches CSS rules to template nodes and returns the
// resolved property maps per node without modifying the AST. Rules are
// expected to be sorted by specificity (lowest first) so that later rules
// override earlier ones.
//
// Takes tree (*ast_domain.TemplateAST) which provides the DOM to query.
// Takes rules ([]styleRule) which contains the sorted CSS rules.
//
// Returns map[*ast_domain.TemplateNode]map[string]property which maps each
// matched node to its cascade-resolved properties.
// Returns []*ast_domain.Diagnostic which contains any selector parse errors.
func matchRulesToNodes(
	tree *ast_domain.TemplateAST,
	rules []styleRule,
) (map[*ast_domain.TemplateNode]map[string]property, []*ast_domain.Diagnostic) {
	result := make(map[*ast_domain.TemplateNode]map[string]property)
	var diagnostics []*ast_domain.Diagnostic

	for _, rule := range rules {
		matchingNodes, diags := ast_domain.QueryAll(tree, rule.selector, "premailer.go")
		if len(diags) > 0 {
			diagnostics = append(diagnostics, diags...)
			continue
		}

		for _, node := range matchingNodes {
			mergeRuleIntoNodeProps(result, node, rule.properties)
		}
	}

	return result, diagnostics
}

// mergeRuleIntoNodeProps merges a rule's properties into the per-node
// property map, creating the node entry if it does not exist.
//
// Takes result (map) which is the per-node property accumulator.
// Takes node (*ast_domain.TemplateNode) which is the matched node.
// Takes ruleProps (map[string]property) which contains the rule's properties.
func mergeRuleIntoNodeProps(
	result map[*ast_domain.TemplateNode]map[string]property,
	node *ast_domain.TemplateNode,
	ruleProps map[string]property,
) {
	nodeProps := result[node]
	if nodeProps == nil {
		nodeProps = make(map[string]property)
		result[node] = nodeProps
	}
	for propName, propVal := range ruleProps {
		var existingProp *property
		if existing, exists := nodeProps[propName]; exists {
			existingProp = &existing
		}
		if shouldApplyProperty(propName, propVal, existingProp, nil) {
			nodeProps[propName] = propVal
		}
	}
}

// matchPseudoRulesToNodes matches pseudo-element CSS rules to template nodes
// and returns the resolved property maps keyed by node and pseudo-element name.
//
// Takes tree (*ast_domain.TemplateAST) which provides the DOM to query.
// Takes rules ([]pseudoElementRule) which contains the sorted pseudo-element
// rules.
//
// Returns map[*ast_domain.TemplateNode]map[string]map[string]property which maps
// each node to its pseudo-element property values keyed by name then property.
// Returns []*ast_domain.Diagnostic which contains any selector parse errors.
func matchPseudoRulesToNodes(
	tree *ast_domain.TemplateAST,
	rules []pseudoElementRule,
) (map[*ast_domain.TemplateNode]map[string]map[string]property, []*ast_domain.Diagnostic) {
	result := make(map[*ast_domain.TemplateNode]map[string]map[string]property)
	var diagnostics []*ast_domain.Diagnostic

	for _, rule := range rules {
		matchingNodes, diags := ast_domain.QueryAll(tree, rule.selector, "premailer.go")
		if len(diags) > 0 {
			diagnostics = append(diagnostics, diags...)
			continue
		}

		for _, node := range matchingNodes {
			mergePseudoRuleIntoNodeProps(result, node, rule.pseudoElement, rule.properties)
		}
	}

	return result, diagnostics
}

// mergePseudoRuleIntoNodeProps merges a pseudo-element rule's properties
// into the per-node pseudo-element property map, creating entries as needed.
//
// Takes result (map) which is the per-node pseudo-element accumulator.
// Takes node (*ast_domain.TemplateNode) which is the matched node.
// Takes pseudoElement (string) which is the pseudo-element name.
// Takes ruleProps (map[string]property) which contains the rule's properties.
func mergePseudoRuleIntoNodeProps(
	result map[*ast_domain.TemplateNode]map[string]map[string]property,
	node *ast_domain.TemplateNode,
	pseudoElement string,
	ruleProps map[string]property,
) {
	pseudoMap := result[node]
	if pseudoMap == nil {
		pseudoMap = make(map[string]map[string]property)
		result[node] = pseudoMap
	}
	props := pseudoMap[pseudoElement]
	if props == nil {
		props = make(map[string]property)
		pseudoMap[pseudoElement] = props
	}
	for propName, propVal := range ruleProps {
		var existingProp *property
		if existing, exists := props[propName]; exists {
			existingProp = &existing
		}
		if shouldApplyProperty(propName, propVal, existingProp, nil) {
			props[propName] = propVal
		}
	}
}

// applyMatchedPropertiesToNode writes matched CSS properties to a node's style
// attribute, respecting original inline style priorities.
//
// Takes node (*ast_domain.TemplateNode) which is the element to modify.
// Takes nodeProps (map[string]property) which contains the cascade-resolved
// properties.
// Takes originalProps (map[string]bool) which tracks properties from the
// original inline style.
// Takes options (*Options) which controls attribute mapping behaviour.
func applyMatchedPropertiesToNode(
	node *ast_domain.TemplateNode,
	nodeProps map[string]property,
	originalProps map[string]bool,
	options *Options,
) {
	currentStyle, _ := node.GetAttribute(literalStyle)
	styleMap := parseStyleAttribute(currentStyle)

	for propName, propVal := range nodeProps {
		var existingProp *property
		if existing, exists := styleMap[propName]; exists {
			existingProp = &existing
		}

		if shouldApplyProperty(propName, propVal, existingProp, originalProps) {
			styleMap[propName] = propVal
		}
	}

	node.SetAttribute(literalStyle, reconstructStyleAttribute(styleMap))

	if !options.SkipHTMLAttributeMapping {
		ApplyAttributesFromStyle(node, styleMap)
	}
}

// reinsertLeftoverRules creates a new style tag for rules that cannot be
// inlined.
//
// Takes rules ([]css_ast.Rule) which contains CSS rules that could not be
// added to element style attributes.
// Takes cssAST (css_ast.AST) which provides the parsed CSS structure for
// converting rules back into text.
func (p *Premailer) reinsertLeftoverRules(rules []css_ast.Rule, cssAST css_ast.AST) {
	if len(rules) == 0 {
		return
	}

	if p.options.MakeLeftoverImportant {
		makeLeftoverRulesImportant(rules)
	}

	cssString := rulesToCSSString(rules, cssAST)
	if cssString == "" {
		return
	}

	p.insertStyleTag(cssString)
}

// insertStyleTag inserts a style tag with the given CSS into the head element.
//
// Takes cssString (string) which contains the CSS rules to insert.
func (p *Premailer) insertStyleTag(cssString string) {
	head := p.findOrCreateHead()

	styleNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  literalStyle,
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: literalNewline + cssString + literalNewline},
		},
	}
	head.Children = append(head.Children, styleNode)
}

// findOrCreateHead finds the head element or creates one if it does not exist.
//
// Returns *ast_domain.TemplateNode which is the existing or newly created head
// element.
func (p *Premailer) findOrCreateHead() *ast_domain.TemplateNode {
	head := p.tree.Find(func(node *ast_domain.TemplateNode) bool {
		return node.NodeType == ast_domain.NodeElement && node.TagName == "head"
	})

	if head != nil {
		return head
	}

	head = &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "head"}

	html := p.tree.Find(func(node *ast_domain.TemplateNode) bool {
		return node.NodeType == ast_domain.NodeElement && node.TagName == "html"
	})

	if html != nil {
		html.Children = append([]*ast_domain.TemplateNode{head}, html.Children...)
	} else {
		p.tree.RootNodes = append([]*ast_domain.TemplateNode{head}, p.tree.RootNodes...)
	}

	return head
}

// propagateDiagnostics converts esbuild log messages into TemplateAST
// diagnostics.
func (p *Premailer) propagateDiagnostics() {
	if !p.log.HasErrors() {
		return
	}

	sourcePath := "inline <style>"
	if p.tree.SourcePath != nil {
		sourcePath = *p.tree.SourcePath
	}

	for _, message := range p.log.Done() {
		var severity ast_domain.Severity
		switch message.Kind {
		case logger.Error:
			severity = ast_domain.Error
		case logger.Warning:
			severity = ast_domain.Warning
		default:
			continue
		}

		var lineText string
		var location ast_domain.Location

		if message.Data.Location != nil {
			lineText = message.Data.Location.LineText
			location = ast_domain.Location{
				Line:   message.Data.Location.Line,
				Column: message.Data.Location.Column + 1,
			}
		}

		p.tree.Diagnostics = append(p.tree.Diagnostics, ast_domain.NewDiagnostic(
			severity,
			message.Data.Text,
			lineText,
			location,
			sourcePath,
		))
	}
}

// htmlTagDiagnosticInfo holds information for creating diagnostics about
// problematic HTML tags.
type htmlTagDiagnosticInfo struct {
	// message is the text shown to the user when this diagnostic is reported.
	message string

	// severity specifies the diagnostic severity level for this HTML tag.
	severity ast_domain.Severity
}

// problematicHTMLTags maps tag names to their diagnostic information.
var problematicHTMLTags = map[string]htmlTagDiagnosticInfo{
	"script": {
		severity: ast_domain.Warning,
		message:  "The <script> element is a security risk and is universally stripped by email clients. All JavaScript will be removed.",
	},
	"form": {
		severity: ast_domain.Warning,
		message:  "The <form> element is not supported. HTML forms do not work in email clients. Link to a form on a web page instead.",
	},
	"input": {
		severity: ast_domain.Warning,
		message:  "The <input> element is not supported. HTML forms do not work in email clients. Link to a form on a web page instead.",
	},
	"textarea": {
		severity: ast_domain.Warning,
		message:  "The <textarea> element is not supported. HTML forms do not work in email clients. Link to a form on a web page instead.",
	},
	"select": {
		severity: ast_domain.Warning,
		message:  "The <select> element is not supported. HTML forms do not work in email clients. Link to a form on a web page instead.",
	},
	"option": {
		severity: ast_domain.Warning,
		message:  "The <option> element is not supported. HTML forms do not work in email clients. Link to a form on a web page instead.",
	},
	"button": {
		severity: ast_domain.Warning,
		message:  "The <button> element is not supported. HTML forms do not work in email clients. Link to a form on a web page instead.",
	},
	"iframe": {
		severity: ast_domain.Warning,
		message:  "The <iframe> element is a security risk and is universally stripped by email clients.",
	},
	"object": {
		severity: ast_domain.Warning,
		message:  "The <object> element is a security risk and is universally stripped by email clients.",
	},
	"embed": {
		severity: ast_domain.Warning,
		message:  "The <embed> element is a security risk and is universally stripped by email clients.",
	},
	"applet": {
		severity: ast_domain.Warning,
		message:  "The <applet> element is a security risk and is universally stripped by email clients.",
	},
	"base": {
		severity: ast_domain.Warning,
		message: "The <base> tag is dangerous in HTML emails. It can break all relative links and image paths, " +
			"especially when the email is forwarded or viewed in different clients.",
	},
	"svg": {
		severity: ast_domain.Warning,
		message: "The <svg> element is not supported in major email clients like Gmail and Outlook (Windows). " +
			"It will be stripped. Use a fallback <img> with a PNG or JPG for universal compatibility.",
	},
	"video": {
		severity: ast_domain.Warning,
		message: "The <video> element has very limited support and will not work in most email clients. " +
			"Consider linking to the media file or using an animated GIF as a preview.",
	},
	"audio": {
		severity: ast_domain.Warning,
		message: "The <audio> element has very limited support and will not work in most email clients. " +
			"Consider linking to the media file or using an animated GIF as a preview.",
	},
	"source": {
		severity: ast_domain.Warning,
		message: "The <source> element has very limited support and will not work in most email clients. " +
			"Consider linking to the media file or using an animated GIF as a preview.",
	},
	"canvas": {
		severity: ast_domain.Warning,
		message:  "The interactive <canvas> element is not supported in email clients.",
	},
	"details": {
		severity: ast_domain.Warning,
		message:  "The interactive <details> element is not supported in email clients.",
	},
	"summary": {
		severity: ast_domain.Warning,
		message:  "The interactive <summary> element is not supported in email clients.",
	},
	"center": {
		severity: ast_domain.Info,
		message: "The <center> tag is obsolete. For better compatibility, use the 'align=\"center\"' attribute " +
			"on a containing <table> or <td>, or apply 'text-align: center' via CSS.",
	},
}

// shouldApplyProperty checks if a new CSS property should replace an existing
// one based on CSS priority rules and !important flags.
//
// CSS priority order (highest to lowest):
//  1. Inline styles with !important.
//  2. CSS rules with !important (by specificity).
//  3. Original inline styles (before CSS processing).
//  4. CSS rules (by specificity).
//
// Takes propName (string) which is the name of the CSS property.
// Takes newProp (property) which is the new value to consider.
// Takes existingProp (*property) which is the current value, or nil if the
// property does not exist.
// Takes originalProps (map[string]bool) which tracks properties that were in
// the original inline style.
//
// Returns bool which is true if the new property should replace the existing
// one.
func shouldApplyProperty(
	propName string,
	newProp property,
	existingProp *property,
	originalProps map[string]bool,
) bool {
	if existingProp != nil && existingProp.important && !newProp.important {
		return false
	}

	if originalProps != nil && originalProps[propName] && !newProp.important {
		return false
	}

	return true
}

// makeLeftoverRulesImportant sets the Important flag on all CSS declarations
// within the given rules. This means email clients like Gmail respect styles
// in <style> tags, as many only apply styles marked !important.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to process.
func makeLeftoverRulesImportant(rules []css_ast.Rule) {
	for i := range rules {
		rule := &rules[i]

		switch r := rule.Data.(type) {
		case *css_ast.RSelector:
			for j := range r.Rules {
				if declaration, ok := r.Rules[j].Data.(*css_ast.RDeclaration); ok {
					declaration.Important = true
				}
			}

		case *css_ast.RAtMedia:
			makeLeftoverRulesImportant(r.Rules)

		case *css_ast.RAtLayer:
			makeLeftoverRulesImportant(r.Rules)
		}
	}
}

// rulesToCSSString converts CSS rules to a string representation.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to convert.
// Takes cssAST (css_ast.AST) which provides the symbol table for the CSS.
//
// Returns string which is the CSS output with colour values converted to hex
// format for email client support.
func rulesToCSSString(rules []css_ast.Rule, cssAST css_ast.AST) string {
	leftoverAST := css_ast.AST{Rules: rules, Symbols: cssAST.Symbols}
	symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{cssAST.Symbols}}

	options := css_printer.Options{MinifyWhitespace: false}
	result := css_printer.Print(leftoverAST, symbolMap, options)
	cssString := strings.TrimSpace(string(result.CSS))

	return convertColorValues(cssString)
}

// createHTMLTagDiagnostic creates a diagnostic for an HTML tag that may cause
// problems in email clients, or returns nil if the tag is safe.
//
// Takes node (*ast_domain.TemplateNode) which provides the HTML tag to check.
// Takes sourcePath (string) which identifies the source file for reporting.
//
// Returns *ast_domain.Diagnostic which describes the issue, or nil if the tag
// is safe.
func createHTMLTagDiagnostic(node *ast_domain.TemplateNode, sourcePath string) *ast_domain.Diagnostic {
	if info, found := problematicHTMLTags[node.TagName]; found {
		return ast_domain.NewDiagnostic(
			info.severity,
			info.message,
			"<"+node.TagName+">",
			node.Location,
			sourcePath,
		)
	}

	return nil
}
