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
	"cmp"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
)

// RuleSet holds the processed and categorised CSS rules, ready for application.
// It is completely independent of the HTML AST.
type RuleSet struct {
	// InlineableRules holds CSS style rules that can be applied as inline styles.
	InlineableRules []styleRule

	// LeftoverRules holds CSS rules that cannot be inlined into elements,
	// such as media queries or rules with !important properties.
	LeftoverRules []css_ast.Rule

	// PseudoElementRules holds CSS rules for ::before and ::after
	// pseudo-elements. These are only populated when the
	// ResolvePseudoElements option is enabled.
	PseudoElementRules []pseudoElementRule
}

// pseudoElementRule holds a CSS rule targeting a pseudo-element, with the
// pseudo-element name separated from the base selector.
type pseudoElementRule struct {
	// properties maps CSS property names to their parsed values.
	properties map[string]property

	// selector is the base CSS selector without the pseudo-element suffix.
	selector string

	// pseudoElement is the pseudo-element name ("before" or "after").
	pseudoElement string

	// specificity is the CSS specificity score for cascade ordering.
	specificity int
}

// styleRule holds a parsed CSS rule that can be applied inline to elements.
type styleRule struct {
	// properties maps CSS property names to their parsed values.
	properties map[string]property

	// selector is the CSS selector string used to find matching HTML nodes.
	selector string

	// specificity is the CSS specificity value used to sort rules.
	specificity int
}

// property holds a single CSS property value and whether it has the !important
// flag set.
type property struct {
	// value is the CSS property value string.
	value string

	// important indicates whether this property has the CSS !important flag set.
	important bool
}

// ruleProcessingContext holds the data needed to process CSS rules.
// It groups related parameters into a single struct for clarity.
type ruleProcessingContext struct {
	// options holds the CSS parsing settings.
	options *Options

	// diagnostics collects diagnostic messages found while processing rules.
	diagnostics *[]*ast_domain.Diagnostic

	// sourcePath is the file path of the CSS source being processed.
	sourcePath string

	// cssAST holds the parsed CSS abstract syntax tree.
	cssAST css_ast.AST

	// symbolMap maps symbol references to their definitions.
	symbolMap ast.SymbolMap
}

// declarationContext holds the data needed to parse CSS declarations.
// It groups related values to reduce the number of function parameters.
type declarationContext struct {
	// options holds the parser settings such as theme and shorthand expansion.
	options *Options

	// diagnostics collects diagnostic messages found during CSS variable resolution.
	diagnostics *[]*ast_domain.Diagnostic

	// sourcePath is the original CSS file path used for resolving relative
	// imports.
	sourcePath string

	// symbols holds the AST symbols used for variable resolution.
	symbols []ast.Symbol

	// symbolMap provides symbol lookup for converting tokens to strings.
	symbolMap ast.SymbolMap
}

// ProcessCSS transforms a parsed CSS AST into a RuleSet for inlining.
// This is the primary entry point for all CSS processing logic, handling
// categorisation, specificity calculation, and sorting.
//
// Takes cssAST (css_ast.AST) which provides the parsed CSS tree to process.
// Takes options (*Options) which controls processing behaviour.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects warnings and errors
// during processing such as undefined CSS variables.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns *RuleSet which contains the processed rules sorted by specificity.
func ProcessCSS(cssAST css_ast.AST, options *Options, diagnostics *[]*ast_domain.Diagnostic, sourcePath string) *RuleSet {
	ruleSet := &RuleSet{}
	symbolMap := ast.SymbolMap{SymbolsForSource: [][]ast.Symbol{cssAST.Symbols}}

	ctx := ruleProcessingContext{
		cssAST:      cssAST,
		symbolMap:   symbolMap,
		options:     options,
		diagnostics: diagnostics,
		sourcePath:  sourcePath,
	}

	for _, rule := range cssAST.Rules {
		processRule(rule, &ctx, ruleSet)
	}

	sortRulesBySpecificity(ruleSet)

	return ruleSet
}

