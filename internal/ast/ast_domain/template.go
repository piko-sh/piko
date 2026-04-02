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

// Defines core template AST types including TemplateAST, TemplateNode, and
// related structures for representing parsed templates. Provides node types,
// HTML attributes, text parts, and methods for cloning, resetting, and managing
// template tree structures.

import (
	"cmp"
	"fmt"
	"strings"
)

// NodeType represents the kind of node in a TemplateAST. It implements
// fmt.Stringer.
type NodeType int

const (
	// NodeElement is an HTML element node in the template AST.
	NodeElement NodeType = iota

	// NodeText represents a text node type in the AST.
	NodeText

	// NodeComment represents a comment node in the AST.
	NodeComment

	// NodeFragment represents a transparent grouping container
	// node (e.g. <fragment> or <template> without shadowrootmode).
	NodeFragment

	// NodeRawHTML represents raw HTML content that should be injected without
	// escaping. This is used for complex conditional comments and VML structures
	// that cannot be represented as standard elements or comments.
	NodeRawHTML
)

const (
	// MaxCloneDepth is the limit for how deep recursive cloning can go.
	// This prevents stack overflow errors when cloning deeply nested trees.
	MaxCloneDepth = 1000

	// attributeNameClass is the HTML class attribute name used when getting or setting
	// CSS classes on template nodes.
	attributeNameClass = "class"
)

// FormatHint indicates the preferred formatting style for a template node.
// It implements fmt.Stringer and is used by the formatter to keep the user's
// original choice of inline or block formatting.
type FormatHint int8

const (
	// FormatAuto is the default format hint that lets the formatter choose based
	// on the content.
	FormatAuto FormatHint = iota

	// FormatInline indicates the node was originally formatted inline (e.g.,
	// <li>Item</li>) and the formatter should preserve this compact style.
	FormatInline

	// FormatBlock indicates the node was originally formatted with children
	// on separate lines. The formatter should preserve this block style.
	FormatBlock
)

// String returns the name of the node type in a readable format.
//
// Returns string which is the name of the node type.
func (n NodeType) String() string {
	switch n {
	case NodeElement:
		return "NodeElement"
	case NodeText:
		return "NodeText"
	case NodeComment:
		return "NodeComment"
	case NodeFragment:
		return "NodeFragment"
	case NodeRawHTML:
		return "NodeRawHTML"
	default:
		return "UnknownNode"
	}
}

// String returns a human-readable name for the FormatHint.
//
// Returns string which is the name of the format hint constant.
func (f FormatHint) String() string {
	switch f {
	case FormatAuto:
		return "FormatAuto"
	case FormatInline:
		return "FormatInline"
	case FormatBlock:
		return "FormatBlock"
	default:
		return "UnknownFormat"
	}
}

// TextPart represents a segment of a text node. It can be either a static
// literal or a dynamic expression to be interpolated (e.g., {{ user.name }}).
type TextPart struct {
	// Expression holds the parsed expression tree for dynamic parts.
	// Nil when IsLiteral is true.
	Expression Expression

	// GoAnnotations holds code generation settings for this text part.
	GoAnnotations *GoGeneratorAnnotation

	// Literal holds the static text content when IsLiteral is true.
	Literal string

	// RawExpression is the original expression string before parsing (e.g.,
	// "user.name"). Empty when IsLiteral is true.
	RawExpression string

	// Location is the source position where this text part starts.
	Location Location

	// IsLiteral indicates whether this part is static text (true) or an expression
	// (false).
	IsLiteral bool
}

// HTMLAttribute represents a standard, static HTML attribute (e.g.,
// class="container").
type HTMLAttribute struct {
	// Name is the attribute name, such as "class", "id", or "href".
	Name string

	// Value is the attribute's string value. Empty for boolean attributes.
	Value string

	// Location marks where the attribute value starts in the source file.
	Location Location

	// NameLocation is the source position where the attribute name begins.
	NameLocation Location

	// AttributeRange spans from the start of the attribute name to the end of
	// the value, or to the end of the name for boolean attributes.
	AttributeRange Range
}

