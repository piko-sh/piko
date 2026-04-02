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

package ast_domain

// Clone creates a shallow copy of the TemplateNode.
//
// Returns *TemplateNode which is a shallow copy of the receiver.
func (n *TemplateNode) Clone() *TemplateNode {
	if n == nil {
		return nil
	}

	return &TemplateNode{
		Key:                cloneExpression(n.Key),
		DirKey:             cloneDirective(n.DirKey),
		DirHTML:            cloneDirective(n.DirHTML),
		GoAnnotations:      cloneGoAnnotations(n.GoAnnotations),
		RuntimeAnnotations: nil,
		AttributeWriters:   cloneDirectWriterSlice(n.AttributeWriters),
		TextContentWriter:  n.TextContentWriter.Clone(),
		CustomEvents:       cloneEventDirectiveMap(n.CustomEvents),
		OnEvents:           cloneEventDirectiveMap(n.OnEvents),
		Binds:              cloneBindsMap(n.Binds),
		TimelineDirectives: cloneBindsMap(n.TimelineDirectives),
		DirContext:         cloneDirective(n.DirContext),
		DirElse:            cloneDirective(n.DirElse),
		DirText:            cloneDirective(n.DirText),
		DirStyle:           cloneDirective(n.DirStyle),
		DirClass:           cloneDirective(n.DirClass),
		DirIf:              cloneDirective(n.DirIf),
		DirElseIf:          cloneDirective(n.DirElseIf),
		DirFor:             cloneDirective(n.DirFor),
		DirShow:            cloneDirective(n.DirShow),
		DirRef:             cloneDirective(n.DirRef),
		DirSlot:            cloneDirective(n.DirSlot),
		DirModel:           cloneDirective(n.DirModel),
		DirScaffold:        cloneDirective(n.DirScaffold),
		TagName:            n.TagName,
		TextContent:        n.TextContent,
		InnerHTML:          n.InnerHTML,
		PrerenderedHTML:    n.PrerenderedHTML,
		Children:           nil,
		RichText:           cloneRichTextParts(n.RichText),
		Attributes:         cloneHTMLAttributes(n.Attributes),
		Diagnostics:        cloneDiagnostics(n.Diagnostics),
		DynamicAttributes:  cloneDynamicAttributes(n.DynamicAttributes),
		Directives:         cloneDirectives(n.Directives),
		Location:           n.Location,
		NodeRange:          n.NodeRange,
		OpeningTagRange:    n.OpeningTagRange,
		ClosingTagRange:    n.ClosingTagRange,
		NodeType:           n.NodeType,
		PreferredFormat:    n.PreferredFormat,
		IsPooled:           false,
		IsContentEditable:  n.IsContentEditable,
		PreserveWhitespace: n.PreserveWhitespace,
	}
}

// DeepClone creates a full, deep copy of the TemplateNode and its entire
// subtree of children.
//
// Returns *TemplateNode which is a deep copy of the receiver.
func (n *TemplateNode) DeepClone() *TemplateNode {
	return n.deepCloneWithDepth(0)
}

// DeepCloneWithScopeAttributes creates a deep copy of the node and its entire
// subtree, injecting `partial` and `p-key` attributes into every element node.
// This is used for prerendering to ensure CSS scoping and hydration work
// correctly for prerendered content.
//
// Takes partialScopeID (string) which is the partial scope for CSS scoping.
//
// Returns *TemplateNode which is the cloned subtree with attributes injected.
func (n *TemplateNode) DeepCloneWithScopeAttributes(partialScopeID string) *TemplateNode {
	return n.deepCloneWithScopeAttributesRecursive(partialScopeID, 0)
}

// deepCloneWithScopeAttributesRecursive performs deep cloning with attribute
// injection and depth tracking to prevent stack overflow.
//
// Takes partialScopeID (string) which identifies the scope for attribute
// injection.
// Takes depth (int) which tracks recursion depth to prevent stack overflow.
//
// Returns *TemplateNode which is the cloned node with scope attributes
// injected.
func (n *TemplateNode) deepCloneWithScopeAttributesRecursive(partialScopeID string, depth int) *TemplateNode {
	if n == nil {
		return nil
	}

	if depth > MaxCloneDepth {
		return n.Clone()
	}

	clone := n.Clone()

	if clone.NodeType == NodeElement {
		clone.Attributes = injectScopeAttributes(clone.Attributes, clone.Key, partialScopeID)
	}

	if len(n.Children) > 0 {
		clone.Children = make([]*TemplateNode, len(n.Children))
		for i, child := range n.Children {
			clone.Children[i] = child.deepCloneWithScopeAttributesRecursive(partialScopeID, depth+1)
		}
	}

	return clone
}