// processRule handles a single CSS rule by sorting it as inlineable, leftover,
// or pseudo-element. It parses declarations and creates style rules for each
// selector.
//
// Takes rule (css_ast.Rule) which is the CSS rule to process.
// Takes ctx (*ruleProcessingContext) which holds shared state for processing.
// Takes ruleSet (*RuleSet) which collects the sorted rules.
func processRule(rule css_ast.Rule, ctx *ruleProcessingContext, ruleSet *RuleSet) {
	r, ok := rule.Data.(*css_ast.RSelector)
	if !ok {
		ruleSet.LeftoverRules = append(ruleSet.LeftoverRules, rule)
		return
	}

	if ctx.options.ResolvePseudoElements {
		pseudoSelectors, normalSelectors := partitionPseudoElementSelectors(r.Selectors)
		if len(pseudoSelectors) > 0 {
			declCtx := declarationContext{
				symbols:     ctx.cssAST.Symbols,
				symbolMap:   ctx.symbolMap,
				options:     ctx.options,
				diagnostics: ctx.diagnostics,
				sourcePath:  ctx.sourcePath,
			}
			properties := parseDeclarations(r.Rules, &declCtx)
			if len(properties) > 0 {
				addPseudoElementRules(pseudoSelectors, properties, ctx.cssAST.Symbols, ctx.symbolMap, ruleSet)
			}
		}
		if len(normalSelectors) == 0 {
			return
		}

		r = &css_ast.RSelector{Selectors: normalSelectors, Rules: r.Rules}
		rule = css_ast.Rule{Data: r, Loc: rule.Loc}
	}

	if !ctx.options.SkipEmailValidation && !isInlineable(rule) {
		ruleSet.LeftoverRules = append(ruleSet.LeftoverRules, rule)
		return
	}

	declCtx := declarationContext{
		symbols:     ctx.cssAST.Symbols,
		symbolMap:   ctx.symbolMap,
		options:     ctx.options,
		diagnostics: ctx.diagnostics,
		sourcePath:  ctx.sourcePath,
	}

	properties := parseDeclarations(r.Rules, &declCtx)
	if len(properties) == 0 {
		return
	}

	if hasImportantProperties(properties) && ctx.options.KeepBangImportant {
		ruleSet.LeftoverRules = append(ruleSet.LeftoverRules, rule)
	}

	addStyleRulesForSelectors(r.Selectors, properties, ctx.cssAST.Symbols, ctx.symbolMap, ruleSet)
}

// partitionPseudoElementSelectors splits selectors into those targeting
// pseudo-elements (::before, ::after) and normal selectors.
//
// Takes selectors ([]css_ast.ComplexSelector) which contains the selectors to
// partition.
//
// Returns pseudoSelectors which contains selectors with pseudo-elements.
// Returns normalSelectors which contains selectors without pseudo-elements.
func partitionPseudoElementSelectors(selectors []css_ast.ComplexSelector) (
	pseudoSelectors []pseudoSelectorInfo,
	normalSelectors []css_ast.ComplexSelector,
) {
	for _, sel := range selectors {
		if name, stripped, ok := extractPseudoElement(sel); ok {
			pseudoSelectors = append(pseudoSelectors, pseudoSelectorInfo{
				pseudoElement:    name,
				strippedSelector: stripped,
				originalSelector: sel,
			})
		} else {
			normalSelectors = append(normalSelectors, sel)
		}
	}
	return pseudoSelectors, normalSelectors
}

// pseudoSelectorInfo pairs a pseudo-element name with its base selector.
type pseudoSelectorInfo struct {
	// pseudoElement is the pseudo-element name ("before" or "after").
	pseudoElement string

	// strippedSelector is the complex selector with the pseudo-element removed.
	strippedSelector css_ast.ComplexSelector

	// originalSelector is the unmodified complex selector.
	originalSelector css_ast.ComplexSelector
}