// DynamicAttribute represents an HTML attribute with a dynamic value,
// bound using the : shorthand syntax (e.g., :title="page.title").
type DynamicAttribute struct {
	// Expression is the parsed expression tree that gives the attribute value.
	Expression Expression

	// GoAnnotations holds code generation settings for this expression.
	GoAnnotations *GoGeneratorAnnotation

	// Name is the attribute name, such as "title" for :title="...".
	Name string

	// RawExpression is the original expression string before parsing.
	RawExpression string

	// Location is the source position where the expression value starts.
	Location Location

	// NameLocation is where the attribute name starts in the source.
	NameLocation Location

	// AttributeRange spans the full attribute from the colon prefix to the
	// closing quote.
	AttributeRange Range
}

// TemplateAST is the root of a parsed template file. It contains the root nodes
// of the tree and all diagnostics found during parsing and transformation.
type TemplateAST struct {
	// SourcePath is the file path of the template source; nil if parsed from a
	// string.
	SourcePath *string

	// ExpiresAtUnixNano is the cache expiry time in nanoseconds since the Unix
	// epoch. Nil means no expiry is set.
	ExpiresAtUnixNano *int64

	// Metadata is the optional YAML front matter from the template.
	Metadata *string

	// queryContext caches the query state for CSS selector searches; not cloned.
	queryContext *QueryContext

	// arena is the RenderArena that owns all nodes in this AST. When set,
	// PutTree releases the arena instead of doing recursive node cleanup.
	arena *RenderArena

	// RootNodes holds the top-level nodes of the parsed template tree.
	RootNodes []*TemplateNode

	// Diagnostics holds any warnings or errors found during parsing.
	Diagnostics []*Diagnostic

	// SourceSize is the original source file size in bytes. Used for metrics and
	// to decide whether to process the template in parallel.
	SourceSize int64

	// Tidied indicates whether TidyAST has been run on this tree.
	Tidied bool

	// isPooled indicates whether this AST came from a sync.Pool.
	isPooled bool
}

// PropValue represents a property value passed to a partial component.
// It holds the expression, location, and code generation settings for a single
// property in a partial invocation.
type PropValue struct {
	// Expression holds the parsed expression tree for this property value.
	Expression Expression

	// InvokerAnnotation holds code generation settings from the calling context.
	InvokerAnnotation *GoGeneratorAnnotation

	// GoFieldName is the Go struct field name that matches this property.
	GoFieldName string

	// Location is the source position where the property value expression starts.
	Location Location

	// NameLocation specifies where the property name begins in source, used for
	// diagnostic messages.
	NameLocation Location

	// IsLoopDependent indicates whether this expression uses a loop variable.
	// This affects how code is generated for partial invocations.
	IsLoopDependent bool
}

// PartialInvocationInfo contains metadata about a partial component invocation.
// It is used during code generation to properly instantiate and render partial
// templates.
type PartialInvocationInfo struct {
	// InvocationKey is a canonical identifier for this specific invocation site,
	// computed from the partial name and passed props. Used for caching and
	// deduplication.
	InvocationKey string

	// PartialAlias is the import alias for the partial's package in generated
	// code.
	PartialAlias string

	// PartialPackageName is the full Go package path of the partial being invoked.
	PartialPackageName string

	// RequestOverrides maps request property names to their override values.
	// These values take priority over the default request settings.
	RequestOverrides map[string]PropValue

	// PassedProps maps property names to values passed to the partial at the call
	// site.
	PassedProps map[string]PropValue

	// InvokerPackageAlias is the import alias for the package that contains the
	// invoking page or partial.
	InvokerPackageAlias string

	// InvokerInvocationKey is the canonical key of the parent partial that
	// invokes this one, used to differentiate nested invocations with identical
	// expression strings but different parent contexts (empty for partials
	// invoked directly from pages).
	InvokerInvocationKey string

	// Location is where the partial invocation appears in source code.
	Location Location
}