// Reset clears all fields, making the node ready for reuse.
// When nodes are arena-allocated, the arena handles memory reuse.
func (n *TemplateNode) Reset() {
	n.resetPrimitiveFields()
	n.resetSliceFields()
	n.resetDirectiveFields()
	n.returnPooledResources()
	n.clearEventMaps()
}

// resetPrimitiveFields sets all basic fields back to their zero values.
func (n *TemplateNode) resetPrimitiveFields() {
	n.NodeType = 0
	n.TagName = ""
	n.TextContent = ""
	n.InnerHTML = ""
	n.PreferredFormat = FormatAuto
	n.IsPooled = false
	n.IsContentEditable = false
	n.PreserveWhitespace = false
}

// resetSliceFields clears all slice fields on the node by setting them to nil
// or to zero length.
func (n *TemplateNode) resetSliceFields() {
	n.Attributes = nil
	n.Children = nil
	n.PrerenderedHTML = nil
	n.DynamicAttributes = n.DynamicAttributes[:0]
	n.Directives = n.Directives[:0]
	n.Diagnostics = n.Diagnostics[:0]
	n.RichText = n.RichText[:0]
}

// resetDirectiveFields clears all directive and annotation fields to nil.
func (n *TemplateNode) resetDirectiveFields() {
	n.DirIf = nil
	n.DirElseIf = nil
	n.DirElse = nil
	n.DirFor = nil
	n.DirShow = nil
	n.DirModel = nil
	n.DirRef = nil
	n.DirSlot = nil
	n.DirClass = nil
	n.DirStyle = nil
	n.DirText = nil
	n.DirHTML = nil
	n.DirKey = nil
	n.DirContext = nil
	n.DirScaffold = nil
	n.Key = nil
	n.GoAnnotations = nil
}

// returnPooledResources returns runtime resources to their pools for reuse.
func (n *TemplateNode) returnPooledResources() {
	if n.RuntimeAnnotations != nil {
		PutRuntimeAnnotation(n.RuntimeAnnotations)
		n.RuntimeAnnotations = nil
	}
	n.AttributeWriters = nil
	if n.TextContentWriter != nil {
		PutDirectWriter(n.TextContentWriter)
		n.TextContentWriter = nil
	}
}

// clearEventMaps removes all entries from the event and binding maps.
func (n *TemplateNode) clearEventMaps() {
	for k := range n.OnEvents {
		delete(n.OnEvents, k)
	}
	for k := range n.CustomEvents {
		delete(n.CustomEvents, k)
	}
	for k := range n.Binds {
		delete(n.Binds, k)
	}
	for k := range n.TimelineDirectives {
		delete(n.TimelineDirectives, k)
	}
}

// Clone creates a shallow copy of the TextPart.
//
// Returns TextPart which is a shallow copy of the receiver.
func (tp *TextPart) Clone() TextPart {
	clone := *tp
	if tp.GoAnnotations != nil {
		clone.GoAnnotations = tp.GoAnnotations.Clone()
	}
	if tp.Expression != nil {
		clone.Expression = tp.Expression.Clone()
	}
	return clone
}

// Clone creates a shallow copy of the DynamicAttribute.
//
// Returns DynamicAttribute which is a shallow copy of the receiver.
func (da *DynamicAttribute) Clone() DynamicAttribute {
	clone := *da
	if da.GoAnnotations != nil {
		clone.GoAnnotations = da.GoAnnotations.Clone()
	}
	if da.Expression != nil {
		clone.Expression = da.Expression.Clone()
	}
	return clone
}

// Clone creates a shallow copy of the Directive.
//
// Returns Directive which is a shallow copy of the receiver.
func (d *Directive) Clone() Directive {
	clone := *d
	if d.GoAnnotations != nil {
		clone.GoAnnotations = d.GoAnnotations.Clone()
	}
	if d.ChainKey != nil {
		clone.ChainKey = d.ChainKey.Clone()
	}
	if d.Expression != nil {
		clone.Expression = d.Expression.Clone()
	}
	return clone
}