// extractPseudoElement checks if the last compound selector in a complex
// selector ends with a ::before or ::after pseudo-element. If so, it returns
// the pseudo-element name, a copy of the selector with the pseudo-element
// removed, and true.
//
// Takes sel (css_ast.ComplexSelector) which is the selector to inspect.
//
// Returns name (string) which is the pseudo-element name.
// Returns stripped (css_ast.ComplexSelector) which is the selector without
// the pseudo-element.
// Returns ok (bool) which is true if a pseudo-element was found.
func extractPseudoElement(sel css_ast.ComplexSelector) (string, css_ast.ComplexSelector, bool) {
	if len(sel.Selectors) == 0 {
		return "", css_ast.ComplexSelector{}, false
	}

	lastIdx := len(sel.Selectors) - 1
	last := sel.Selectors[lastIdx]

	for i, sub := range last.SubclassSelectors {
		pseudo, ok := sub.Data.(*css_ast.SSPseudoClass)
		if !ok || !pseudo.IsElement {
			continue
		}
		if pseudo.Name != "before" && pseudo.Name != "after" {
			continue
		}

		strippedCompound := css_ast.CompoundSelector{
			Combinator:                last.Combinator,
			NestingSelectorLocs:       last.NestingSelectorLocs,
			WasEmptyFromLocalOrGlobal: last.WasEmptyFromLocalOrGlobal,
			TypeSelector:              last.TypeSelector,
			SubclassSelectors:         make([]css_ast.SubclassSelector, 0, len(last.SubclassSelectors)-1),
		}
		strippedCompound.SubclassSelectors = append(strippedCompound.SubclassSelectors, last.SubclassSelectors[:i]...)
		strippedCompound.SubclassSelectors = append(strippedCompound.SubclassSelectors, last.SubclassSelectors[i+1:]...)

		stripped := css_ast.ComplexSelector{
			Selectors: make([]css_ast.CompoundSelector, len(sel.Selectors)),
		}
		copy(stripped.Selectors, sel.Selectors)
		stripped.Selectors[lastIdx] = strippedCompound

		return pseudo.Name, stripped, true
	}

	return "", css_ast.ComplexSelector{}, false
}

// addPseudoElementRules creates pseudo-element rules from the given selectors
// and properties.
//
// Takes selectors ([]pseudoSelectorInfo) which contains the pseudo-element
// selector information.
// Takes properties (map[string]property) which holds the CSS properties.
// Takes symbols ([]ast.Symbol) which provides symbol references.
// Takes symbolMap (ast.SymbolMap) which maps symbol references to names.
// Takes ruleSet (*RuleSet) which receives the created pseudo-element rules.
func addPseudoElementRules(
	selectors []pseudoSelectorInfo,
	properties map[string]property,
	symbols []ast.Symbol,
	symbolMap ast.SymbolMap,
	ruleSet *RuleSet,
) {
	for _, info := range selectors {
		selectorString := complexSelectorToString(info.strippedSelector, symbols, symbolMap)
		ruleSet.PseudoElementRules = append(ruleSet.PseudoElementRules, pseudoElementRule{
			specificity:   calculateSpecificityAST(info.originalSelector),
			selector:      selectorString,
			pseudoElement: info.pseudoElement,
			properties:    properties,
		})
	}
}

// hasImportantProperties checks whether any property has the !important flag.
//
// Takes properties (map[string]property) which contains the CSS properties to
// check.
//
// Returns bool which is true if any property has the !important flag set.
func hasImportantProperties(properties map[string]property) bool {
	for _, prop := range properties {
		if prop.important {
			return true
		}
	}
	return false
}

// addStyleRulesForSelectors creates a style rule for each selector in the list.
//
// Takes selectors ([]css_ast.ComplexSelector) which contains the parsed CSS
// selectors to process.
// Takes properties (map[string]property) which holds the CSS properties for
// the rule.
// Takes symbols ([]ast.Symbol) which provides symbol references for selector
// conversion.
// Takes symbolMap (ast.SymbolMap) which maps symbol references to their names.
// Takes ruleSet (*RuleSet) which receives the created style rules.
func addStyleRulesForSelectors(
	selectors []css_ast.ComplexSelector,
	properties map[string]property,
	symbols []ast.Symbol,
	symbolMap ast.SymbolMap,
	ruleSet *RuleSet,
) {
	for _, complexSel := range selectors {
		selectorString := complexSelectorToString(complexSel, symbols, symbolMap)
		ruleSet.InlineableRules = append(ruleSet.InlineableRules, styleRule{
			specificity: calculateSpecificityAST(complexSel),
			selector:    selectorString,
			properties:  properties,
		})
	}
}