// TemplateNode is the core building block of the AST, representing an element,
// text, comment, or fragment in the template.
//
// Field order is optimised for memory alignment to minimise struct padding.
type TemplateNode struct {
	// Key is the auto-assigned expression used for DOM reconciliation and
	// component state tracking. Computed by the key assigner during TidyAST from:
	// user-provided p-key (highest priority), loop variables from p-for, context
	// from p-context, or auto-generated indices (e.g., "r.0:1:2" for root 0, child
	// 1, child 2).
	Key Expression

	// DirScaffold holds the p-scaffold directive for component structure
	// generation. Framework-internal directive used during compilation to build
	// static scaffold HTML.
	DirScaffold *Directive

	// DirHTML holds the p-html directive for injecting raw HTML content.
	// Expression results bypass HTML escaping (like Vue's v-html); use only with
	// trusted content.
	DirHTML *Directive

	// GoAnnotations holds code generation settings found during analysis. It
	// contains type information, variable names, and optimisation hints for the
	// Go generator; nil means no annotations are present.
	GoAnnotations *GoGeneratorAnnotation

	// RuntimeAnnotations holds per-render state such as loop iteration context.
	// It is populated during render, returned to pool afterward, and not cloned.
	RuntimeAnnotations *RuntimeAnnotation

	// TextContentWriter holds structured text content parts for zero-allocation
	// rendering of dynamic text with interpolations, used instead of
	// TextContent and pooled/released when the node is returned to pool.
	TextContentWriter *DirectWriter

	// CustomEvents maps custom event names to their p-event handlers for
	// component-specific events (e.g., p-event:update="handleUpdate"). Multiple
	// handlers can be registered for the same event name.
	CustomEvents map[string][]Directive

	// OnEvents maps DOM event names to their p-on handlers for standard browser
	// events (e.g., p-on:click="handleClick"), equivalent to Vue's v-on/@
	// shorthand with support for modifiers like .prevent and .stop.
	OnEvents map[string][]Directive

	// Binds maps attribute names to p-bind directives for dynamic attribute
	// binding. Works like Vue's v-bind or the : shorthand.
	Binds map[string]*Directive

	// TimelineDirectives maps timeline directive arguments to
	// their p-timeline directives (e.g. p-timeline:hidden).
	//
	// Used by the animation behaviour to control element
	// visibility during timeline playback.
	TimelineDirectives map[string]*Directive

	// DirContext holds the p-context directive for scoping key generation. It sets
	// a prefix for automatic key assignment within the subtree, which affects how
	// child node Key fields are computed during TidyAST.
	DirContext *Directive

	// DirElse holds the p-else directive, the fallback branch in an if-else
	// chain rendered when all preceding p-if and p-else-if conditions are false,
	// with no expression (always nil) and a ChainKey linking back to the
	// originating p-if.
	DirElse *Directive

	// DirText holds the p-text directive for setting element text content from
	// an expression whose evaluated result is HTML-escaped and replaces any
	// static text content, equivalent to Vue's v-text.
	DirText *Directive

	// DirStyle holds the p-style directive for dynamic inline style binding.
	DirStyle *Directive

	// DirClass holds the p-class directive for dynamic class binding.
	DirClass *Directive

	// DirIf holds the p-if directive for conditional rendering.
	//
	// Equivalent to Vue's v-if. When the expression evaluates to false,
	// the element and all its descendants are excluded from output.
	DirIf *Directive

	// DirElseIf holds the p-else-if directive, forming part of an if-else chain.
	// Evaluated only when the preceding p-if was false; uses ChainKey to link back
	// to the originating p-if node.
	DirElseIf *Directive

	// DirFor holds the p-for loop directive for iterating over collections.
	// The expression is a ForInExpression with format "item in items" or
	// "(index, item) in items"; equivalent to Vue's v-for.
	DirFor *Directive

	// DirShow holds the p-show directive for visibility control via CSS display.
	// Unlike p-if, the element is always rendered but conditionally shown/hidden,
	// similar to Vue's v-show.
	DirShow *Directive

	// DirRef holds the p-ref directive for creating a named reference to
	// this element, making the DOM element accessible to scripts. It is
	// equivalent to Vue's ref attribute.
	DirRef *Directive

	// DirSlot holds the p-slot directive for assigning this element to a named
	// slot during partial expansion. The value is a raw string slot name (e.g.,
	// "header").
	DirSlot *Directive

	// DirModel holds the p-model directive for two-way data binding on form
	// elements (input, textarea, select), similar to Vue's v-model.
	DirModel *Directive

	// DirKey holds the p-key directive for setting a unique key in loops.
	// This key identifies each item when the list changes.
	DirKey *Directive

	// TextContent holds plain text for NodeText nodes without interpolations.
	// When a text node contains {{ }} expressions, RichText is used instead.
	TextContent string

	// TagName is the HTML element tag name, such as "div", "span", or
	// "custom-component". Empty for non-element nodes like text, comment, or
	// fragment.
	TagName string

	// InnerHTML holds raw HTML content for NodeRawHTML nodes. Used for complex
	// conditional comments (e.g., <!--[if mso]>...</!--[endif]-->) and VML
	// structures that cannot be shown as standard elements.
	InnerHTML string

	// PrerenderedHTML holds precomputed HTML bytes for this entire subtree,
	// set only when GoAnnotations.IsFullyPrerenderable is true, enabling
	// zero-copy output to the quicktemplate writer without walking the AST.
	PrerenderedHTML []byte

	// Children contains the nested child nodes within this element or fragment.
	Children []*TemplateNode

	// RichText contains text parts for nodes with {{ }} interpolations.
	// Alternates between literal strings and expression parts; when empty,
	// TextContent holds plain static text instead.
	RichText []TextPart

	// Attributes holds static HTML attributes as name-value pairs.
	// For dynamic attributes, use DynamicAttributes or Binds instead.
	Attributes []HTMLAttribute

	// Diagnostics holds parsing warnings and errors for this node.
	Diagnostics []*Diagnostic

	// DynamicAttributes holds attributes that use the :attr="expr" shorthand.
	// Each entry has a name and an expression that is checked at render time.
	DynamicAttributes []DynamicAttribute

	// Directives holds directives that do not have their own fields.
	// Most directives are moved to specific Dir* fields during TidyAST.
	Directives []Directive

	// AttributeWriters holds DirectWriters for dynamic attributes that need
	// zero-allocation rendering, each identified by a Name field (e.g.,
	// "title", "href", "p-key") and pooled/released when the node is returned
	// to pool.
	AttributeWriters []*DirectWriter

	// ClosingTagRange is the range of the closing tag, from '</' to '>'. This is
	// synthetic for void or self-closing elements and is only relevant for
	// NodeElement.
	ClosingTagRange Range

	// OpeningTagRange is the position of the opening tag, from '<' to '>'.
	// Only applies to NodeElement.
	OpeningTagRange Range

	// NodeRange is the full span of the node, from the start of the opening tag
	// to the end of the closing tag. For text or comment nodes, this covers all
	// of the content.
	NodeRange Range

	// Location is the position in the source file where this node starts.
	Location Location

	// NodeType identifies the kind of node: NodeElement, NodeText, NodeComment,
	// NodeFragment, or NodeRawHTML. This determines which fields are used.
	NodeType NodeType

	// PreferredFormat indicates the original formatting style of this node,
	// used by the formatter to preserve user intent (inline vs block) and
	// defaulting to FormatAuto (zero value) which defers to heuristics.
	PreferredFormat FormatHint

	// IsPooled indicates whether this node came from a sync.Pool and should be
	// returned to the pool via Reset() when no longer needed.
	IsPooled bool

	// IsContentEditable indicates whether the element has contenteditable="true".
	// This affects how text content is handled during rendering.
	IsContentEditable bool

	// PreserveWhitespace indicates that this text node's whitespace (newlines,
	// tabs, runs of spaces) must not be collapsed. Set for text inside pre,
	// code, textarea, and contenteditable elements.
	PreserveWhitespace bool
}

