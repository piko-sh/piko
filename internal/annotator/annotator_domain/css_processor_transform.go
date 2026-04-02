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

// Transforms CSS selectors by applying component-specific scoping rules to ensure style isolation.
// Handles special pseudo-classes like :global and :deep, and validates selectors against component root elements.

import (
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	es_logger "piko.sh/piko/internal/esbuild/logger"
)

// cssScopeTransformer holds the state for a single CSS scoping job.
type cssScopeTransformer struct {
	// markers holds patterns to match :global() and :deep() in CSS selectors.
	markers *scopingMarkers

	// scopeID is the unique identifier added to CSS selectors to scope styles.
	scopeID string

	// roots lists the root elements used to check if selectors could match.
	roots []rootDescriptor

	// fileSyms holds symbols from the current file used for marker class lookup.
	fileSyms []es_ast.Symbol

	// keyframesDepth tracks how deep we are inside @keyframes rules.
	keyframesDepth int

	// sourceIndex is the position of the source file used to find symbols.
	sourceIndex uint32
}

// transform applies scope changes to each rule in the given slice.
//
// Takes rules ([]css_ast.Rule) which is the list of CSS rules to update.
func (t *cssScopeTransformer) transform(rules []css_ast.Rule) {
	for i := range rules {
		t.transformRule(&rules[i])
	}
}

// transformRule applies CSS scoping to a single rule based on its type.
//
// Takes r (*css_ast.Rule) which is the rule to transform.
func (t *cssScopeTransformer) transformRule(r *css_ast.Rule) {
	switch data := r.Data.(type) {
	case *css_ast.RSelector:
		t.transformSelectorRule(r, data)
	case *css_ast.RKnownAt:
		t.transformKnownAtRule(data)
	case *css_ast.RAtLayer:
		if data.Rules != nil {
			t.transform(data.Rules)
		}
	case *css_ast.RAtKeyframes:
		t.transformKeyframesRule(data)
	case *css_ast.RAtMedia:
		if data.Rules != nil {
			t.transform(data.Rules)
		}
	}
}

// transformSelectorRule applies CSS scoping to the selectors in a rule.
//
// Takes r (*css_ast.Rule) which provides the location for error messages.
// Takes data (*css_ast.RSelector) which holds the selectors to transform.
func (t *cssScopeTransformer) transformSelectorRule(r *css_ast.Rule, data *css_ast.RSelector) {
	if t.keyframesDepth > 0 || data == nil || len(data.Selectors) == 0 {
		return
	}

	scoped := make([]css_ast.ComplexSelector, 0, len(data.Selectors)*2)
	for _, selector := range data.Selectors {
		scoped = t.transformSingleSelector(scoped, selector, r.Loc)
	}

	if len(scoped) > 0 {
		data.Selectors = scoped
	}
}

// transformSingleSelector processes a single CSS selector and
// applies scoping.
//
// Takes scoped ([]css_ast.ComplexSelector) which collects the
// processed selectors.
// Takes selector (css_ast.ComplexSelector) which is the selector
// to transform.
// Takes loc (es_logger.Loc) which is the source location for
// error reporting.
//
// Returns []css_ast.ComplexSelector which contains the updated
// list of scoped selectors.
func (t *cssScopeTransformer) transformSingleSelector(scoped []css_ast.ComplexSelector, selector css_ast.ComplexSelector, loc es_logger.Loc) []css_ast.ComplexSelector {
	globalMarker, hasGlobalMarker := hasMarkerClass(selector, "__piko_global__", t.fileSyms)
	if hasGlobalMarker {
		return t.handleGlobalMarker(scoped, selector, globalMarker, loc)
	}

	deepMarker, hasDeepMarker := hasMarkerClass(selector, "__piko_deep__", t.fileSyms)
	if hasDeepMarker {
		return t.handleDeepMarker(scoped, selector, deepMarker, loc)
	}

	if shouldSkipScoping(selector) {
		return append(scoped, selector)
	}

	return t.scopeSelectorWithRootCheck(scoped, selector.Clone(), loc)
}

// handleGlobalMarker processes a selector with a :global marker.
//
// The :global() pseudo-class makes the entire selector unscoped, meaning no
// scope attribute is added to any element in the selector. This is used for
// styles that need to target elements outside the partial boundary.
//
// Takes scoped ([]css_ast.ComplexSelector) which holds the selectors built so
// far.
// Takes selector (css_ast.ComplexSelector) which is the selector containing the
// marker.
// Takes marker (*markerLocation) which identifies where the global marker
// appears.
//
// Returns []css_ast.ComplexSelector which is the updated list with the cleaned
// selector added without scoping.
func (*cssScopeTransformer) handleGlobalMarker(scoped []css_ast.ComplexSelector, selector css_ast.ComplexSelector, marker *markerLocation, _ es_logger.Loc) []css_ast.ComplexSelector {
	cleaned := removeMarkerClass(selector, marker)
	return append(scoped, cleaned)
}