// Clone creates a deep copy of the PropValue.
//
// Deep cloning the Expression is critical for correct code generation when the
// same partial invocation is processed in different contexts (e.g., SSR
// inlining). Each context may set different BaseCodeGenVarName annotations on
// the expressions, and without deep cloning, these mutations would be shared
// and cause incorrect code.
//
// Returns PropValue which is a deep copy of the receiver.
func (pv *PropValue) Clone() PropValue {
	var clonedExpr Expression
	if pv.Expression != nil {
		clonedExpr = pv.Expression.Clone()
	}
	return PropValue{
		Expression:        clonedExpr,
		Location:          pv.Location,
		NameLocation:      pv.NameLocation,
		InvokerAnnotation: pv.InvokerAnnotation.Clone(),
		GoFieldName:       pv.GoFieldName,
		IsLoopDependent:   pv.IsLoopDependent,
	}
}

// GetGoAnnotation returns the code generation annotation for this prop value.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation.
func (pv *PropValue) GetGoAnnotation() *GoGeneratorAnnotation {
	return pv.InvokerAnnotation
}

// SetGoAnnotation sets the code generation annotation for this prop value.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to use.
func (pv *PropValue) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	pv.InvokerAnnotation = ann
}

// Clone creates a shallow copy of the TemplateAST.
//
// Returns *TemplateAST which is a shallow copy of the receiver.
func (ast *TemplateAST) Clone() *TemplateAST {
	if ast == nil {
		return nil
	}

	rootNodesCopy := make([]*TemplateNode, len(ast.RootNodes))
	copy(rootNodesCopy, ast.RootNodes)

	diagnosticsCopy := make([]*Diagnostic, len(ast.Diagnostics))
	copy(diagnosticsCopy, ast.Diagnostics)

	return &TemplateAST{
		SourcePath:        ast.SourcePath,
		ExpiresAtUnixNano: ast.ExpiresAtUnixNano,
		Metadata:          ast.Metadata,
		queryContext:      nil,
		RootNodes:         rootNodesCopy,
		Diagnostics:       diagnosticsCopy,
		SourceSize:        ast.SourceSize,
		Tidied:            ast.Tidied,
		isPooled:          false,
	}
}

// DeepClone creates a full, deep copy of the TemplateAST and its entire node
// tree.
//
// Returns *TemplateAST which is an independent copy with cloned nodes and
// diagnostics. The queryContext field is not cloned as it is runtime state.
func (ast *TemplateAST) DeepClone() *TemplateAST {
	if ast == nil {
		return nil
	}

	diagnosticsCopy := make([]*Diagnostic, len(ast.Diagnostics))
	copy(diagnosticsCopy, ast.Diagnostics)

	return &TemplateAST{
		SourcePath:        ast.SourcePath,
		ExpiresAtUnixNano: ast.ExpiresAtUnixNano,
		Metadata:          ast.Metadata,
		queryContext:      nil,
		RootNodes:         DeepCloneSlice(ast.RootNodes),
		Diagnostics:       diagnosticsCopy,
		SourceSize:        ast.SourceSize,
		Tidied:            ast.Tidied,
		isPooled:          false,
	}
}

// Clone creates a deep copy of the PartialInvocationInfo.
//
// Returns *PartialInvocationInfo which is the copied instance, or nil if the
// receiver is nil.
func (p *PartialInvocationInfo) Clone() *PartialInvocationInfo {
	if p == nil {
		return nil
	}

	clone := &PartialInvocationInfo{
		InvocationKey:        p.InvocationKey,
		PartialAlias:         p.PartialAlias,
		PartialPackageName:   p.PartialPackageName,
		RequestOverrides:     nil,
		PassedProps:          nil,
		InvokerPackageAlias:  p.InvokerPackageAlias,
		Location:             p.Location,
		InvokerInvocationKey: "",
	}

	if p.RequestOverrides != nil {
		clone.RequestOverrides = make(map[string]PropValue, len(p.RequestOverrides))
		for k, v := range p.RequestOverrides {
			clone.RequestOverrides[k] = v.Clone()
		}
	}

	if p.PassedProps != nil {
		clone.PassedProps = make(map[string]PropValue, len(p.PassedProps))
		for k, v := range p.PassedProps {
			clone.PassedProps[k] = v.Clone()
		}
	}

	return clone
}

