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
	"slices"
	"strings"

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_parser"
	"piko.sh/piko/internal/esbuild/css_printer"
	"piko.sh/piko/internal/esbuild/logger"
)

// ExtractBodyStyles extracts styles for the body selector from CSS and returns
// them as inline styles plus the remaining CSS with body rules removed.
//
// Body styles are applied inline even when the body element does not exist
// during premailer processing. Handles both simple "body { ... }"
// rules and compound selectors like ".foo, body { ... }". Uses esbuild's CSS
// parser for accurate parsing.
//
// Takes css (string) which is the CSS content to process.
//
// Returns inlineStyles (string) which contains the extracted body styles
// formatted for inline use, with !important declarations stripped.
// Returns cleanedCSS (string) which is the original CSS with body rules
// removed.
func ExtractBodyStyles(css string) (inlineStyles string, cleanedCSS string) {
	if css == "" {
		return "", ""
	}

	log := logger.NewDeferLog(logger.DeferLogAll, nil)
	source := logger.Source{Contents: css}
	cssAST := css_parser.Parse(log, source, css_parser.Options{})

	styleMap := make(map[string]property)
	var cleanedRules []css_ast.Rule

	for _, rule := range cssAST.Rules {
		processedRule := processRuleForBodyExtraction(rule, &styleMap, cssAST.Symbols)
		if processedRule != nil {
			cleanedRules = append(cleanedRules, *processedRule)
		}
	}

	inlineStyles = buildInlineStylesFromMap(styleMap)
	cleanedCSS = rebuildCSSFromRules(cleanedRules, cssAST.Symbols)

	return inlineStyles, cleanedCSS
}

// parseStyleAttribute converts a style attribute string into a map for easy
// merging.
//
// Takes style (string) which contains CSS declarations separated by semicolons.
//
// Returns map[string]property which maps property names to their values and
// importance flags.
func parseStyleAttribute(style string) map[string]property {
	props := make(map[string]property)
	for declaration := range strings.SplitSeq(style, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		parts := strings.SplitN(declaration, ":", 2)
		if len(parts) != 2 {
			continue
		}
		propName := strings.TrimSpace(parts[0])
		propVal := strings.TrimSpace(parts[1])

		if propName == "" || propVal == "" {
			continue
		}

		isImportant := strings.HasSuffix(strings.ToLower(propVal), "!important")
		if isImportant {
			propVal = strings.TrimSpace(propVal[:len(propVal)-len("!important")])
		}

		propVal = convertColorValues(propVal)

		props[propName] = property{value: propVal, important: isImportant}
	}
	return props
}

// reconstructStyleAttribute converts a style map back into a CSS string for use
// in an HTML style attribute.
//
// The !important flag is always stripped from inline styles because inline
// styles already have the highest specificity, making it unnecessary. Some
// email clients may not handle !important in inline styles correctly. If you
// need !important for webmail overrides, use the KeepBangImportant option which
// keeps the rule in the style block.
//
// Takes styleMap (map[string]property) which contains the CSS properties to
// rebuild.
//
// Returns string which is the formatted CSS string with properties sorted
// alphabetically, or an empty string if the map is empty.
func reconstructStyleAttribute(styleMap map[string]property) string {
	if len(styleMap) == 0 {
		return ""
	}

	styles := make([]string, 0, len(styleMap))
	for name, prop := range styleMap {
		formatted := name + ": " + prop.value
		styles = append(styles, formatted)
	}
	slices.Sort(styles)
	return strings.Join(styles, "; ")
}