// directiveSetter assigns a directive to the appropriate field on a node.
type directiveSetter func(n *TemplateNode, d *Directive)

var (
	// directiveSetters maps directive types to their corresponding setter
	// functions.
	directiveSetters = map[DirectiveType]directiveSetter{
		DirectiveIf:       func(n *TemplateNode, d *Directive) { n.DirIf = d },
		DirectiveElseIf:   func(n *TemplateNode, d *Directive) { n.DirElseIf = d },
		DirectiveElse:     func(n *TemplateNode, d *Directive) { n.DirElse = d },
		DirectiveFor:      func(n *TemplateNode, d *Directive) { n.DirFor = d },
		DirectiveShow:     func(n *TemplateNode, d *Directive) { n.DirShow = d },
		DirectiveModel:    func(n *TemplateNode, d *Directive) { n.DirModel = d },
		DirectiveRef:      func(n *TemplateNode, d *Directive) { n.DirRef = d },
		DirectiveSlot:     func(n *TemplateNode, d *Directive) { n.DirSlot = d },
		DirectiveClass:    func(n *TemplateNode, d *Directive) { n.DirClass = d },
		DirectiveStyle:    func(n *TemplateNode, d *Directive) { n.DirStyle = d },
		DirectiveText:     func(n *TemplateNode, d *Directive) { n.DirText = d },
		DirectiveHTML:     func(n *TemplateNode, d *Directive) { n.DirHTML = d },
		DirectiveKey:      func(n *TemplateNode, d *Directive) { n.DirKey = d },
		DirectiveContext:  func(n *TemplateNode, d *Directive) { n.DirContext = d },
		DirectiveScaffold: func(n *TemplateNode, d *Directive) { n.DirScaffold = d },
	}

	// directiveAccessors maps directive types to functions that retrieve the
	// corresponding directive field from a TemplateNode. This dispatch table
	// replaces conditional logic with a lookup.
	directiveAccessors = map[DirectiveType]func(*TemplateNode) *Directive{
		DirectiveIf:      func(n *TemplateNode) *Directive { return n.DirIf },
		DirectiveElseIf:  func(n *TemplateNode) *Directive { return n.DirElseIf },
		DirectiveElse:    func(n *TemplateNode) *Directive { return n.DirElse },
		DirectiveFor:     func(n *TemplateNode) *Directive { return n.DirFor },
		DirectiveShow:    func(n *TemplateNode) *Directive { return n.DirShow },
		DirectiveModel:   func(n *TemplateNode) *Directive { return n.DirModel },
		DirectiveRef:     func(n *TemplateNode) *Directive { return n.DirRef },
		DirectiveSlot:    func(n *TemplateNode) *Directive { return n.DirSlot },
		DirectiveClass:   func(n *TemplateNode) *Directive { return n.DirClass },
		DirectiveStyle:   func(n *TemplateNode) *Directive { return n.DirStyle },
		DirectiveText:    func(n *TemplateNode) *Directive { return n.DirText },
		DirectiveHTML:    func(n *TemplateNode) *Directive { return n.DirHTML },
		DirectiveKey:     func(n *TemplateNode) *Directive { return n.DirKey },
		DirectiveContext: func(n *TemplateNode) *Directive { return n.DirContext },
	}
)