// sortRulesBySpecificity sorts rules by specificity from lowest to highest.
//
// Takes ruleSet (*RuleSet) which contains the rules to sort.
func sortRulesBySpecificity(ruleSet *RuleSet) {
	slices.SortStableFunc(ruleSet.InlineableRules, func(a, b styleRule) int {
		return cmp.Compare(a.specificity, b.specificity)
	})
	slices.SortStableFunc(ruleSet.PseudoElementRules, func(a, b pseudoElementRule) int {
		return cmp.Compare(a.specificity, b.specificity)
	})
}

// parseDeclarations extracts CSS property declarations from a list of rules.
// CSS variables are resolved first, then colours are converted, then shorthand
// properties are expanded if enabled.
//
// Takes rules ([]css_ast.Rule) which contains the CSS rules to parse.
// Takes ctx (*declarationContext) which provides options and variable context.
//
// Returns map[string]property which maps CSS property names to their values.
func parseDeclarations(rules []css_ast.Rule, ctx *declarationContext) map[string]property {
	props := make(map[string]property)
	for _, declRule := range rules {
		if declaration, ok := declRule.Data.(*css_ast.RDeclaration); ok {
			processDeclaration(declaration, ctx, props)
		}
	}
	return props
}

// processDeclaration handles a single CSS declaration and adds it to the
// properties map. It resolves CSS variables, converts colour values, and
// expands shorthand properties when enabled.
//
// Takes declaration (*css_ast.RDeclaration) which is the CSS
// declaration to process.
// Takes ctx (*declarationContext) which provides the context for resolving
// variables.
// Takes props (map[string]property) which stores the resulting properties.
func processDeclaration(declaration *css_ast.RDeclaration, ctx *declarationContext, props map[string]property) {
	propName := declaration.KeyText
	isImportant := declaration.Important

	resolvedTokens := resolveCSSVariablesWithContext(declaration.Value, ctx)

	propValue := tokensToString(resolvedTokens, ctx.symbols, ctx.symbolMap)

	propValue = convertColorValues(propValue)

	if ctx.options.ExpandShorthands {
		expanded := expandShorthand(propName, propValue)
		if expanded != nil {
			for longhandName, longhandValue := range expanded {
				props[longhandName] = property{
					value:     convertColorValues(longhandValue),
					important: isImportant,
				}
			}
			return
		}
	}

	props[propName] = property{
		value:     propValue,
		important: isImportant,
	}
}

// isInlineable checks if a CSS rule can be safely inlined into elements.
//
// It returns false if the rule contains non-structural pseudo-classes or
// pseudo-elements. Structural pseudo-classes like :first-child can be
// evaluated and are considered inlineable. Only RSelector rules are
// candidates; @-rules such as @media or @font-face are not inlineable.
//
// Takes rule (css_ast.Rule) which is the CSS rule to check.
//
// Returns bool which is true if the rule can be safely inlined.
func isInlineable(rule css_ast.Rule) bool {
	r, ok := rule.Data.(*css_ast.RSelector)
	if !ok {
		return false
	}

	for _, complexSel := range r.Selectors {
		if !isComplexSelectorInlineable(complexSel) {
			return false
		}
	}

	return true
}

// isComplexSelectorInlineable checks if a complex selector can be inlined.
//
// Takes selector (css_ast.ComplexSelector) which contains the compound
// selectors to check.
//
// Returns bool which is true if all compound selectors in the complex
// selector can be inlined.
func isComplexSelectorInlineable(selector css_ast.ComplexSelector) bool {
	for _, compoundSel := range selector.Selectors {
		if !isCompoundSelectorInlineable(compoundSel) {
			return false
		}
	}
	return true
}

// isCompoundSelectorInlineable checks whether a compound selector can be
// inlined into an element's style attribute.
//
// Takes selector (css_ast.CompoundSelector) which is the selector to check.
//
// Returns bool which is true if the selector can be safely inlined.
func isCompoundSelectorInlineable(selector css_ast.CompoundSelector) bool {
	if isBodyTypeSelector(selector.TypeSelector) {
		return false
	}

	if isUniversalSelector(selector.TypeSelector) {
		return false
	}

	return areSubclassSelectorsInlineable(selector.SubclassSelectors)
}

