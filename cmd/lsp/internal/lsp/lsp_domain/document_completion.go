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

package lsp_domain

import (
	"context"
	"fmt"
	goast "go/ast"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// typeBuiltInFunction is the type name for Go built-in functions.
// This must match the constant in annotator_domain/analysis_context.go.
const typeBuiltInFunction = "builtin_function"

var (
	// eventPlaceholders defines the special placeholder variables available in
	// event handlers.
	eventPlaceholders = []struct {
		name          string
		detail        string
		documentation string
	}{
		{
			name:   "$event",
			detail: "js.Event",
			documentation: `**Browser Event Object**
	
	The native JavaScript event object passed to event handlers.
	Pass it to your handler function to access event properties.
	
	**Example:**
	` + "```html" + `
	<button p-on:click="handleClick($event)">Click</button>
	` + "```" + `
	
	**In your handler:**
	` + "```typescript" + `
	function handleClick(event: Event) {
	    event.preventDefault();
	    console.log(event.target);
	}
	` + "```",
		},
		{
			name:   "$form",
			detail: "pk.FormData",
			documentation: `**Form Data Handle**
	
	Collects form data from the closest ancestor <form> element.
	Returns a FormDataHandle with methods to access the data.
	
	**Example:**
	` + "```html" + `
	<form p-on:submit="handleSubmit($form)">
	    <input name="email" />
	    <button type="submit">Submit</button>
	</form>
	` + "```" + `
	
	**Methods available:**
	- ` + "`toObject()`" + ` - Convert to plain object
	- ` + "`toJSON()`" + ` - Convert to JSON string
	- ` + "`get(key)`" + ` - Get a specific field value
	- ` + "`has(key)`" + ` - Check if field exists`,
		},
	}

	// directiveCompletions defines all available directive completions with
	// metadata.
	directiveCompletions = []directiveCompletionInfo{
		{Name: "if", NeedsValue: true, SortPriority: "01"},
		{Name: "else-if", NeedsValue: true, SortPriority: "02"},
		{Name: "else", NeedsValue: false, SortPriority: "03"},
		{Name: "for", NeedsValue: true, SortPriority: "04"},
		{Name: "show", NeedsValue: true, SortPriority: "05"},

		{Name: "text", NeedsValue: true, SortPriority: "10"},
		{Name: "html", NeedsValue: true, SortPriority: "11"},
		{Name: "model", NeedsValue: true, SortPriority: "12"},
		{Name: "bind", NeedsValue: true, NeedsArgument: true, ArgumentPlaceholder: "attr", SortPriority: "13"},

		{Name: "on", NeedsValue: true, NeedsArgument: true, ArgumentPlaceholder: "event", SortPriority: "20"},

		{Name: "class", NeedsValue: true, SortPriority: "30"},
		{Name: "style", NeedsValue: true, SortPriority: "31"},

		{Name: "ref", NeedsValue: true, SortPriority: "40"},
		{Name: "key", NeedsValue: true, SortPriority: "41"},
		{Name: "slot", NeedsValue: true, SortPriority: "42"},
		{Name: "context", NeedsValue: true, SortPriority: "43"},
		{Name: "timeline", NeedsValue: false, NeedsArgument: true, ArgumentPlaceholder: "hidden", SortPriority: "44"},

		{Name: "format", NeedsValue: true, SortPriority: "50"},
		{Name: "format-decimal", NeedsValue: true, SortPriority: "51"},
		{Name: "format-money", NeedsValue: true, SortPriority: "52"},
		{Name: "format-date", NeedsValue: true, SortPriority: "53"},
		{Name: "format-decimal-precision", NeedsValue: true, SortPriority: "54"},
		{Name: "format-money-symbol", NeedsValue: true, SortPriority: "55"},
		{Name: "format-date-layout", NeedsValue: true, SortPriority: "56"},

		{Name: "scaffold", NeedsValue: false, SortPriority: "90"},
	}
)

// GetCompletions returns completion suggestions for the given position.
// It uses the AnalysisMap to access the SymbolTable at that location and
// provides context-aware suggestions based on what the user is typing.
//
// Takes position (protocol.Position) which specifies the cursor location.
//
// Returns *protocol.CompletionList which contains the completion suggestions.
// Returns error when completion analysis fails.
func (d *document) GetCompletions(ctx context.Context, position protocol.Position) (*protocol.CompletionList, error) {
	if d.isPKCFile() {
		return d.getPKCCompletions(position)
	}

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil || d.AnalysisMap == nil {
		return &protocol.CompletionList{
			IsIncomplete: false,
			Items:        []protocol.CompletionItem{},
		}, nil
	}

	completionCtx := analyseCompletionContext(d, position)

	switch completionCtx.TriggerKind {
	case triggerMemberAccess:
		return d.getMemberCompletions(ctx, position, completionCtx.BaseExpression, completionCtx.Prefix)

	case triggerDirective:
		return d.getDirectiveCompletions(completionCtx.Prefix)

	case triggerPartialAlias:
		return d.getPartialAliasCompletions(completionCtx.Prefix)

	case triggerDirectiveValue:
		return d.getScopeCompletionsWithPrefix(position, completionCtx.Prefix)

	case triggerEventHandler:
		return d.getEventHandlerCompletions(completionCtx.Prefix)

	case triggerPartialName:
		return d.getPartialNameCompletions(completionCtx.Prefix)

	case triggerRefAccess:
		return d.getRefCompletions(completionCtx.Prefix)

	case triggerStateAccessJS:
		return d.getStateFieldCompletionsJS(completionCtx.Prefix)

	case triggerPropsAccessJS:
		return d.getPropsFieldCompletionsJS(completionCtx.Prefix)

	case triggerPikoNamespace:
		return d.getPikoNamespaceCompletions(completionCtx.Prefix)

	case triggerPikoSubNamespace:
		return d.getPikoSubNamespaceCompletions(completionCtx.Namespace, completionCtx.Prefix)

	case triggerActionNamespace:
		return d.getActionNamespaceCompletions(completionCtx.Prefix)

	case triggerCSSClassValue:
		return d.getCSSClassCompletions(completionCtx.Prefix)

	default:
		return d.getScopeCompletions(position)
	}
}

// getScopeCompletions returns all symbols visible at the given position.
//
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *protocol.CompletionList which contains the available symbols.
// Returns error when the scope lookup fails.
func (d *document) getScopeCompletions(position protocol.Position) (*protocol.CompletionList, error) {
	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil {
		return &protocol.CompletionList{
			IsIncomplete: false,
			Items:        []protocol.CompletionItem{},
		}, nil
	}

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil || analysisCtx.Symbols == nil {
		return &protocol.CompletionList{
			IsIncomplete: false,
			Items:        []protocol.CompletionItem{},
		}, nil
	}

	symbolNames := analysisCtx.Symbols.AllSymbolNames()

	items := make([]protocol.CompletionItem, 0, len(symbolNames))
	for _, name := range symbolNames {
		items = append(items, buildSymbolCompletionItem(name, analysisCtx.Symbols))
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getScopeCompletionsWithPrefix returns symbols visible at the position,
// filtered by the given prefix.
//
// Takes position (protocol.Position) which specifies the cursor position.
// Takes prefix (string) which filters completions to names starting with this
// text (case-insensitive).
//
// Returns *protocol.CompletionList which contains the matching symbols.
// Returns error when the scope lookup fails.
func (d *document) getScopeCompletionsWithPrefix(position protocol.Position, prefix string) (*protocol.CompletionList, error) {
	if d.AnnotationResult == nil {
		return emptyCompletionList(), nil
	}
	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil {
		return emptyCompletionList(), nil
	}

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil || analysisCtx.Symbols == nil {
		return emptyCompletionList(), nil
	}

	symbolNames := analysisCtx.Symbols.AllSymbolNames()
	prefixLower := toLower(prefix)

	items := make([]protocol.CompletionItem, 0, len(symbolNames))
	for _, name := range symbolNames {
		if prefix != "" && !hasPrefix(toLower(name), prefixLower) {
			continue
		}
		items = append(items, buildSymbolCompletionItem(name, analysisCtx.Symbols))
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getMemberCompletions provides completions for member access (e.g.,
// state.user.<here>). It resolves the base expression's type and provides all
// accessible fields and methods that match the given prefix.
//
// Takes position (protocol.Position) which specifies the cursor position.
// Takes baseExpr (string) which is the expression before the dot.
// Takes prefix (string) which filters completions to those starting with this
// text (case-insensitive).
//
// Returns *protocol.CompletionList which contains the matching members.
// Returns error when resolution fails.
func (d *document) getMemberCompletions(ctx context.Context, position protocol.Position, baseExpr string, prefix string) (*protocol.CompletionList, error) {
	if !d.hasCompletionPrerequisites() {
		return emptyCompletionList(), nil
	}

	if position.Character == 0 {
		return emptyCompletionList(), nil
	}

	baseAnnotation := d.resolveBaseAnnotation(ctx, position, baseExpr)
	if baseAnnotation == nil || baseAnnotation.ResolvedType == nil || baseAnnotation.ResolvedType.TypeExpression == nil {
		return emptyCompletionList(), nil
	}

	analysisCtx := d.getCompletionContext(position)
	if analysisCtx == nil {
		return emptyCompletionList(), nil
	}

	namedType := d.resolveToNamedType(baseAnnotation, analysisCtx)
	if namedType == nil {
		return emptyCompletionList(), nil
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        buildMemberCompletionItems(namedType, prefix),
	}, nil
}

// hasCompletionPrerequisites checks if the document has the necessary data
// for completions.
//
// Returns bool which is true when all required fields are present.
func (d *document) hasCompletionPrerequisites() bool {
	return d.AnnotationResult != nil &&
		d.AnnotationResult.AnnotatedAST != nil &&
		d.AnalysisMap != nil &&
		d.TypeInspector != nil
}

// resolveBaseAnnotation finds the Go annotation for a base expression.
// It first tries AST lookup, then falls back to text-based resolution.
//
// Takes position (protocol.Position) which specifies the cursor position.
// Takes baseExpr (string) which is the base expression text to resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// information, or nil if resolution fails.
func (d *document) resolveBaseAnnotation(ctx context.Context, position protocol.Position, baseExpr string) *ast_domain.GoGeneratorAnnotation {
	_, l := logger_domain.From(ctx, log)

	posBeforeDot := protocol.Position{
		Line:      position.Line,
		Character: position.Character - 1,
	}

	if baseExprNode, _ := findExpressionAtPosition(ctx, d.AnnotationResult.AnnotatedAST, posBeforeDot, d.URI.Filename()); baseExprNode != nil {
		if ann := baseExprNode.GetGoAnnotation(); ann != nil && ann.ResolvedType != nil {
			return ann
		}
	}

	l.Trace("Completion: AST lookup failed, using text-based resolution",
		logger_domain.String("baseExpr", baseExpr))

	resolvedType := d.resolveExpressionFromText(ctx, baseExpr, position)
	if resolvedType == nil {
		return nil
	}

	return &ast_domain.GoGeneratorAnnotation{ResolvedType: resolvedType}
}

// getCompletionContext returns the analysis context for completion at the
// given position.
//
// Takes position (protocol.Position) which specifies the cursor location.
//
// Returns *annotator_domain.AnalysisContext which provides the analysis
// context for the node at the position, or nil if no node is found.
func (d *document) getCompletionContext(position protocol.Position) *annotator_domain.AnalysisContext {
	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil {
		return nil
	}

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil {
		return nil
	}
	return analysisCtx
}

// resolveToNamedType converts a base annotation to a named type with full
// metadata.
//
// Takes baseAnnotation (*ast_domain.GoGeneratorAnnotation) which contains the
// type to resolve.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides the
// current analysis context, including source paths.
//
// Returns *inspector_dto.Type which is the resolved named type, or nil if
// resolution fails.
func (d *document) resolveToNamedType(
	baseAnnotation *ast_domain.GoGeneratorAnnotation,
	analysisCtx *annotator_domain.AnalysisContext,
) *inspector_dto.Type {
	resolvedType := d.TypeInspector.ResolveToUnderlyingAST(
		baseAnnotation.ResolvedType.TypeExpression,
		analysisCtx.CurrentGoSourcePath,
	)

	namedType, _ := d.TypeInspector.ResolveExprToNamedType(
		resolvedType,
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)
	if namedType != nil {
		return namedType
	}

	return resolveTypeViaCanonicalPath(d.TypeInspector, baseAnnotation.ResolvedType)
}

// resolveTypeViaCanonicalPath resolves a type using the canonical package
// path stored in the annotation.
//
// Takes ti (TypeInspectorPort) which provides type resolution.
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which contains the
// canonical package path from the original type resolution.
//
// Returns *inspector_dto.Type which is the resolved type, or nil if lookup
// fails.
func resolveTypeViaCanonicalPath(ti TypeInspectorPort, resolvedType *ast_domain.ResolvedTypeInfo) *inspector_dto.Type {
	if resolvedType == nil || resolvedType.CanonicalPackagePath == "" {
		return nil
	}

	typeName, _, ok := inspector_domain.DeconstructTypeExpr(resolvedType.TypeExpression)
	if !ok {
		return nil
	}

	directExpr := &goast.Ident{Name: typeName}
	namedType, _ := ti.ResolveExprToNamedType(
		directExpr,
		resolvedType.CanonicalPackagePath,
		"",
	)
	return namedType
}

// directiveCompletionInfo holds completion details for a directive.
type directiveCompletionInfo struct {
	// Name is the directive name without the "p-" prefix (e.g. "if", "bind").
	Name string

	// ArgumentPlaceholder is the placeholder text for the argument (e.g., "attr",
	// "event").
	ArgumentPlaceholder string

	// SortPriority controls the order in the completion list; lower values
	// appear higher.
	SortPriority string

	// NeedsValue indicates whether the directive requires a value (="").
	NeedsValue bool

	// NeedsArgument indicates whether the directive requires an argument (e.g.,
	// p-bind:attr).
	NeedsArgument bool
}

// getDirectiveCompletions provides completions for Piko directives (p-<here>).
// It returns all available directives with proper snippets that auto-insert
// the attribute value with cursor positioning.
//
// Takes prefix (string) which filters directives to those starting with this
// text (case-insensitive). Empty string means no filtering.
//
// Returns *protocol.CompletionList which contains matching Piko directives.
// Returns error when completion generation fails.
func (*document) getDirectiveCompletions(prefix string) (*protocol.CompletionList, error) {
	return getStaticDirectiveCompletions(prefix), nil
}

// getPartialAliasCompletions provides completions for partial component
// aliases (is="<here>"). It gets all available partial imports from the
// current component's VirtualModule and offers them as completion options.
//
// Takes prefix (string) which filters the partial aliases to match.
//
// Returns *protocol.CompletionList which contains the matching completions.
// Returns error when the completion list cannot be built.
func (d *document) getPartialAliasCompletions(prefix string) (*protocol.CompletionList, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return emptyCompletionList(), nil
	}

	currentComponent := d.findCurrentComponent()
	if currentComponent == nil || currentComponent.Source == nil {
		return emptyCompletionList(), nil
	}

	items := buildPartialImportItems(currentComponent, prefix)
	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// findCurrentComponent locates the VirtualComponent for this document.
//
// Returns *annotator_dto.VirtualComponent which matches the document's file
// path, or nil if no match is found.
func (d *document) findCurrentComponent() *annotator_dto.VirtualComponent {
	if d.ProjectResult == nil || d.ProjectResult.VirtualModule == nil || d.ProjectResult.VirtualModule.Graph == nil {
		return nil
	}

	filePath, err := uriToPath(d.URI)
	if err != nil {
		return nil
	}

	for _, comp := range d.ProjectResult.VirtualModule.ComponentsByHash {
		if comp.Source.SourcePath == filePath {
			return comp
		}
	}
	return nil
}

// getEventHandlerCompletions provides completions for p-on:*="" handlers.
// It extracts exported functions from the TypeScript/JavaScript script block
// and includes special placeholder variables ($event, $form).
//
// Takes prefix (string) which filters completions to those containing this
// substring.
//
// Returns *protocol.CompletionList which contains matching exported functions
// and placeholder variables.
// Returns error when extraction fails.
func (d *document) getEventHandlerCompletions(prefix string) (*protocol.CompletionList, error) {
	items := make([]protocol.CompletionItem, 0)

	items = append(items, d.getEventPlaceholderCompletions(prefix)...)

	functions := d.extractClientScriptFunctions()
	for _, function := range functions {
		if prefix != "" && !containsSubstring(function.Name, prefix) {
			continue
		}

		detail := "Function"
		if function.Exported {
			detail = "Exported function"
		}

		items = append(items, protocol.CompletionItem{
			Label:  function.Name,
			Kind:   protocol.CompletionItemKindFunction,
			Detail: detail,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: detail + " from script block",
			},
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getEventPlaceholderCompletions returns completion items for $event and $form.
// These are special placeholder variables available in event handlers.
//
// Takes prefix (string) which filters completions to those matching this
// prefix.
//
// Returns []protocol.CompletionItem which contains matching placeholder
// variables.
func (*document) getEventPlaceholderCompletions(prefix string) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(eventPlaceholders))
	for _, p := range eventPlaceholders {
		if prefix != "" && !containsSubstring(p.name, prefix) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:      p.name,
			Kind:       protocol.CompletionItemKindVariable,
			Detail:     p.detail,
			SortText:   "0" + p.name,
			InsertText: p.name,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: p.documentation,
			},
		})
	}
	return items
}

// getPartialNameCompletions provides completions for reloadPartial() and
// reloadGroup() calls.
//
// Takes prefix (string) which filters partial names by matching the text.
//
// Returns *protocol.CompletionList which contains the names of partials
// available in the current component.
// Returns error when completion generation fails.
func (d *document) getPartialNameCompletions(prefix string) (*protocol.CompletionList, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return emptyCompletionList(), nil
	}

	currentComponent := d.findCurrentComponent()
	if currentComponent == nil {
		return emptyCompletionList(), nil
	}

	items := make([]protocol.CompletionItem, 0)
	for _, imp := range currentComponent.Source.PikoImports {
		if prefix != "" && !containsSubstring(imp.Alias, prefix) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:  imp.Alias,
			Kind:   protocol.CompletionItemKindModule,
			Detail: "Partial: " + imp.Path,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: "Reload partial component from " + imp.Path,
			},
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getRefCompletions provides completions for refs.* access in JavaScript.
// It extracts all p-ref attribute values from the template.
//
// Takes prefix (string) which filters results to names that contain this text.
//
// Returns *protocol.CompletionList which contains matching element references.
// Returns error when extraction fails.
func (d *document) getRefCompletions(prefix string) (*protocol.CompletionList, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return emptyCompletionList(), nil
	}

	refNames := extractPRefNames(d.AnnotationResult.AnnotatedAST, d.URI.Filename())

	items := make([]protocol.CompletionItem, 0, len(refNames))
	for _, name := range refNames {
		if prefix != "" && !containsSubstring(name, prefix) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:  name,
			Kind:   protocol.CompletionItemKindField,
			Detail: "Element reference",
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: "DOM element with p-ref=\"" + name + "\"",
			},
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getStateFieldCompletionsJS provides completions for state.* access in
// JavaScript script blocks. It resolves the RenderReturnTypeExpression to get the
// available state fields.
//
// Takes prefix (string) which filters completions to those containing the
// given substring.
//
// Returns *protocol.CompletionList which contains matching state field
// completions.
// Returns error when resolution fails.
func (d *document) getStateFieldCompletionsJS(prefix string) (*protocol.CompletionList, error) {
	comp := d.findCurrentComponent()
	if comp == nil || comp.Source == nil || comp.Source.Script == nil || comp.Source.Script.RenderReturnTypeExpression == nil {
		return emptyCompletionList(), nil
	}

	return d.getFieldCompletionsJS(
		comp,
		comp.Source.Script.RenderReturnTypeExpression,
		prefix,
		formatStateFieldDoc,
	)
}

// getPropsFieldCompletionsJS provides completions for props.*
// access in JavaScript script blocks. It resolves the
// PropsTypeExpression to get the available prop fields.
//
// Takes prefix (string) which filters completions to those containing the
// given substring.
//
// Returns *protocol.CompletionList which contains matching prop field
// completions.
// Returns error when resolution fails.
func (d *document) getPropsFieldCompletionsJS(prefix string) (*protocol.CompletionList, error) {
	comp := d.findCurrentComponent()
	if comp == nil || comp.Source == nil || comp.Source.Script == nil || comp.Source.Script.PropsTypeExpression == nil {
		return emptyCompletionList(), nil
	}

	return d.getFieldCompletionsJS(
		comp,
		comp.Source.Script.PropsTypeExpression,
		prefix,
		formatPropsFieldDoc,
	)
}

// fieldDocFormatter formats documentation for a field completion item.
type fieldDocFormatter func(field *inspector_dto.Field, tsType string) string

// getFieldCompletionsJS provides completions for field access in JavaScript
// script blocks. It resolves the type expression and builds completion items.
//
// Takes comp (*annotator_dto.VirtualComponent) which is the component context.
// Takes typeExpr (goast.Expr) which is the Go AST expression to resolve.
// Takes prefix (string) which filters completions.
// Takes docFormatter (fieldDocFormatter) which formats the documentation.
//
// Returns *protocol.CompletionList which contains matching field completions.
// Returns error when resolution fails.
func (d *document) getFieldCompletionsJS(
	comp *annotator_dto.VirtualComponent,
	typeExpr goast.Expr,
	prefix string,
	docFormatter fieldDocFormatter,
) (*protocol.CompletionList, error) {
	if d.TypeInspector == nil {
		return emptyCompletionList(), nil
	}

	namedType, _ := d.TypeInspector.ResolveExprToNamedType(
		typeExpr,
		comp.CanonicalGoPackagePath,
		comp.VirtualGoFilePath,
	)

	if namedType == nil || len(namedType.Fields) == 0 {
		return emptyCompletionList(), nil
	}

	tsGen := newTypeScriptGenerator()
	items := make([]protocol.CompletionItem, 0, len(namedType.Fields))

	for _, field := range namedType.Fields {
		if field.IsEmbedded {
			continue
		}

		if prefix != "" && !containsSubstring(field.Name, prefix) {
			continue
		}

		tsType := tsGen.goTypeToTypeScript(field)

		items = append(items, protocol.CompletionItem{
			Label:  field.Name,
			Kind:   protocol.CompletionItemKindField,
			Detail: tsType,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: docFormatter(field, tsType),
			},
		})
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// getPikoNamespaceCompletions returns completions for the piko.* namespace.
//
// Takes prefix (string) which filters completions to those matching the prefix.
//
// Returns *protocol.CompletionList which contains piko namespace completions.
// Returns error when completion fails.
func (*document) getPikoNamespaceCompletions(prefix string) (*protocol.CompletionList, error) {
	namespaces := []struct {
		Name   string
		Detail string
	}{
		{"nav", "Navigation helpers (reload, navigate, etc.)"},
		{"form", "Form utilities (submit, reset, etc.)"},
		{"session", "Session management"},
		{"toast", "Toast notifications"},
		{"modal", "Modal dialogs"},
		{"clipboard", "Clipboard operations"},
	}

	var items []protocol.CompletionItem
	for _, namespace := range namespaces {
		if prefix == "" || strings.HasPrefix(namespace.Name, prefix) {
			items = append(items, protocol.CompletionItem{
				Label:      namespace.Name,
				Kind:       protocol.CompletionItemKindModule,
				Detail:     namespace.Detail,
				InsertText: namespace.Name,
			})
		}
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getPikoSubNamespaceCompletions returns completions for piko sub-namespaces.
//
// Takes namespace (string) which is the sub-namespace (e.g., "nav", "form").
// Takes prefix (string) which filters completions to those matching the prefix.
//
// Returns *protocol.CompletionList which contains sub-namespace completions.
// Returns error when completion fails.
func (*document) getPikoSubNamespaceCompletions(namespace, prefix string) (*protocol.CompletionList, error) {
	var functions []struct {
		Name   string
		Detail string
	}

	switch namespace {
	case "nav":
		functions = []struct {
			Name   string
			Detail string
		}{
			{"reload", "Reload the current page"},
			{"navigate", "Navigate to a new URL"},
			{"back", "Go back in browser history"},
			{"forward", "Go forward in browser history"},
		}
	case "form":
		functions = []struct {
			Name   string
			Detail string
		}{
			{"submit", "Submit a form"},
			{"reset", "Reset form fields"},
			{"validate", "Validate form fields"},
		}
	case "toast":
		functions = []struct {
			Name   string
			Detail string
		}{
			{"success", "Show success toast"},
			{"error", "Show error toast"},
			{"info", "Show info toast"},
			{"warning", "Show warning toast"},
		}
	}

	var items []protocol.CompletionItem
	for _, function := range functions {
		if prefix == "" || strings.HasPrefix(function.Name, prefix) {
			items = append(items, protocol.CompletionItem{
				Label:      function.Name,
				Kind:       protocol.CompletionItemKindFunction,
				Detail:     function.Detail,
				InsertText: function.Name + "()",
			})
		}
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getActionNamespaceCompletions returns hierarchical completions for the
// action.* namespace based on the prefix depth.
//
// The prefix determines the completion level:
//   - No dots (e.g. "" or "em"): show unique namespace groups as modules
//   - One dot with namespace (e.g. "email." or "email.Con"): show actions
//     within that namespace as functions
//
// Takes prefix (string) which filters completions to those matching the prefix.
//
// Returns *protocol.CompletionList which contains action completions.
// Returns error when completion fails.
func (d *document) getActionNamespaceCompletions(prefix string) (*protocol.CompletionList, error) {
	if d.ProjectResult == nil || d.ProjectResult.VirtualModule == nil {
		return &protocol.CompletionList{IsIncomplete: false, Items: []protocol.CompletionItem{}}, nil
	}

	manifest := d.ProjectResult.VirtualModule.ActionManifest
	if manifest == nil || len(manifest.Actions) == 0 {
		return &protocol.CompletionList{IsIncomplete: false, Items: []protocol.CompletionItem{}}, nil
	}

	namespace, actionPrefix, hasNamespace := parseActionCompletionPrefix(prefix)
	if hasNamespace {
		return d.getActionFunctionCompletions(manifest, namespace, actionPrefix)
	}

	return d.getActionNamespaceGroupCompletions(manifest, prefix)
}

// getActionNamespaceGroupCompletions returns completions for the top-level
// namespace groups (e.g. "email", "user") filtered by the prefix.
//
// Takes manifest (*annotator_dto.ActionManifest) which provides the actions.
// Takes filter (string) which filters namespace names.
//
// Returns *protocol.CompletionList which contains namespace module completions.
// Returns error which is always nil.
func (*document) getActionNamespaceGroupCompletions(
	manifest *annotator_dto.ActionManifest,
	filter string,
) (*protocol.CompletionList, error) {
	seen := make(map[string]int)
	for i := range manifest.Actions {
		action := &manifest.Actions[i]
		namespace := extractNamespaceFromActionName(action.Name)
		if filter != "" && !strings.HasPrefix(namespace, filter) {
			continue
		}
		seen[namespace]++
	}

	items := make([]protocol.CompletionItem, 0, len(seen))
	for namespace, count := range seen {
		detail := fmt.Sprintf("Action package (%d action(s))", count)
		items = append(items, protocol.CompletionItem{
			Label:      namespace,
			Kind:       protocol.CompletionItemKindModule,
			Detail:     detail,
			InsertText: namespace,
		})
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getActionFunctionCompletions returns completions for actions within a
// specific namespace, filtered by the action name prefix.
//
// Takes manifest (*annotator_dto.ActionManifest) which provides the actions.
// Takes namespace (string) which is the namespace to search within.
// Takes filter (string) which filters action names.
//
// Returns *protocol.CompletionList which contains action function completions.
// Returns error which is always nil.
func (*document) getActionFunctionCompletions(
	manifest *annotator_dto.ActionManifest,
	namespace, filter string,
) (*protocol.CompletionList, error) {
	nsPrefix := namespace + "."
	items := make([]protocol.CompletionItem, 0)

	for i := range manifest.Actions {
		action := &manifest.Actions[i]
		if !strings.HasPrefix(action.Name, nsPrefix) {
			continue
		}

		actionName := action.Name[len(nsPrefix):]
		if filter != "" && !strings.HasPrefix(actionName, filter) {
			continue
		}

		detail := buildActionCompletionSignature(action)

		items = append(items, protocol.CompletionItem{
			Label:            actionName,
			Kind:             protocol.CompletionItemKindFunction,
			Detail:           detail,
			InsertText:       actionName + "($1)",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: action.Description,
			},
		})
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// emptyCompletionList returns an empty completion list.
//
// Returns *protocol.CompletionList which has no items and is marked as
// complete.
func emptyCompletionList() *protocol.CompletionList {
	return &protocol.CompletionList{IsIncomplete: false, Items: []protocol.CompletionItem{}}
}

// buildSymbolCompletionItem creates a completion item for a symbol. For
// built-in functions, it adds brackets as a snippet and uses Function kind.
//
// Takes name (string) which is the symbol name to look up.
// Takes symbols (*annotator_domain.SymbolTable) which holds the symbol data.
//
// Returns protocol.CompletionItem set up for the symbol type.
func buildSymbolCompletionItem(name string, symbols *annotator_domain.SymbolTable) protocol.CompletionItem {
	symbol, found := symbols.Find(name)
	if !found {
		return protocol.CompletionItem{
			Label: name,
			Kind:  protocol.CompletionItemKindVariable,
		}
	}

	if isBuiltInFunction(symbol) {
		insertText := name + "($1)$0"
		return protocol.CompletionItem{
			Label:            name,
			Kind:             protocol.CompletionItemKindFunction,
			InsertText:       insertText,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		}
	}

	return protocol.CompletionItem{
		Label: name,
		Kind:  protocol.CompletionItemKindVariable,
	}
}

// isBuiltInFunction reports whether a symbol represents a built-in function.
//
// Takes symbol (annotator_domain.Symbol) which is the symbol to check.
//
// Returns bool which is true if the symbol is a built-in function.
func isBuiltInFunction(symbol annotator_domain.Symbol) bool {
	if symbol.TypeInfo == nil || symbol.TypeInfo.TypeExpression == nil {
		return false
	}
	identifier, ok := symbol.TypeInfo.TypeExpression.(*goast.Ident)
	return ok && identifier.Name == typeBuiltInFunction
}

// buildMemberCompletionItems builds completion items from a named type's
// fields and methods, filtered by the given prefix.
//
// Takes namedType (*inspector_dto.Type) which provides the fields and methods
// to turn into completion items.
// Takes prefix (string) which filters results to names starting with this text
// (case-insensitive). An empty string means no filtering.
//
// Returns []protocol.CompletionItem which contains one item for each matching
// field and method of the named type.
func buildMemberCompletionItems(namedType *inspector_dto.Type, prefix string) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(namedType.Fields)+len(namedType.Methods))
	prefixLower := toLower(prefix)

	for _, field := range namedType.Fields {
		if prefix != "" && !hasPrefix(toLower(field.Name), prefixLower) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:  field.Name,
			Kind:   protocol.CompletionItemKindField,
			Detail: field.TypeString,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: "Field of type " + field.TypeString,
			},
		})
	}

	for _, method := range namedType.Methods {
		if prefix != "" && !hasPrefix(toLower(method.Name), prefixLower) {
			continue
		}
		detail := method.Signature.ToSignatureString()
		insertText := method.Name + "($1)$0"
		items = append(items, protocol.CompletionItem{
			Label:            method.Name,
			Kind:             protocol.CompletionItemKindMethod,
			Detail:           detail,
			InsertText:       insertText,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: "Method with signature: " + detail,
			},
		})
	}

	return items
}

// hasPrefix checks whether s starts with the given prefix.
//
// Takes s (string) which is the string to check.
// Takes prefix (string) which is the prefix to look for.
//
// Returns bool which is true if s starts with prefix, false otherwise.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// getStaticDirectiveCompletions returns directive completions without needing
// a document. The fast path uses this because directive completions are fixed.
//
// Takes prefix (string) which filters directives to those starting with this
// text (case-insensitive). An empty string means no filtering.
//
// Returns *protocol.CompletionList which contains matching Piko directives.
func getStaticDirectiveCompletions(prefix string) *protocol.CompletionList {
	items := make([]protocol.CompletionItem, 0, len(directiveCompletions))
	prefixLower := toLower(prefix)

	for _, directive := range directiveCompletions {
		if prefix != "" && !hasPrefix(toLower(directive.Name), prefixLower) {
			continue
		}
		item := buildDirectiveCompletionItem(directive)
		items = append(items, item)
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

// buildDirectiveCompletionItem creates a completion item for a directive.
//
// Takes directive (directiveCompletionInfo) which describes the
// directive to build.
//
// Returns protocol.CompletionItem which is the formatted completion item ready
// for use in the editor.
func buildDirectiveCompletionItem(directive directiveCompletionInfo) protocol.CompletionItem {
	fullName := "p-" + directive.Name
	var insertText string
	var insertFormat protocol.InsertTextFormat

	if directive.NeedsArgument {
		insertText = directive.Name + ":${1:" + directive.ArgumentPlaceholder + "}=\"$2\"$0"
		insertFormat = protocol.InsertTextFormatSnippet
	} else if directive.NeedsValue {
		insertText = directive.Name + "=\"$1\"$0"
		insertFormat = protocol.InsertTextFormatSnippet
	} else {
		insertText = directive.Name
		insertFormat = protocol.InsertTextFormatPlainText
	}

	var docContent string
	if directiveDocumentation, exists := pikoDirectiveDocumentations[fullName]; exists {
		docContent = directiveDocumentation.Description
		if directiveDocumentation.Syntax != "" {
			docContent += "\n\n**Syntax:** `" + directiveDocumentation.Syntax + "`"
		}
		if directiveDocumentation.Example != "" {
			docContent += "\n\n**Example:**\n```html\n" + directiveDocumentation.Example + "\n```"
		}
	} else {
		docContent = "Piko directive"
	}

	return protocol.CompletionItem{
		Label:            directive.Name,
		Kind:             protocol.CompletionItemKindProperty,
		Detail:           fullName,
		InsertText:       insertText,
		InsertTextFormat: insertFormat,
		FilterText:       directive.Name,
		SortText:         directive.SortPriority,
		Documentation: &protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: docContent,
		},
	}
}

// buildPartialImportItems creates completion items from a component's partial
// imports.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the source
// component containing partial imports.
// Takes prefix (string) which filters imports to those containing this
// substring.
//
// Returns []protocol.CompletionItem which contains completion suggestions for
// matching partial imports.
func buildPartialImportItems(comp *annotator_dto.VirtualComponent, prefix string) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(comp.Source.PikoImports))

	for _, imp := range comp.Source.PikoImports {
		if prefix != "" && !containsSubstring(imp.Alias, prefix) {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:  imp.Alias,
			Kind:   protocol.CompletionItemKindModule,
			Detail: imp.Path,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: "Partial component from " + imp.Path,
			},
		})
	}

	return items
}

// containsSubstring checks whether s contains substr, ignoring letter case.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns bool which is true if substr is found within s.
func containsSubstring(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

// toLower converts a string to lowercase using simple ASCII rules.
//
// Takes s (string) which is the input string to convert.
//
// Returns string which is the lowercase version of the input.
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// contains reports whether substr is within s.
//
// Takes s (string) which is the string to search within.
// Takes substr (string) which is the substring to find.
//
// Returns bool which is true if substr is found, false otherwise.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
}

// indexOfSubstring finds the position of a substring within a string.
//
// Takes s (string) which is the string to search in.
// Takes substr (string) which is the substring to find.
//
// Returns int which is the index of the first match, or -1 if not found.
func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := range len(substr) {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// extractPRefNames walks the AST to find all p-ref attribute values.
//
// Takes tree (*TemplateAST) which is the parsed template to search.
// Takes docPath (string) which filters nodes to only those from this file.
//
// Returns []string which contains the unique p-ref names found, in order.
func extractPRefNames(tree *ast_domain.TemplateAST, docPath string) []string {
	refNames := make([]string, 0)
	seen := make(map[string]bool)

	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil {
			if *node.GoAnnotations.OriginalSourcePath != docPath {
				return true
			}
		}

		if node.DirRef != nil && node.DirRef.RawExpression != "" {
			name := node.DirRef.RawExpression
			if !seen[name] {
				seen[name] = true
				refNames = append(refNames, name)
			}
		}

		return true
	})

	return refNames
}

// formatStateFieldDoc creates markdown documentation for a state field.
//
// Takes field (*inspector_dto.Field) which provides the field metadata.
// Takes tsType (string) which specifies the TypeScript type.
//
// Returns string which contains the formatted markdown documentation.
func formatStateFieldDoc(field *inspector_dto.Field, tsType string) string {
	documentation := "**State field**\n\n"
	documentation += "```typescript\n"
	documentation += "state." + field.Name + ": " + tsType + "\n"
	documentation += "```\n"
	if field.TypeString != "" && field.TypeString != tsType {
		documentation += "\nGo type: `" + field.TypeString + "`"
	}
	return documentation
}

// formatPropsFieldDoc creates markdown text for a props field.
//
// Takes field (*inspector_dto.Field) which provides the field details.
// Takes tsType (string) which specifies the TypeScript type.
//
// Returns string which contains the formatted markdown text.
func formatPropsFieldDoc(field *inspector_dto.Field, tsType string) string {
	documentation := "**Props field**\n\n"
	documentation += "```typescript\n"
	documentation += "props." + field.Name + ": " + tsType + "\n"
	documentation += "```\n"
	if field.TypeString != "" && field.TypeString != tsType {
		documentation += "\nGo type: `" + field.TypeString + "`"
	}
	return documentation
}

// parseActionCompletionPrefix splits the prefix into namespace and action
// filter parts. If the prefix contains a dot, the part before the dot is the
// namespace and the part after is the action filter.
//
// Takes prefix (string) which is the text after "action.".
//
// Returns namespace (string) which is the part before the dot, or empty.
// Returns actionFilter (string) which is the part after the dot, or the full
// prefix if no dot is present.
// Returns hasNamespace (bool) which is true when a dot separator was found.
func parseActionCompletionPrefix(prefix string) (namespace, actionFilter string, hasNamespace bool) {
	if namespace, filter, found := strings.Cut(prefix, "."); found {
		return namespace, filter, true
	}
	return "", prefix, false
}

// buildActionCompletionSignature constructs a TypeScript function signature
// for display in the completion item detail.
//
// Takes action (*annotator_dto.ActionDefinition) which provides the action
// definition containing type information.
//
// Returns string which is the formatted TypeScript signature.
func buildActionCompletionSignature(action *annotator_dto.ActionDefinition) string {
	params := buildActionParamList(action.CallParams, false)

	var outputType string
	if action.OutputType != nil {
		outputType = action.OutputType.TSType
		if outputType == "" {
			outputType = action.OutputType.Name
		}
	}

	if outputType != "" {
		return action.TSFunctionName + "(" + params + "): ActionBuilder<" + outputType + ">"
	}
	return action.TSFunctionName + "(" + params + "): ActionBuilder<void>"
}