// GetDirective retrieves the first directive of a given type from the
// distributed fields.
//
// Takes dirType (DirectiveType) which specifies the type of directive to find.
//
// Returns *Directive which is the matching directive, or nil if not found.
func (n *TemplateNode) GetDirective(dirType DirectiveType) *Directive {
	if n == nil {
		return nil
	}

	if accessor, ok := directiveAccessors[dirType]; ok {
		return accessor(n)
	}

	for i := range n.Directives {
		if n.Directives[i].Type == dirType {
			return &n.Directives[i]
		}
	}

	return nil
}

// GetDirectives retrieves all directives of a given type, especially for
// repeatable directives.
//
// Takes dirType (DirectiveType) which specifies the type of directive to find.
//
// Returns []Directive which contains all matching directives, or nil if the
// receiver is nil or no directives match.
func (n *TemplateNode) GetDirectives(dirType DirectiveType) []Directive {
	if n == nil {
		return nil
	}

	if dirs, ok := n.getMultiInstanceDirectives(dirType); ok {
		return dirs
	}

	if single := n.GetDirective(dirType); single != nil {
		return []Directive{*single}
	}

	return n.findDirectivesInSlice(dirType)
}

// HasDirective checks if a node has a directive of a given type.
//
// Takes dirType (DirectiveType) which specifies the type of directive to check
// for.
//
// Returns bool which is true if the node has at least one directive of the
// specified type.
func (n *TemplateNode) HasDirective(dirType DirectiveType) bool {
	switch dirType {
	case DirectiveOn:
		return len(n.OnEvents) > 0
	case DirectiveEvent:
		return len(n.CustomEvents) > 0
	case DirectiveBind:
		return len(n.Binds) > 0
	default:
		return n.GetDirective(dirType) != nil
	}
}