// isBodyTypeSelector checks whether a type selector targets the body element.
//
// Takes typeSelector (*css_ast.NamespacedName) which is the CSS type selector
// to check.
//
// Returns bool which is true if the selector targets the body element.
func isBodyTypeSelector(typeSelector *css_ast.NamespacedName) bool {
	return typeSelector != nil && typeSelector.Name.Text == "body"
}

// isUniversalSelector reports whether the given type selector is the universal
// selector (*).
//
// Takes typeSelector (*css_ast.NamespacedName) which is the selector to check.
//
// Returns bool which is true if the selector is the universal selector.
func isUniversalSelector(typeSelector *css_ast.NamespacedName) bool {
	return typeSelector != nil && typeSelector.Name.Text == "*"
}

// areSubclassSelectorsInlineable checks whether all subclass selectors in a
// list can be inlined.
//
// Takes subclasses ([]css_ast.SubclassSelector) which is the list of subclass
// selectors to check.
//
// Returns bool which is true if all selectors can be inlined, or false if any
// selector cannot be inlined.
func areSubclassSelectorsInlineable(subclasses []css_ast.SubclassSelector) bool {
	for _, subClass := range subclasses {
		if !isSubclassSelectorInlineable(subClass) {
			return false
		}
	}
	return true
}

// isSubclassSelectorInlineable checks if a single subclass selector can be
// used as an inline style.
//
// Takes subClass (SubclassSelector) which is the selector to check.
//
// Returns bool which is true if the selector can be inlined, false otherwise.
func isSubclassSelectorInlineable(subClass css_ast.SubclassSelector) bool {
	switch pseudoClass := subClass.Data.(type) {
	case *css_ast.SSPseudoClass:
		return isStructuralPseudoClass(pseudoClass)

	case *css_ast.SSPseudoClassWithSelectorList:
		return false

	case *css_ast.SSHash, *css_ast.SSClass, *css_ast.SSAttribute:
		return true

	default:
		return true
	}
}

// isStructuralPseudoClass checks if a pseudo-class is structural.
//
// Structural pseudo-classes depend only on DOM structure. They can be worked
// out and inlined when the HTML is changed.
//
// Takes pseudoClass (*css_ast.SSPseudoClass) which is the pseudo-class to check.
//
// Returns bool which is true if the pseudo-class is structural and can be
// inlined.
func isStructuralPseudoClass(pseudoClass *css_ast.SSPseudoClass) bool {
	if pseudoClass.IsElement {
		return false
	}

	structuralPseudoClasses := map[string]bool{
		"first-child":      true,
		"last-child":       true,
		"first-of-type":    true,
		"last-of-type":     true,
		"nth-child":        true,
		"nth-last-child":   true,
		"nth-of-type":      true,
		"nth-last-of-type": true,
		"only-child":       true,
		"only-of-type":     true,
		"empty":            true,
		"root":             false,
	}

	return structuralPseudoClasses[pseudoClass.Name]
}

// calculateSpecificityAST computes a numeric score for a CSS selector by
// traversing its AST.
//
// This is far more accurate than regex-based approaches. The scoring is based
// on the W3C standard: ID > class/attribute/pseudo-class >
// element/pseudo-element.
//
// Takes selector (css_ast.ComplexSelector) which is the parsed selector to score.
//
// Returns int which is the computed specificity score.
func calculateSpecificityAST(selector css_ast.ComplexSelector) int {
	var a, b, c int

	for _, compoundSel := range selector.Selectors {
		if compoundSel.TypeSelector != nil && compoundSel.TypeSelector.Name.Text != "*" {
			c++
		}

		for _, subClass := range compoundSel.SubclassSelectors {
			switch s := subClass.Data.(type) {
			case *css_ast.SSHash:
				a++
			case *css_ast.SSClass, *css_ast.SSAttribute:
				b++
			case *css_ast.SSPseudoClass:
				if s.IsElement {
					c++
				} else {
					b++
				}
			}
		}
	}

	return (a * specificityWeightID) + (b * specificityWeightClass) + (c * specificityWeightElement)
}