// handleDeepMarker processes a selector containing a deep marker.
//
// This implements Vue.js-style :deep() behaviour where the scope attribute is
// attached to the element immediately BEFORE :deep(), not at the beginning
// of the entire selector chain.
//
// Takes scoped ([]css_ast.ComplexSelector) which holds selectors
// processed so far.
// Takes selector (css_ast.ComplexSelector) which is the selector
// containing the deep marker.
// Takes marker (*markerLocation) which identifies the marker
// position to remove.
// Takes loc (es_logger.Loc) which provides the source location
// for diagnostics.
//
// Returns []css_ast.ComplexSelector which contains the scoped
// selectors with the cleaned selector appended.
func (t *cssScopeTransformer) handleDeepMarker(scoped []css_ast.ComplexSelector, selector css_ast.ComplexSelector, marker *markerLocation, loc es_logger.Loc) []css_ast.ComplexSelector {
	if marker.compoundIndex == 0 {
		cleaned := removeMarkerClass(selector, marker)
		return append(scoped, scopeDescendant(cleaned, t.scopeID, loc))
	}

	result := selector.Clone()

	for i := range marker.compoundIndex {
		comp := &result.Selectors[i]
		if !shouldSkipCompoundScoping(comp) {
			addScopeToCompound(comp, t.scopeID)
		}
	}

	return append(scoped, removeMarkerClass(result, marker))
}

// scopeSelectorWithRootCheck adds a direct-scoped version of a selector.
//
// All elements in a partial now have the `partial` attribute, so direct matching
// (.class[partial~=xxx]) is sufficient for proper scoping. Descendant matching
// ([partial~=xxx] .class) is only used with :deep() for intentional child styling.
//
// Takes scoped ([]css_ast.ComplexSelector) which holds the accumulated results.
// Takes selector (css_ast.ComplexSelector) which is the selector to scope.
//
// Returns []css_ast.ComplexSelector which contains the updated scoped selectors.
func (t *cssScopeTransformer) scopeSelectorWithRootCheck(scoped []css_ast.ComplexSelector, selector css_ast.ComplexSelector, _ es_logger.Loc) []css_ast.ComplexSelector {
	return append(scoped, scopeDirect(selector, t.scopeID))
}

// transformKnownAtRule processes a known at-rule. It tracks keyframes depth
// and transforms any nested rules.
//
// Takes data (*css_ast.RKnownAt) which is the at-rule to transform.
func (t *cssScopeTransformer) transformKnownAtRule(data *css_ast.RKnownAt) {
	if strings.Contains(strings.ToLower(data.AtToken), "keyframes") {
		t.keyframesDepth++
		defer func() { t.keyframesDepth-- }()
	}
	if data.Rules != nil {
		t.transform(data.Rules)
	}
}

// transformKeyframesRule processes a keyframes at-rule by transforming its
// nested rule blocks.
//
// Takes data (*css_ast.RAtKeyframes) which contains the keyframes rule to
// transform.
func (t *cssScopeTransformer) transformKeyframesRule(data *css_ast.RAtKeyframes) {
	t.keyframesDepth++
	defer func() { t.keyframesDepth-- }()
	for i := range data.Blocks {
		if data.Blocks[i].Rules != nil {
			t.transform(data.Blocks[i].Rules)
		}
	}
}

// rootDescriptor holds the parsed attributes of a template root element.
type rootDescriptor struct {
	// classes holds CSS class names found in the template.
	classes map[string]struct{}

	// tag is the HTML element name to match against.
	tag string

	// id is the HTML id attribute value of the root element.
	id string

	// hasDynamicClasses indicates the root has a :class binding or p-class directive.
	hasDynamicClasses bool
}

// markerLocation holds the position of a marker class within a compound
// declaration. It tracks both the compound index and the subclass index.
type markerLocation struct {
	// compoundIndex is the index of the compound selector within the complex selector.
	compoundIndex int

	// subclassIndex is the index of the subclass selector within
	// the compound selector.
	subclassIndex int
}

// newCSSScopeTransformer creates a CSS scope transformer for the given scope.
//
// Takes scopeID (string) which identifies the scope for CSS isolation.
// Takes template (*ast_domain.TemplateAST) which provides the template structure.
// Takes symbols ([]es_ast.Symbol) which contains the ES symbols for the file.
//
// Returns *cssScopeTransformer which is ready to transform CSS selectors.
func newCSSScopeTransformer(scopeID string, template *ast_domain.TemplateAST, symbols []es_ast.Symbol) *cssScopeTransformer {
	return &cssScopeTransformer{
		markers:        nil,
		scopeID:        scopeID,
		roots:          extractRootDescriptors(template),
		fileSyms:       symbols,
		keyframesDepth: 0,
		sourceIndex:    0,
	}
}