// getMultiInstanceDirectives returns directives for types that can have
// multiple instances.
//
// Takes dirType (DirectiveType) which specifies the type of directive to
// collect.
//
// Returns []Directive which contains the collected directives of the
// requested type.
// Returns bool which indicates whether the directive type supports multiple
// instances.
func (n *TemplateNode) getMultiInstanceDirectives(dirType DirectiveType) ([]Directive, bool) {
	switch dirType {
	case DirectiveOn:
		return collectEventDirectives(n.OnEvents), true
	case DirectiveEvent:
		return collectEventDirectives(n.CustomEvents), true
	case DirectiveBind:
		return collectBindDirectives(n.Binds), true
	default:
		return nil, false
	}
}

// findDirectivesInSlice searches the raw Directives slice for entries matching
// the given directive type.
//
// Takes dirType (DirectiveType) which specifies the type of directive to find.
//
// Returns []Directive which contains all matching directives, or nil if none
// are found.
func (n *TemplateNode) findDirectivesInSlice(dirType DirectiveType) []Directive {
	var found []Directive
	for i := range n.Directives {
		if n.Directives[i].Type == dirType {
			found = append(found, n.Directives[i])
		}
	}
	return found
}

// distributeDirectives moves directives from the raw Directives slice into
// typed fields, checking that non-repeatable directives appear only once on
// the element.
func (n *TemplateNode) distributeDirectives() {
	var filtered []Directive
	seenDirectives := make(map[DirectiveType]Location)
	for i := range n.Directives {
		d := &n.Directives[i]
		if !isValidDirective(d) {
			continue
		}
		if shouldCheckDuplicate(d.Type) && isDuplicateDirective(d, seenDirectives, n) {
			continue
		}
		if !distributeDirectiveToNode(n, d, &filtered) {
			filtered = append(filtered, *d)
		}
	}
	n.Directives = filtered
}

// distributeDirectivesRecursively applies the distribution logic to the node
// and all its descendants.
func (n *TemplateNode) distributeDirectivesRecursively() {
	n.distributeDirectives()
	for _, child := range n.Children {
		child.distributeDirectivesRecursively()
	}
}