// processRuleForBodyExtraction processes a single CSS rule by extracting body
// styles and returning a modified rule with body selectors removed.
//
// Takes rule (css_ast.Rule) which is the CSS rule to process.
// Takes styleMap (*map[string]property) which collects the extracted body
// styles.
// Takes symbols ([]ast.Symbol) which provides symbol context for extraction.
//
// Returns *css_ast.Rule which is the modified rule without body selectors,
// or nil if the rule should be removed.
func processRuleForBodyExtraction(rule css_ast.Rule, styleMap *map[string]property, symbols []ast.Symbol) *css_ast.Rule {
	r, ok := rule.Data.(*css_ast.RSelector)
	if !ok {
		return &rule
	}

	hasBodySelector, nonBodySelectors := separateBodySelectors(r.Selectors)

	if hasBodySelector {
		extractBodyDeclarations(r.Rules, symbols, styleMap)
	}

	if len(nonBodySelectors) > 0 {
		modifiedRule := rule
		modifiedRule.Data = &css_ast.RSelector{
			Selectors: nonBodySelectors,
			Rules:     r.Rules,
		}
		return &modifiedRule
	}

	return nil
}

// separateBodySelectors splits CSS selectors into body and non-body groups.
//
// Takes selectors ([]css_ast.ComplexSelector) which contains the CSS selectors
// to check.
//
// Returns hasBody (bool) which is true if any selector targets the body
// element.
// Returns nonBodySelectors ([]css_ast.ComplexSelector) which contains all
// selectors that do not target the body element.
func separateBodySelectors(selectors []css_ast.ComplexSelector) (hasBody bool, nonBodySelectors []css_ast.ComplexSelector) {
	for _, complexSel := range selectors {
		if isSimpleBodySelector(complexSel) {
			hasBody = true
		} else {
			nonBodySelectors = append(nonBodySelectors, complexSel)
		}
	}
	return hasBody, nonBodySelectors
}

// isSimpleBodySelector checks if a complex selector is a simple "body"
// selector.
//
// Takes selector (css_ast.ComplexSelector) which is the selector to check.
//
// Returns bool which is true if the selector is exactly "body" with no
// subclass selectors such as classes, IDs, or pseudo-classes.
func isSimpleBodySelector(selector css_ast.ComplexSelector) bool {
	if len(selector.Selectors) != 1 {
		return false
	}

	compound := selector.Selectors[0]
	if compound.TypeSelector == nil || compound.TypeSelector.Name.Text != "body" {
		return false
	}

	return len(compound.SubclassSelectors) == 0
}

// extractBodyDeclarations gets CSS declarations from rules and adds them to the
// style map.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to process.
// Takes symbols ([]ast.Symbol) which provides symbol lookup for tokens.
// Takes styleMap (*map[string]property) which receives the extracted
// properties.
func extractBodyDeclarations(rules []css_ast.Rule, symbols []ast.Symbol, styleMap *map[string]property) {
	symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{symbols}}

	for _, declRule := range rules {
		declaration, ok := declRule.Data.(*css_ast.RDeclaration)
		if !ok {
			continue
		}

		propName := declaration.KeyText
		propValue := tokensToString(declaration.Value, symbols, symbolMap)
		propValue = convertColorValues(propValue)

		(*styleMap)[propName] = property{
			value:     propValue,
			important: declaration.Important,
		}
	}
}

// buildInlineStylesFromMap builds an inline style string from a property map.
//
// Takes styleMap (map[string]property) which contains the CSS properties to
// convert.
//
// Returns string which is the inline style attribute value. Adds a trailing
// semicolon if the result is not empty.
func buildInlineStylesFromMap(styleMap map[string]property) string {
	inlineStyles := reconstructStyleAttribute(styleMap)
	if inlineStyles != "" {
		inlineStyles += ";"
	}
	return inlineStyles
}

// rebuildCSSFromRules turns parsed CSS rules back into CSS text.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to rebuild.
// Takes symbols ([]ast.Symbol) which provides the symbol table for the AST.
//
// Returns string which contains the rebuilt CSS text, or an empty string if
// rules is empty.
func rebuildCSSFromRules(rules []css_ast.Rule, symbols []ast.Symbol) string {
	if len(rules) == 0 {
		return ""
	}

	cleanedAST := css_ast.AST{
		Rules:   rules,
		Symbols: symbols,
	}
	symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{symbols}}
	options := css_printer.Options{MinifyWhitespace: false}
	result := css_printer.Print(cleanedAST, symbolMap, options)
	return strings.TrimSpace(string(result.CSS))
}