// extractRootDescriptors builds a list of root descriptors from a template AST.
//
// When the template is nil or has no root nodes, returns nil.
//
// Takes t (*ast_domain.TemplateAST) which provides the parsed template tree.
//
// Returns []rootDescriptor which contains a descriptor for each element node.
func extractRootDescriptors(t *ast_domain.TemplateAST) []rootDescriptor {
	if t == nil || len(t.RootNodes) == 0 {
		return nil
	}

	out := make([]rootDescriptor, 0, len(t.RootNodes))
	for _, n := range t.RootNodes {
		if n.NodeType != ast_domain.NodeElement {
			continue
		}
		out = append(out, buildRootDescriptor(n))
	}
	return out
}

// buildRootDescriptor creates a descriptor for a single root element node.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to process.
//
// Returns rootDescriptor which holds the tag name, ID, and CSS classes.
func buildRootDescriptor(n *ast_domain.TemplateNode) rootDescriptor {
	d := rootDescriptor{
		classes:           make(map[string]struct{}),
		tag:               strings.ToLower(n.TagName),
		id:                "",
		hasDynamicClasses: hasDynamicClasses(n),
	}
	extractStaticAttributes(n, &d)
	return d
}

// extractStaticAttributes reads id and class values from static attributes.
//
// Takes n (*ast_domain.TemplateNode) which provides the node to inspect.
// Takes d (*rootDescriptor) which receives the extracted values.
func extractStaticAttributes(n *ast_domain.TemplateNode, d *rootDescriptor) {
	for i := range n.Attributes {
		a := &n.Attributes[i]
		switch strings.ToLower(a.Name) {
		case "id":
			d.id = strings.TrimSpace(a.Value)
		case "class":
			for c := range strings.FieldsSeq(a.Value) {
				d.classes[strings.ToLower(c)] = struct{}{}
			}
		}
	}
}

// hasDynamicClasses checks if a template node has dynamic class bindings.
//
// Takes n (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node has v-bind:class or :class bindings.
func hasDynamicClasses(n *ast_domain.TemplateNode) bool {
	if n.DirClass != nil {
		return true
	}
	for i := range n.DynamicAttributes {
		if strings.EqualFold(n.DynamicAttributes[i].Name, "class") {
			return true
		}
	}
	return false
}

// shouldSkipScoping checks if a selector should not be scoped.
//
// Takes selector (css_ast.ComplexSelector) which is the selector to check.
//
// Returns bool which is true for :root, html, and body selectors.
func shouldSkipScoping(selector css_ast.ComplexSelector) bool {
	if len(selector.Selectors) == 0 {
		return false
	}

	first := selector.Selectors[0]

	for _, sub := range first.SubclassSelectors {
		if pseudo, ok := sub.Data.(*css_ast.SSPseudoClass); ok {
			if strings.EqualFold(pseudo.Name, "root") {
				return true
			}
		}
	}

	if first.TypeSelector != nil {
		tag := strings.ToLower(first.TypeSelector.Name.Text)
		if tag == "html" || tag == "body" {
			return true
		}
	}

	return false
}

// hasMarkerClass checks if a CSS selector contains a given marker class and
// returns its position.
//
// Takes selector (css_ast.ComplexSelector) which is the CSS selector to search.
// Takes markerName (string) which is the marker class name to find.
// Takes symbols ([]es_ast.Symbol) which is the symbol table for name lookup.
//
// Returns *markerLocation which is the position of the marker class, or nil
// if not found.
// Returns bool which is true if the marker class was found.
func hasMarkerClass(selector css_ast.ComplexSelector, markerName string, symbols []es_ast.Symbol) (*markerLocation, bool) {
	for i, comp := range selector.Selectors {
		for j, sub := range comp.SubclassSelectors {
			if class, ok := sub.Data.(*css_ast.SSClass); ok {
				if int(class.Name.Ref.InnerIndex) < len(symbols) {
					className := symbols[class.Name.Ref.InnerIndex].OriginalName
					if strings.Contains(className, markerName) {
						return &markerLocation{compoundIndex: i, subclassIndex: j}, true
					}
				}
			}
		}
	}
	return nil, false
}

// removeMarkerClass removes the marker class at the given position
// from a selector.
//
// Takes selector (css_ast.ComplexSelector) which is the selector
// to modify.
// Takes loc (*markerLocation) which shows where the marker class
// is.
//
// Returns css_ast.ComplexSelector which is a copy with the marker
// removed. When loc is nil or points to a position that does not
// exist, returns the original selector or a simple copy.
func removeMarkerClass(selector css_ast.ComplexSelector, loc *markerLocation) css_ast.ComplexSelector {
	if loc == nil {
		return selector
	}

	result := selector.Clone()
	if loc.compoundIndex >= len(result.Selectors) {
		return result
	}

	comp := &result.Selectors[loc.compoundIndex]
	if loc.subclassIndex >= len(comp.SubclassSelectors) {
		return result
	}

	newSubclasses := make([]css_ast.SubclassSelector, 0, len(comp.SubclassSelectors)-1)
	for i, sub := range comp.SubclassSelectors {
		if i != loc.subclassIndex {
			newSubclasses = append(newSubclasses, sub)
		}
	}
	comp.SubclassSelectors = newSubclasses

	return result
}