// SetArena attaches a RenderArena to this AST for cleanup via PutTree. When an
// arena is attached, PutTree will release the arena instead of doing recursive
// node cleanup.
//
// Takes arena (*RenderArena) which is the arena that owns all nodes in this
// AST.
func (t *TemplateAST) SetArena(arena *RenderArena) {
	t.arena = arena
}

// isSingleInstance checks if a directive type may only appear once per element.
//
// Takes d (DirectiveType) which is the directive type to check.
//
// Returns bool which is true if the directive may only appear once.
func isSingleInstance(d DirectiveType) bool {
	switch d {
	case DirectiveIf, DirectiveElseIf, DirectiveElse, DirectiveFor, DirectiveShow,
		DirectiveModel, DirectiveRef, DirectiveClass, DirectiveStyle, DirectiveText,
		DirectiveHTML, DirectiveKey, DirectiveContext, DirectiveScaffold:
		return true
	default:
		return false
	}
}

// isValidDirective checks whether a directive is valid for distribution.
// This is called after parseAllExpressions, so RawExpression values are
// already normalised by the AST transformation layer.
//
// Takes d (*Directive) which is the directive to check.
//
// Returns bool which is true when the directive has an expression, is an
// else directive, or is a raw string directive such as p-ref.
func isValidDirective(d *Directive) bool {
	if d.Type == DirectiveElse || d.Type == DirectiveTimeline {
		return true
	}
	if rawStringDirectives[d.Type] {
		return d.RawExpression != ""
	}
	return d.Expression != nil
}

// shouldCheckDuplicate reports whether the directive type should be checked
// for duplicates.
//
// Takes dt (DirectiveType) which specifies the directive type to check.
//
// Returns bool which is true if the directive type allows only one instance.
func shouldCheckDuplicate(dt DirectiveType) bool {
	return isSingleInstance(dt)
}

// isDuplicateDirective checks if a directive has already been seen and records
// a diagnostic if so.
//
// Takes d (*Directive) which is the directive to check.
// Takes seenDirectives (map[DirectiveType]Location) which tracks directive
// types that have been seen and where they first appeared.
// Takes n (*TemplateNode) which is the node used for recording diagnostics.
//
// Returns bool which is true if the directive is a duplicate.
func isDuplicateDirective(d *Directive, seenDirectives map[DirectiveType]Location, n *TemplateNode) bool {
	if firstLocation, exists := seenDirectives[d.Type]; exists {
		recordDuplicateDirectiveDiagnostic(n, d, firstLocation)
		return true
	}
	seenDirectives[d.Type] = d.NameLocation
	return false
}

// recordDuplicateDirectiveDiagnostic adds a diagnostic when a directive
// appears more than once on the same element.
//
// Takes n (*TemplateNode) which receives the diagnostic.
// Takes d (*Directive) which is the duplicate directive.
// Takes firstLocation (Location) which marks where the first instance appeared.
func recordDuplicateDirectiveDiagnostic(n *TemplateNode, d *Directive, firstLocation Location) {
	directiveName := cmp.Or(DirectiveTypeToName[d.Type], "p-"+strings.ToLower(d.Type.String()))
	message := fmt.Sprintf("Duplicate '%s' directive found on the same element. The first instance was at line %d, column %d.",
		directiveName, firstLocation.Line, firstLocation.Column)
	n.Diagnostics = append(n.Diagnostics, NewDiagnosticWithCode(Warning, message, directiveName, CodeDuplicateDirective, d.NameLocation, ""))
}

// distributeDirectiveToNode assigns a directive to the appropriate field on the
// node.
//
// Takes n (*TemplateNode) which receives the directive assignment.
// Takes d (*Directive) which is the directive to distribute.
//
// Returns bool which is true if the directive was handled, or false
// if it should be added to the filtered list.
func distributeDirectiveToNode(n *TemplateNode, d *Directive, _ *[]Directive) bool {
	if setter, ok := directiveSetters[d.Type]; ok {
		setter(n, d)
		return true
	}

	switch d.Type {
	case DirectiveBind:
		return handleBindDirective(n, d)
	case DirectiveOn:
		return handleOnDirective(n, d)
	case DirectiveEvent:
		return handleEventDirective(n, d)
	case DirectiveTimeline:
		return handleTimelineDirective(n, d)
	default:
		return false
	}
}