// deepCloneWithDepth creates a deep copy with depth tracking to prevent stack
// overflow.
//
// Takes depth (int) which tracks the current recursion level.
//
// Returns *TemplateNode which is the cloned node with all children copied.
func (n *TemplateNode) deepCloneWithDepth(depth int) *TemplateNode {
	if n == nil {
		return nil
	}

	if depth > MaxCloneDepth {
		return n.Clone()
	}

	clone := n.Clone()

	if len(n.Children) > 0 {
		clone.Children = deepCloneSliceWithDepth(n.Children, depth+1)
	}

	return clone
}

// DeepCloneSlice creates a deep copy of a slice of TemplateNode pointers.
//
// Takes nodes ([]*TemplateNode) which is the slice to clone.
//
// Returns []*TemplateNode which is a new slice containing cloned nodes.
func DeepCloneSlice(nodes []*TemplateNode) []*TemplateNode {
	return deepCloneSliceWithDepth(nodes, 0)
}

// cloneHTMLAttributes creates a copy of an HTMLAttribute slice.
//
// Takes attrs ([]HTMLAttribute) which is the slice to copy.
//
// Returns []HTMLAttribute which is a new slice with copied elements.
func cloneHTMLAttributes(attrs []HTMLAttribute) []HTMLAttribute {
	attrsCopy := make([]HTMLAttribute, len(attrs))
	copy(attrsCopy, attrs)
	return attrsCopy
}

// cloneDynamicAttributes creates a deep copy of a DynamicAttribute slice.
//
// Takes attrs ([]DynamicAttribute) which is the slice to copy.
//
// Returns []DynamicAttribute which is a new slice with cloned elements.
func cloneDynamicAttributes(attrs []DynamicAttribute) []DynamicAttribute {
	dynAttrsCopy := make([]DynamicAttribute, len(attrs))
	for i := range attrs {
		dynAttrsCopy[i] = attrs[i].Clone()
	}
	return dynAttrsCopy
}

// cloneDirectives creates a deep copy of a slice of directives.
//
// Takes directives ([]Directive) which is the slice to copy.
//
// Returns []Directive which is a new slice with cloned elements.
func cloneDirectives(directives []Directive) []Directive {
	directivesCopy := make([]Directive, len(directives))
	for i := range directives {
		directivesCopy[i] = directives[i].Clone()
	}
	return directivesCopy
}

// cloneRichTextParts creates a deep copy of a TextPart slice.
//
// Takes parts ([]TextPart) which is the slice to copy.
//
// Returns []TextPart which is a new slice with cloned elements, or nil if
// parts is nil.
func cloneRichTextParts(parts []TextPart) []TextPart {
	if parts == nil {
		return nil
	}
	partsCopy := make([]TextPart, len(parts))
	for i := range parts {
		partsCopy[i] = parts[i].Clone()
	}
	return partsCopy
}

// cloneDiagnostics creates a deep copy of a slice of diagnostics.
//
// Takes diagnostics ([]*Diagnostic) which is the slice to clone.
//
// Returns []*Diagnostic which is a new slice with cloned elements, or nil if
// the input is nil.
func cloneDiagnostics(diagnostics []*Diagnostic) []*Diagnostic {
	if diagnostics == nil {
		return nil
	}
	diagnosticsCopy := make([]*Diagnostic, len(diagnostics))
	for i, d := range diagnostics {
		diagnosticsCopy[i] = d.Clone()
	}
	return diagnosticsCopy
}

// cloneEventDirectiveMap creates a deep copy of an event directive map.
//
// Takes events (map[string][]Directive) which is the source map to copy.
//
// Returns map[string][]Directive which is a new map with cloned directives.
// Returns nil when the input is nil.
func cloneEventDirectiveMap(events map[string][]Directive) map[string][]Directive {
	if events == nil {
		return nil
	}
	eventsCopy := make(map[string][]Directive, len(events))
	for k, v := range events {
		vCopy := make([]Directive, len(v))
		for i := range v {
			vCopy[i] = v[i].Clone()
		}
		eventsCopy[k] = vCopy
	}
	return eventsCopy
}