// tokensToString converts a list of CSS tokens into a string, replacing any
// symbol references with their original names.
//
// Takes tokens ([]css_ast.Token) which contains the CSS tokens to convert.
// Takes symbols ([]ast.Symbol) which provides the symbol table.
// Takes symbolMap (ast.SymbolMap) which maps references to their symbols.
//
// Returns string which is the rebuilt CSS text with symbols resolved.
func tokensToString(tokens []css_ast.Token, symbols []ast.Symbol, symbolMap ast.SymbolMap) string {
	var builder strings.Builder
	for i, t := range tokens {
		if (t.Whitespace&css_ast.WhitespaceBefore) != 0 && i > 0 {
			_ = builder.WriteByte(' ')
		}
		switch t.Kind {
		case css_lexer.TSymbol:
			ref := ast.Ref{SourceIndex: 0, InnerIndex: t.PayloadIndex}
			symbol := symbolMap.Get(ref)
			builder.WriteString(symbol.OriginalName)
		case css_lexer.TFunction:
			builder.WriteString(t.Text)
			_ = builder.WriteByte('(')
			if t.Children != nil {
				builder.WriteString(tokensToString(*t.Children, symbols, symbolMap))
			}
			_ = builder.WriteByte(')')
		case css_lexer.THash:
			_ = builder.WriteByte('#')
			builder.WriteString(t.Text)
		default:
			builder.WriteString(t.Text)
		}
		if (t.Whitespace & css_ast.WhitespaceAfter) != 0 {
			_ = builder.WriteByte(' ')
		}
	}
	return builder.String()
}

// complexSelectorToString converts a complex CSS selector AST back into a
// string, replacing symbol references with their original names.
//
// Takes selector (css_ast.ComplexSelector) which is the
// complex selector to convert.
// Takes symbols ([]ast.Symbol) which provides the symbol definitions.
// Takes symbolMap (ast.SymbolMap) which maps symbol references to definitions.
//
// Returns string which is the CSS selector text with combinators and spaces.
func complexSelectorToString(selector css_ast.ComplexSelector, symbols []ast.Symbol, symbolMap ast.SymbolMap) string {
	var builder strings.Builder
	for i, compound := range selector.Selectors {
		if i > 0 {
			if compound.Combinator.Byte != 0 {
				_ = builder.WriteByte(' ')
				_ = builder.WriteByte(compound.Combinator.Byte)
				_ = builder.WriteByte(' ')
			} else {
				_ = builder.WriteByte(' ')
			}
		}
		builder.WriteString(compoundSelectorToString(compound, symbols, symbolMap))
	}
	return builder.String()
}

// compoundSelectorToString converts a compound selector to its string form.
//
// A compound selector is a single unit like "div.card" that combines a type
// selector with zero or more subclass selectors.
//
// Takes selector (css_ast.CompoundSelector) which is the compound selector to
// convert.
// Takes symbols ([]ast.Symbol) which provides symbol definitions for name
// lookup.
// Takes symbolMap (ast.SymbolMap) which maps symbol references to their
// definitions.
//
// Returns string which is the string form of the compound selector.
func compoundSelectorToString(selector css_ast.CompoundSelector, symbols []ast.Symbol, symbolMap ast.SymbolMap) string {
	var builder strings.Builder
	if selector.TypeSelector != nil {
		builder.WriteString(selector.TypeSelector.Name.Text)
	}
	for _, ss := range selector.SubclassSelectors {
		appendSubclassSelector(&builder, ss, symbols, symbolMap)
	}
	return builder.String()
}

// appendSubclassSelector writes a CSS subclass selector to the string builder.
// It handles different selector types: hash, class, attribute, and pseudo-class.
//
// Takes builder (*strings.Builder) which receives the formatted selector output.
// Takes ss (css_ast.SubclassSelector) which is the selector to format.
// Takes symbols ([]ast.Symbol) which provides symbol data for pseudo-classes.
// Takes symbolMap (ast.SymbolMap) which maps symbol references to names.
func appendSubclassSelector(builder *strings.Builder, ss css_ast.SubclassSelector, symbols []ast.Symbol, symbolMap ast.SymbolMap) {
	switch s := ss.Data.(type) {
	case *css_ast.SSHash:
		appendHashSelector(builder, s, symbolMap)
	case *css_ast.SSClass:
		appendClassSelector(builder, s, symbolMap)
	case *css_ast.SSAttribute:
		appendAttributeSelector(builder, s)
	case *css_ast.SSPseudoClass:
		appendPseudoClassSelector(builder, s, symbols, symbolMap)
	case *css_ast.SSPseudoClassWithSelectorList:
		appendPseudoClassWithSelectorList(builder, s, symbols, symbolMap)
	}
}