// handleBindDirective adds a bind directive to the node's bind map.
//
// When a bind directive with the same argument already exists, a warning is
// added but processing still succeeds.
//
// Takes n (*TemplateNode) which receives the bind directive.
// Takes d (*Directive) which contains the bind directive to add.
//
// Returns bool which is always true to show processing succeeded.
func handleBindDirective(n *TemplateNode, d *Directive) bool {
	if n.Binds == nil {
		n.Binds = make(map[string]*Directive)
	}
	if existing, exists := n.Binds[d.Arg]; exists {
		message := fmt.Sprintf("Duplicate 'p-bind:%s' directive. The first instance was at line %d, column %d.",
			d.Arg, existing.NameLocation.Line, existing.NameLocation.Column)
		n.Diagnostics = append(n.Diagnostics, NewDiagnosticWithCode(Warning, message, "p-bind:"+d.Arg, CodeDuplicateDirective, d.NameLocation, ""))
		return true
	}
	n.Binds[d.Arg] = d
	return true
}

// handleOnDirective processes a p-on event binding directive.
//
// Takes n (*TemplateNode) which receives the parsed event binding.
// Takes d (*Directive) which contains the directive to process.
//
// Returns bool which indicates whether the directive was handled.
func handleOnDirective(n *TemplateNode, d *Directive) bool {
	if n.OnEvents == nil {
		n.OnEvents = make(map[string][]Directive)
	}
	n.OnEvents[d.Arg] = append(n.OnEvents[d.Arg], *d)
	return true
}

// handleEventDirective attaches an event directive to a template node.
//
// Takes n (*TemplateNode) which is the node to attach the event to.
// Takes d (*Directive) which contains the event directive to attach.
//
// Returns bool which is true when the directive was handled.
func handleEventDirective(n *TemplateNode, d *Directive) bool {
	if n.CustomEvents == nil {
		n.CustomEvents = make(map[string][]Directive)
	}
	n.CustomEvents[d.Arg] = append(n.CustomEvents[d.Arg], *d)
	return true
}

// handleTimelineDirective adds a timeline directive to the node's timeline map.
//
// Takes n (*TemplateNode) which receives the timeline directive.
// Takes d (*Directive) which contains the timeline directive to add.
//
// Returns bool which is always true to show processing succeeded.
func handleTimelineDirective(n *TemplateNode, d *Directive) bool {
	if n.TimelineDirectives == nil {
		n.TimelineDirectives = make(map[string]*Directive)
	}
	n.TimelineDirectives[d.Arg] = d
	return true
}

// collectEventDirectives gathers all directives from an event map into a
// single slice.
//
// Takes events (map[string][]Directive) which maps event names to their
// associated directives.
//
// Returns []Directive which contains all directives from all events combined.
func collectEventDirectives(events map[string][]Directive) []Directive {
	if len(events) == 0 {
		return nil
	}
	var all []Directive
	for _, dirs := range events {
		all = append(all, dirs...)
	}
	return all
}

// collectBindDirectives gathers bind directive pointers into a slice, skipping
// nil entries.
//
// Takes binds (map[string]*Directive) which contains the bind directives to
// collect.
//
// Returns []Directive which contains the collected directives, or nil if the
// input map is empty.
func collectBindDirectives(binds map[string]*Directive) []Directive {
	if len(binds) == 0 {
		return nil
	}
	all := make([]Directive, 0, len(binds))
	for _, directive := range binds {
		if directive != nil {
			all = append(all, *directive)
		}
	}
	return all
}