// cloneBindsMap creates a deep copy of a binds map.
//
// Takes binds (map[string]*Directive) which is the map to copy.
//
// Returns map[string]*Directive which is a new map with copied directives,
// or nil if the input is nil.
func cloneBindsMap(binds map[string]*Directive) map[string]*Directive {
	if binds == nil {
		return nil
	}
	bindsCopy := make(map[string]*Directive, len(binds))
	for k, v := range binds {
		bindsCopy[k] = cloneDirective(v)
	}
	return bindsCopy
}

// cloneGoAnnotations creates a copy of a GoGeneratorAnnotation.
//
// Takes ann (*GoGeneratorAnnotation) which is the annotation to copy.
//
// Returns *GoGeneratorAnnotation which is a deep copy, or nil if ann is nil.
func cloneGoAnnotations(ann *GoGeneratorAnnotation) *GoGeneratorAnnotation {
	if ann == nil {
		return nil
	}
	return ann.Clone()
}

// cloneDirective creates a copy of the given directive.
//
// When d is nil, returns nil.
//
// Takes d (*Directive) which is the directive to copy.
//
// Returns *Directive which is a new copy of the directive.
func cloneDirective(d *Directive) *Directive {
	if d == nil {
		return nil
	}
	return new(d.Clone())
}

// cloneExpression returns a deep copy of the given expression.
//
// When e is nil, returns nil.
//
// Takes e (Expression) which is the expression to copy.
//
// Returns Expression which is the copied expression, or nil if e was nil.
func cloneExpression(e Expression) Expression {
	if e == nil {
		return nil
	}
	return e.Clone()
}

// injectScopeAttributes adds partial and p-key attributes to an attributes
// slice if they are not already present.
//
// Takes attrs ([]HTMLAttribute) which is the existing attributes slice.
// Takes key (Expression) which is the node's key expression (may be nil).
// Takes partialScopeID (string) which is the partial scope ID to add.
//
// Returns []HTMLAttribute which is a new slice with the added attributes.
func injectScopeAttributes(attrs []HTMLAttribute, key Expression, partialScopeID string) []HTMLAttribute {
	needsPartial := partialScopeID != "" && !hasAttributeByName(attrs, "partial")
	needsPKey := key != nil && !hasAttributeByName(attrs, "p-key")

	if !needsPartial && !needsPKey {
		return attrs
	}

	capacity := len(attrs)
	if needsPartial {
		capacity++
	}
	if needsPKey {
		capacity++
	}

	newAttrs := make([]HTMLAttribute, len(attrs), capacity)
	copy(newAttrs, attrs)

	if needsPartial {
		newAttrs = append(newAttrs, HTMLAttribute{Name: "partial", Value: partialScopeID})
	}

	if needsPKey {
		if keyValue := extractStaticKeyString(key); keyValue != "" {
			newAttrs = append(newAttrs, HTMLAttribute{Name: "p-key", Value: keyValue})
		}
	}

	return newAttrs
}

// hasAttributeByName checks whether an attribute with the given name exists.
//
// Takes attrs ([]HTMLAttribute) which is the list of attributes to search.
// Takes name (string) which is the name to look for.
//
// Returns bool which is true if an attribute with the name is found.
func hasAttributeByName(attrs []HTMLAttribute, name string) bool {
	for i := range attrs {
		if attrs[i].Name == name {
			return true
		}
	}
	return false
}

// extractStaticKeyString extracts the string value from a static key
// expression.
// Returns empty string for nil or non-string-literal keys.
//
// Takes key (Expression) which is the key expression to extract from.
//
// Returns string which is the key value, or empty if not extractable.
func extractStaticKeyString(key Expression) string {
	if key == nil {
		return ""
	}
	if sl, ok := key.(*StringLiteral); ok {
		return sl.Value
	}
	return ""
}

// deepCloneSliceWithDepth creates a copy of a slice of template nodes while
// tracking the current cloning depth.
//
// Takes nodes ([]*TemplateNode) which is the slice of nodes to copy.
// Takes depth (int) which tracks how many levels deep the copy has gone.
//
// Returns []*TemplateNode which is a full copy of the input slice, or nil if
// the input is nil.
func deepCloneSliceWithDepth(nodes []*TemplateNode, depth int) []*TemplateNode {
	if nodes == nil {
		return nil
	}

	clonedSlice := make([]*TemplateNode, len(nodes))
	for i, node := range nodes {
		clonedSlice[i] = node.deepCloneWithDepth(depth)
	}
	return clonedSlice
}