// appendHashSelector appends an ID selector (e.g. "#header") to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted selector.
// Takes s (*css_ast.SSHash) which contains the hash selector to append.
// Takes symbolMap (ast.SymbolMap) which maps symbol references to names.
func appendHashSelector(builder *strings.Builder, s *css_ast.SSHash, symbolMap ast.SymbolMap) {
	ref := ast.Ref{SourceIndex: 0, InnerIndex: s.Name.Ref.InnerIndex}
	symbol := symbolMap.Get(ref)
	_ = builder.WriteByte('#')
	builder.WriteString(symbol.OriginalName)
}

// appendClassSelector appends a class selector (e.g. ".card") to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted selector string.
// Takes s (*css_ast.SSClass) which provides the class selector to append.
// Takes symbolMap (ast.SymbolMap) which resolves symbol references to their
// original names.
func appendClassSelector(builder *strings.Builder, s *css_ast.SSClass, symbolMap ast.SymbolMap) {
	ref := ast.Ref{SourceIndex: 0, InnerIndex: s.Name.Ref.InnerIndex}
	symbol := symbolMap.Get(ref)
	_ = builder.WriteByte('.')
	builder.WriteString(symbol.OriginalName)
}

// appendAttributeSelector appends a CSS attribute selector to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes s (*css_ast.SSAttribute) which holds the attribute selector data.
func appendAttributeSelector(builder *strings.Builder, s *css_ast.SSAttribute) {
	_ = builder.WriteByte('[')
	builder.WriteString(s.NamespacedName.Name.Text)
	if s.MatcherOp != "" {
		builder.WriteString(s.MatcherOp)
		builder.WriteString(`"` + strings.ReplaceAll(s.MatcherValue, `"`, `\"`) + `"`)
	}
	_ = builder.WriteByte(']')
}

// appendPseudoClassSelector writes a CSS pseudo-class or pseudo-element
// selector to the string builder. It handles selectors like ":hover" for
// pseudo-classes or "::before" for pseudo-elements.
//
// Takes builder (*strings.Builder) which receives the formatted selector.
// Takes s (*css_ast.SSPseudoClass) which contains the pseudo-class details.
// Takes symbols ([]ast.Symbol) which provides symbol resolution.
// Takes symbolMap (ast.SymbolMap) which maps symbol references.
func appendPseudoClassSelector(builder *strings.Builder, s *css_ast.SSPseudoClass, symbols []ast.Symbol, symbolMap ast.SymbolMap) {
	if s.IsElement {
		builder.WriteString("::")
	} else {
		_ = builder.WriteByte(':')
	}
	builder.WriteString(s.Name)
	if s.Args != nil {
		_ = builder.WriteByte('(')
		builder.WriteString(tokensToString(s.Args, symbols, symbolMap))
		_ = builder.WriteByte(')')
	}
}

// appendPseudoClassWithSelectorList writes a functional pseudo-class to the
// builder (e.g. ":is(.a, .b)").
//
// Takes builder (*strings.Builder) which collects the CSS output.
// Takes s (*css_ast.SSPseudoClassWithSelectorList) which holds the
// pseudo-class name and its list of selectors.
// Takes symbols ([]ast.Symbol) which holds symbol definitions for lookups.
// Takes symbolMap (ast.SymbolMap) which links symbol references to their
// definitions.
func appendPseudoClassWithSelectorList(builder *strings.Builder, s *css_ast.SSPseudoClassWithSelectorList, symbols []ast.Symbol, symbolMap ast.SymbolMap) {
	_ = builder.WriteByte(':')
	builder.WriteString(s.Kind.String())
	_ = builder.WriteByte('(')
	for i, selector := range s.Selectors {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(complexSelectorToString(selector, symbols, symbolMap))
	}
	_ = builder.WriteByte(')')
}
