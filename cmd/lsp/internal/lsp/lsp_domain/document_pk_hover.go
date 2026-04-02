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
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// PKHoverContext holds the context needed to show hover information for a
// PK element, including its kind, name, and location in the source file.
type PKHoverContext struct {
	// Name is the identifier of the PK element being hovered over.
	Name string

	// Kind specifies the type of PK definition: handler, partial, or reference.
	Kind PKDefinitionKind

	// Range specifies the text span to highlight for the hover.
	Range protocol.Range

	// Position is the cursor location in the document.
	Position protocol.Position
}

// GetPKHoverInfo returns hover information for PK-specific constructs.
// This handles p-on:click="handler" to show handler function signatures,
// reloadPartial('name') to show partial information, and refs.refName to show
// element information.
//
// Takes position (protocol.Position) which specifies the cursor position to query.
//
// Returns *protocol.Hover which contains the hover content, or nil if no
// PK construct is found at the position.
// Returns error when hover information cannot be retrieved.
func (d *document) GetPKHoverInfo(ctx context.Context, position protocol.Position) (*protocol.Hover, error) {
	_, l := logger_domain.From(ctx, log)

	if len(d.Content) == 0 {
		return nil, nil
	}

	hoverCtx := d.analysePKHoverContext(position)
	if hoverCtx == nil || hoverCtx.Kind == PKDefUnknown {
		return nil, nil
	}

	l.Debug("GetPKHoverInfo: Found PK context",
		logger_domain.String("kind", hoverCtx.kindString()),
		logger_domain.String("name", hoverCtx.Name))

	switch hoverCtx.Kind {
	case PKDefHandler:
		if d.isPKCFile() {
			return d.getPKCHandlerHover(hoverCtx)
		}
		return d.getHandlerHover(hoverCtx)
	case PKDefPartial:
		return d.getPartialHover(hoverCtx, false)
	case PKDefPartialTag:
		return d.getPartialHover(hoverCtx, true)
	case PKDefRef:
		return d.getRefHover(hoverCtx)
	case PKDefPikoElement:
		return d.getPikoElementHover(hoverCtx)
	case PKDefDirective:
		return d.getDirectiveHover(hoverCtx)
	case PKDefBuiltinFunction:
		return d.getBuiltinHover(hoverCtx)
	case PKDefTemplateTag:
		return d.getTemplateTagHover(hoverCtx)
	case PKDefActionRoot, PKDefActionNamespace, PKDefActionName:
		return d.getActionHover(hoverCtx)
	case PKDefActionParamKey:
		return d.getActionParamKeyHover(hoverCtx)
	case PKDefPKCStateProperty:
		return d.getPKCStatePropertyHover(hoverCtx)
	default:
		return nil, nil
	}
}

// kindString returns the readable name for the hover context kind.
//
// Returns string which is the human-readable name for the context kind.
func (ctx *PKHoverContext) kindString() string {
	switch ctx.Kind {
	case PKDefHandler:
		return "handler"
	case PKDefPartial:
		return "partial"
	case PKDefPartialTag:
		return "partial-tag"
	case PKDefRef:
		return "ref"
	case PKDefPikoElement:
		return "piko-element"
	case PKDefDirective:
		return "directive"
	case PKDefBuiltinFunction:
		return "builtin"
	case PKDefTemplateTag:
		return "template-tag"
	case PKDefActionRoot:
		return "action-root"
	case PKDefActionNamespace:
		return "action-namespace"
	case PKDefActionName:
		return "action-name"
	case PKDefActionParamKey:
		return "action-param-key"
	case PKDefPKCStateProperty:
		return "pkc-state-property"
	default:
		return "unknown"
	}
}

// analysePKHoverContext determines the PK hover context at the cursor
// position.
//
// Takes position (protocol.Position) which specifies the cursor
// location to analyse.
//
// Returns *PKHoverContext which describes the hover context, or nil if no
// context applies.
func (d *document) analysePKHoverContext(position protocol.Position) *PKHoverContext {
	lines := strings.Split(string(d.Content), newlineChar)
	if int(position.Line) >= len(lines) {
		return nil
	}

	line := lines[position.Line]
	cursor := int(position.Character)

	if ctx := d.checkTemplateTagHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkPikoPartialTagHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkPikoElementHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkDirectiveHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkBuiltinHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkActionParamKeyHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkActionHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkHandlerHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkPartialHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkIsAttributeHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkRefHoverContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if d.isPKCFile() {
		if ctx := d.checkPKCStatePropertyHoverContext(line, cursor, position); ctx != nil {
			return ctx
		}
	}

	return nil
}

// checkIsAttributeHoverContext checks if cursor is on an is="..." attribute
// for hover. This handles the partial alias attribute that triggers partial
// invocation.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context if the cursor is on an
// is="..." attribute, or nil if no match.
func (*document) checkIsAttributeHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	patterns := []string{`is="`, `is='`}

	for _, pattern := range patterns {
		index := strings.LastIndex(line[:min(cursor+20, len(line))], pattern)
		if index == -1 || index > cursor {
			continue
		}

		quoteChar := pattern[len(pattern)-1]
		startPosition := index + len(pattern)
		endPosition := findQuoteEndPosition(line, startPosition, quoteChar)

		if endPosition == -1 {
			continue
		}

		if cursor < index || cursor > endPosition+1 {
			continue
		}

		partialName := line[startPosition:endPosition]
		if partialName == "" {
			continue
		}

		return &PKHoverContext{
			Kind:     PKDefPartial,
			Name:     partialName,
			Position: position,
			Range: protocol.Range{
				Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(index)},
				End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(endPosition + 1)},
			},
		}
	}

	return nil
}

var (
	// handlerHoverPatterns are the directive patterns to search for handler hover
	// context.
	handlerHoverPatterns = []string{
		`p-on:click="`, `p-on:change="`, `p-on:submit="`, `p-on:input="`,
		`p-on:focus="`, `p-on:blur="`, `p-on:keydown="`, `p-on:keyup="`,
	}

	// partialHoverPatterns are the patterns to search for partial hover context.
	partialHoverPatterns = []string{`reloadPartial('`, `reloadPartial("`, `partial('`, `partial("`}
)

// checkHandlerHoverContext checks if the cursor is on a handler name for hover.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character offset within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context when the cursor is on
// a handler name, or nil when no handler pattern matches.
func (*document) checkHandlerHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	for _, pattern := range handlerHoverPatterns {
		if ctx := tryExtractHandlerHoverContext(line, cursor, position, pattern); ctx != nil {
			return ctx
		}
	}
	return nil
}

// checkPartialHoverContext checks if cursor is on a partial name for hover.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context if a partial name
// pattern matches at the cursor position, or nil if no pattern matches.
func (*document) checkPartialHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	for _, pattern := range partialHoverPatterns {
		if ctx := tryExtractPartialHoverContext(line, cursor, position, pattern); ctx != nil {
			return ctx
		}
	}
	return nil
}

// checkRefHoverContext checks if the cursor is on a ref name for hover.
//
// Takes line (string) which contains the text to search for a ref pattern.
// Takes cursor (int) which gives the cursor position within the line.
// Takes position (protocol.Position) which gives the document position context.
//
// Returns *PKHoverContext which holds the ref hover context, or nil if the
// cursor is not on a valid ref name.
func (*document) checkRefHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	name, startPosition, endPosition, ok := scanIdentifierAfterPrefix(line, cursor, "refs.", patternSearchRadius)
	if !ok {
		return nil
	}

	return &PKHoverContext{
		Kind:     PKDefRef,
		Name:     name,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(startPosition)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(endPosition)},
		},
	}
}

// getHandlerHover returns hover information for an event handler.
//
// Takes ctx (*PKHoverContext) which provides the hover request context.
//
// Returns *protocol.Hover which contains the hover information to display.
// Returns error when parsing fails.
func (d *document) getHandlerHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	sfcResult := d.getSFCResult()
	if sfcResult == nil {
		return nil, nil
	}

	clientScript, found := sfcResult.ClientScript()
	if !found || clientScript.Content == "" {
		return d.makeSimpleHover(ctx, "Event handler (no client script)")
	}

	parseLog := logger.NewDeferLog(logger.DeferLogAll, nil)

	tree, ok := js_parser.Parse(
		parseLog,
		logger.Source{
			Index:          0,
			KeyPath:        logger.Path{Text: d.URI.Filename()},
			PrettyPaths:    logger.PrettyPaths{Rel: d.URI.Filename(), Abs: d.URI.Filename()},
			Contents:       clientScript.Content,
			IdentifierName: d.URI.Filename(),
		},
		js_parser.OptionsFromConfig(&config.Options{
			TS: config.TSOptions{Parse: true},
		}),
	)

	if !ok {
		return d.makeSimpleHover(ctx, "Event handler")
	}

	signature := d.findFunctionSignature(&tree, ctx.Name)
	if signature != "" {
		return d.makeCodeHover(ctx, signature, "typescript")
	}

	return d.makeSimpleHover(ctx, fmt.Sprintf("Event handler `%s`", ctx.Name))
}

// findFunctionSignature extracts the signature of a function from the AST.
//
// Takes tree (*js_ast.AST) which contains the parsed syntax tree to search.
// Takes functionName (string) which specifies the name of the function to find.
//
// Returns string which contains the function signature, or empty if not found.
func (d *document) findFunctionSignature(tree *js_ast.AST, functionName string) string {
	for i := range tree.Parts {
		part := &tree.Parts[i]
		for _, statement := range part.Stmts {
			sig := d.extractFunctionSignature(statement, tree.Symbols, functionName)
			if sig != "" {
				return sig
			}
		}
	}
	return ""
}

// extractFunctionSignature gets the function signature from a statement.
//
// Takes statement (js_ast.Stmt) which is the statement to extract from.
// Takes symbols ([]ast.Symbol) which provides symbol information.
// Takes functionName (string) which is the name of the function.
//
// Returns string which is the extracted signature, or empty if not found.
func (d *document) extractFunctionSignature(statement js_ast.Stmt, symbols []ast.Symbol, functionName string) string {
	switch s := statement.Data.(type) {
	case *js_ast.SFunction:
		return d.extractRegularFunctionSignature(s, symbols, functionName)
	case *js_ast.SLocal:
		return d.extractLocalFunctionSignature(s, symbols, functionName)
	case *js_ast.SExportDefault:
		return d.extractDefaultFunctionSignature(s, symbols, functionName)
	default:
		return ""
	}
}

// extractRegularFunctionSignature extracts the signature from a regular
// function declaration.
//
// Takes s (*js_ast.SFunction) which provides the function statement to extract.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
// Takes functionName (string) which specifies the function name to match.
//
// Returns string which contains the formatted signature, or empty if the
// function has no name or the name does not match.
func (d *document) extractRegularFunctionSignature(s *js_ast.SFunction, symbols []ast.Symbol, functionName string) string {
	if s.Fn.Name == nil {
		return ""
	}
	name := getSymbolName(s.Fn.Name.Ref, symbols)
	if name != functionName {
		return ""
	}
	return d.formatFunctionSignature(name, s.Fn, s.IsExport)
}

// extractLocalFunctionSignature extracts the signature from an exported local
// declaration.
//
// Takes s (*js_ast.SLocal) which provides the local declaration to check.
// Takes symbols ([]ast.Symbol) which provides symbol data for binding names.
// Takes functionName (string) which specifies the function name to match.
//
// Returns string which is the formatted arrow signature, or empty if not found
// or not exported.
func (d *document) extractLocalFunctionSignature(s *js_ast.SLocal, symbols []ast.Symbol, functionName string) string {
	if !s.IsExport {
		return ""
	}
	for _, declaration := range s.Decls {
		name := extractBindingName(declaration.Binding, symbols)
		if name == functionName {
			return d.formatArrowSignature(name, declaration.ValueOrNil)
		}
	}
	return ""
}

// extractDefaultFunctionSignature gets the signature from a default export
// function.
//
// Takes s (*js_ast.SExportDefault) which is the default export statement.
// Takes symbols ([]ast.Symbol) which provides the symbol table for name lookup.
// Takes functionName (string) which is the expected function name to match.
//
// Returns string which is the formatted signature, or empty if no match is
// found.
func (d *document) extractDefaultFunctionSignature(s *js_ast.SExportDefault, symbols []ast.Symbol, functionName string) string {
	fnStmt, ok := s.Value.Data.(*js_ast.SFunction)
	if !ok || fnStmt.Fn.Name == nil {
		return ""
	}
	name := getSymbolName(fnStmt.Fn.Name.Ref, symbols)
	if name != functionName {
		return ""
	}
	return d.formatFunctionSignature(name, fnStmt.Fn, true)
}

// formatFunctionSignature formats a function declaration signature.
//
// Takes name (string) which specifies the function name.
// Takes jsFunction (js_ast.Fn) which provides the function AST node to format.
// Takes isExport (bool) which indicates whether to include the export keyword.
//
// Returns string which contains the formatted function signature.
func (*document) formatFunctionSignature(name string, jsFunction js_ast.Fn, isExport bool) string {
	var b strings.Builder

	if isExport {
		b.WriteString("export ")
	}

	if jsFunction.IsAsync {
		b.WriteString("async ")
	}

	b.WriteString("function ")
	b.WriteString(name)
	b.WriteString("(")

	params := make([]string, 0, len(jsFunction.Args))
	for i := range jsFunction.Args {
		params = append(params, fmt.Sprintf("argument%d", i))
	}
	b.WriteString(strings.Join(params, ", "))
	b.WriteString(")")

	return b.String()
}

// formatArrowSignature formats an arrow function as a TypeScript signature.
//
// Takes name (string) which is the export constant name.
// Takes value (js_ast.Expr) which is the expression to format.
//
// Returns string which is the formatted TypeScript signature.
func (*document) formatArrowSignature(name string, value js_ast.Expr) string {
	var b strings.Builder
	b.WriteString("export const ")
	b.WriteString(name)

	if arrow, ok := value.Data.(*js_ast.EArrow); ok {
		b.WriteString(" = (")
		params := make([]string, 0, len(arrow.Args))
		for range arrow.Args {
			params = append(params, "...")
		}
		b.WriteString(strings.Join(params, ", "))
		b.WriteString(") => ")
		if arrow.IsAsync {
			b.WriteString("Promise<void>")
		} else {
			b.WriteString("void")
		}
	} else {
		b.WriteString(": Function")
	}

	return b.String()
}

// getPartialHover returns hover information for a partial.
//
// Takes ctx (*PKHoverContext) which provides the hover context including the
// partial name and range.
// Takes showProps (bool) which controls whether to show properties in the
// hover content.
//
// Returns *protocol.Hover which contains the formatted hover content.
// Returns error when the simple hover cannot be created.
func (d *document) getPartialHover(ctx *PKHoverContext, showProps bool) (*protocol.Hover, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return d.makeSimpleHover(ctx, fmt.Sprintf("Partial `%s`", ctx.Name))
	}

	currentComponent := d.findCurrentComponent()
	if currentComponent == nil {
		return d.makeSimpleHover(ctx, fmt.Sprintf("Partial `%s`", ctx.Name))
	}

	imp := d.findPikoImportByAlias(currentComponent, ctx.Name)
	if imp == nil {
		return d.makeSimpleHover(ctx, fmt.Sprintf("Partial `%s`", ctx.Name))
	}

	content := d.buildPartialHoverContent(imp.Path, showProps)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: content,
		},
		Range: &ctx.Range,
	}, nil
}

// findPikoImportByAlias finds a Piko import by its alias name.
//
// Takes component (*annotator_dto.VirtualComponent) which provides the source
// containing the Piko imports to search.
// Takes alias (string) which is the alias name to match.
//
// Returns *annotator_dto.PikoImport which is the matching import, or nil if
// no import with the given alias exists.
func (*document) findPikoImportByAlias(component *annotator_dto.VirtualComponent, alias string) *annotator_dto.PikoImport {
	for i := range component.Source.PikoImports {
		if component.Source.PikoImports[i].Alias == alias {
			return &component.Source.PikoImports[i]
		}
	}
	return nil
}

// buildPartialHoverContent builds the markdown content for a partial hover.
//
// Takes importPath (string) which specifies the partial component to document.
// Takes showProps (bool) which controls whether to include the props table.
//
// Returns string which contains the formatted markdown hover content.
func (d *document) buildPartialHoverContent(importPath string, showProps bool) string {
	var b strings.Builder

	vc := d.findVirtualComponentByImportPath(importPath)

	partialName := extractPartialName(importPath)
	_, _ = fmt.Fprintf(&b, "## `<%s>`\n\n", partialName)

	b.WriteString("Imported partial component\n\n")
	_, _ = fmt.Fprintf(&b, "**Import:** `%s`\n\n", importPath)

	if vc == nil {
		return b.String()
	}

	if showProps {
		d.appendPropsTable(&b, vc)
	}
	appendComponentLinks(&b, vc)

	return b.String()
}

// appendPropsTable appends a Props table to the builder if the component has
// props.
//
// Takes b (*strings.Builder) which receives the formatted table output.
// Takes vc (*annotator_dto.VirtualComponent) which provides the component to
// extract props from.
func (d *document) appendPropsTable(b *strings.Builder, vc *annotator_dto.VirtualComponent) {
	props := d.extractPropsFromComponent(vc)
	if len(props) == 0 {
		return
	}

	b.WriteString("\n**Props:**\n\n")
	b.WriteString("| Name | Type | Required |\n")
	b.WriteString("|------|------|----------|\n")
	for _, prop := range props {
		required := ""
		if prop.isRequired {
			required = "yes"
		}
		_, _ = fmt.Fprintf(b, "| `%s` | `%s` | %s |\n", prop.name, prop.typeName, required)
	}
}

// propInfo holds details about a single prop for hover display.
type propInfo struct {
	// name is the property name in lowercase, from the struct tag or field name.
	name string

	// typeName is the Go type name shown in the properties table.
	typeName string

	// isRequired indicates whether the prop must be provided.
	isRequired bool
}

// extractPropsFromComponent extracts Props struct fields from a
// VirtualComponent.
//
// Takes vc (*annotator_dto.VirtualComponent) which is the component to inspect.
//
// Returns []propInfo which contains the extracted prop details, or nil if no
// Props struct is found.
func (*document) extractPropsFromComponent(vc *annotator_dto.VirtualComponent) []propInfo {
	propsStruct := findPropsStructInComponent(vc)
	if propsStruct == nil {
		return nil
	}

	return extractPropsFromStruct(propsStruct)
}

// findVirtualComponentByImportPath finds a VirtualComponent by its import path.
//
// Takes importPath (string) which is the Piko import path to search for.
//
// Returns *annotator_dto.VirtualComponent which is the matching component, or
// nil if not found.
func (d *document) findVirtualComponentByImportPath(importPath string) *annotator_dto.VirtualComponent {
	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return nil
	}

	for _, vc := range d.AnnotationResult.VirtualModule.ComponentsByHash {
		if vc.Source != nil && vc.Source.ModuleImportPath == importPath {
			return vc
		}
	}

	return nil
}

// checkTemplateTagHoverContext checks if the cursor is on a <template> or
// </template> tag for hover.
//
// Takes line (string) which is the current line text.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context when the cursor is on
// a template tag, or nil when no match is found.
func (*document) checkTemplateTagHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	for _, pattern := range []string{"<template", "</template"} {
		index := strings.Index(line, pattern)
		if index == -1 {
			continue
		}

		tagStart := index + 1
		if pattern[1] == '/' {
			tagStart = index + 2
		}
		tagEnd := tagStart + len("template")

		if cursor >= tagStart && cursor <= tagEnd {
			return &PKHoverContext{
				Kind:     PKDefTemplateTag,
				Name:     "template",
				Position: position,
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(tagStart)},
					End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(tagEnd)},
				},
			}
		}
	}

	return nil
}

// checkPikoPartialTagHoverContext checks if the cursor is on a <piko:partial>
// tag name for hover. This shows the partial-specific info including Props.
//
// Takes line (string) which is the current line text.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context when the cursor is on
// a piko:partial tag name, or nil when no match is found.
func (d *document) checkPikoPartialTagHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	for _, pattern := range []string{"<piko:partial", "</piko:partial"} {
		index := strings.Index(line, pattern)
		if index == -1 {
			continue
		}

		tagStart := index + 1
		if pattern[1] == '/' {
			tagStart = index + 2
		}
		tagEnd := tagStart + len("piko:partial")

		if cursor < tagStart || cursor > tagEnd {
			continue
		}

		partialName := d.extractIsAttributeValueMultiLine(position.Line)
		if partialName == "" {
			continue
		}

		return &PKHoverContext{
			Kind:     PKDefPartialTag,
			Name:     partialName,
			Position: position,
			Range: protocol.Range{
				Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(tagStart)},
				End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(tagEnd)},
			},
		}
	}

	return nil
}

// getTemplateTagHover returns hover information for a template tag.
//
// Takes ctx (*PKHoverContext) which provides the hover context.
//
// Returns *protocol.Hover which contains the formatted hover content with a
// link to the generated file.
// Returns error when the hover cannot be created.
func (d *document) getTemplateTagHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	var b strings.Builder
	b.WriteString("```html\n")
	b.WriteString("<template>\n")
	b.WriteString("```\n")
	b.WriteString("\nThe template block contains the component's HTML markup.\n")

	vc := d.findCurrentComponent()
	if vc != nil && vc.VirtualGoFilePath != "" {
		_, _ = fmt.Fprintf(&b, "\n[Open generated file](%s)\n", uri.File(vc.VirtualGoFilePath))
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: b.String(),
		},
		Range: &ctx.Range,
	}, nil
}

// getRefHover returns hover information for a ref element.
//
// Takes ctx (*PKHoverContext) which provides the ref name and range.
//
// Returns *protocol.Hover which contains the formatted hover content.
// Returns error when the ref information cannot be found.
func (d *document) getRefHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	elementInfo := d.findRefElementInfo(ctx.Name)

	var b strings.Builder
	b.WriteString("```typescript\n")
	_, _ = fmt.Fprintf(&b, "refs.%s: %s | null\n", ctx.Name, elementInfo.elementType)
	b.WriteString("```\n\n")

	if elementInfo.tagName != "" {
		_, _ = fmt.Fprintf(&b, "Element: `<%s>`\n", elementInfo.tagName)
	}

	_, _ = fmt.Fprintf(&b, "Access via `refs.%s`", ctx.Name)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: b.String(),
		},
		Range: &ctx.Range,
	}, nil
}

// refElementInfo holds details about where a reference points in the HTML.
type refElementInfo struct {
	// tagName is the HTML element tag name where the ref is defined.
	tagName string

	// elementType is the HTML element type name (e.g. "HTMLDivElement").
	elementType string
}

// findRefElementInfo finds details about the element with a given p-ref.
//
// Takes refName (string) which specifies the p-ref attribute value to search
// for.
//
// Returns refElementInfo which contains the tag name and element type for the
// matched element. Returns default values if no match is found.
func (d *document) findRefElementInfo(refName string) refElementInfo {
	info := refElementInfo{
		elementType: "HTMLElement",
	}

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return info
	}

	d.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.DirRef != nil && node.DirRef.RawExpression == refName {
			info.tagName = node.TagName
			info.elementType = tagNameToElementType(node.TagName)
			return false
		}
		return true
	})

	return info
}

// makeSimpleHover creates a hover with plain text content.
//
// Takes ctx (*PKHoverContext) which provides the position and range for the
// hover.
// Takes text (string) which is the plain text content to show.
//
// Returns *protocol.Hover which contains the hover content and position.
// Returns error which is always nil.
func (*document) makeSimpleHover(ctx *PKHoverContext, text string) (*protocol.Hover, error) {
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.PlainText,
			Value: text,
		},
		Range: &ctx.Range,
	}, nil
}

// makeCodeHover creates a hover response with a formatted code block.
//
// Takes ctx (*PKHoverContext) which provides the hover position and range.
// Takes code (string) which contains the code to display.
// Takes language (string) which sets the syntax highlighting language.
//
// Returns *protocol.Hover which contains the formatted code block.
// Returns error which is always nil.
func (*document) makeCodeHover(ctx *PKHoverContext, code, language string) (*protocol.Hover, error) {
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: fmt.Sprintf("```%s\n%s\n```", language, code),
		},
		Range: &ctx.Range,
	}, nil
}

// tryExtractHandlerHoverContext tries to extract a handler hover context for a
// single pattern.
//
// Takes line (string) which contains the text to search within.
// Takes cursor (int) which is the cursor position in the line.
// Takes position (protocol.Position) which gives the document position.
// Takes pattern (string) which is the handler pattern to match.
//
// Returns *PKHoverContext which holds the hover context if found, or nil if no
// valid handler context could be extracted.
func tryExtractHandlerHoverContext(line string, cursor int, position protocol.Position, pattern string) *PKHoverContext {
	index := strings.LastIndex(line[:min(cursor+20, len(line))], pattern)
	if index == -1 || index > cursor {
		return nil
	}

	if isCursorOnEventPlaceholder(line, cursor) {
		return nil
	}

	startPosition, endPosition := findHandlerNameBounds(line, index+len(pattern))
	if endPosition == -1 {
		return nil
	}

	if cursor < startPosition || cursor > startPosition+endPosition {
		return nil
	}

	handlerName := extractCleanHandlerName(line[startPosition : startPosition+endPosition])
	if handlerName == "" {
		return nil
	}

	return &PKHoverContext{
		Kind:     PKDefHandler,
		Name:     handlerName,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(startPosition)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(startPosition + len(handlerName))},
		},
	}
}

// isCursorOnEventPlaceholder checks if the cursor is on $event or $form.
// These special placeholders should get their own hover showing synthetic
// type info.
//
// Takes line (string) which is the line text to search.
// Takes cursor (int) which is the cursor position within the line.
//
// Returns bool which is true if the cursor is on $event or $form.
func isCursorOnEventPlaceholder(line string, cursor int) bool {
	placeholders := []string{"$event", "$form"}

	for _, placeholder := range placeholders {
		searchStart := max(0, cursor-len(placeholder))
		searchEnd := min(len(line), cursor+len(placeholder))

		segment := line[searchStart:searchEnd]
		placeholderIndex := strings.Index(segment, placeholder)
		if placeholderIndex == -1 {
			continue
		}

		absStart := searchStart + placeholderIndex
		absEnd := absStart + len(placeholder)

		if cursor >= absStart && cursor < absEnd {
			return true
		}
	}

	return false
}

// findHandlerNameBounds finds the start and end positions of a handler name
// in a line of text.
//
// Takes line (string) which contains the text to search.
// Takes startPosition (int) which specifies where to begin searching.
//
// Returns start (int) which is the start position (same as startPosition).
// Returns end (int) which is the end position, or -1 if no delimiter is found.
func findHandlerNameBounds(line string, startPosition int) (start int, end int) {
	endPosition := strings.Index(line[startPosition:], `"`)
	if endPosition == -1 {
		endPosition = strings.Index(line[startPosition:], `(`)
	}
	return startPosition, endPosition
}

// extractCleanHandlerName removes any trailing parentheses and whitespace
// from a handler name string.
//
// Takes raw (string) which is the handler name that may include parentheses.
//
// Returns string which is the cleaned handler name.
func extractCleanHandlerName(raw string) string {
	handlerName := raw
	if parenIndex := strings.Index(handlerName, "("); parenIndex != -1 {
		handlerName = handlerName[:parenIndex]
	}
	return strings.TrimSpace(handlerName)
}

// tryExtractPartialHoverContext tries to extract a partial hover context for
// a single pattern.
//
// Takes line (string) which contains the text to search within.
// Takes cursor (int) which specifies the cursor position in the line.
// Takes position (protocol.Position) which provides the document position.
// Takes pattern (string) which defines the pattern to match against.
//
// Returns *PKHoverContext which contains the extracted context, or nil if the
// pattern is not found or the cursor is outside the matched range.
func tryExtractPartialHoverContext(line string, cursor int, position protocol.Position, pattern string) *PKHoverContext {
	name, startPosition, endPosition, ok := tryExtractQuotedValue(line, cursor, pattern)
	if !ok {
		return nil
	}

	return &PKHoverContext{
		Kind:     PKDefPartial,
		Name:     name,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(startPosition)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(endPosition)},
		},
	}
}

// findQuoteEndPosition finds the position of the closing quote character in
// a line.
//
// Takes line (string) which contains the text to search.
// Takes startPosition (int) which specifies where to begin searching.
// Takes quoteChar (byte) which is the quote character to find.
//
// Returns int which is the position of the closing quote, or -1 if not found.
func findQuoteEndPosition(line string, startPosition int, quoteChar byte) int {
	for i := startPosition; i < len(line); i++ {
		if line[i] == quoteChar {
			return i
		}
	}
	return -1
}

// extractPartialName extracts the partial name from an import path.
// For example, "myapp/partials/status_badge.pk" returns "status_badge".
//
// Takes importPath (string) which is the full import path to extract from.
//
// Returns string which is the final path segment without the .pk or .pkc
// extension.
func extractPartialName(importPath string) string {
	name := strings.TrimSuffix(importPath, ".pk")
	name = strings.TrimSuffix(name, ".pkc")

	if index := strings.LastIndex(name, "/"); index >= 0 {
		name = name[index+1:]
	}

	return name
}

// appendComponentLinks appends source and generated file links to the builder.
//
// Takes b (*strings.Builder) which receives the formatted file links.
// Takes vc (*annotator_dto.VirtualComponent) which provides the source and
// generated file paths.
func appendComponentLinks(b *strings.Builder, vc *annotator_dto.VirtualComponent) {
	if vc.Source != nil && vc.Source.SourcePath != "" {
		_, _ = fmt.Fprintf(b, "\n[Open source file](%s)\n", uri.File(vc.Source.SourcePath))
	}
	if vc.VirtualGoFilePath != "" {
		_, _ = fmt.Fprintf(b, "\n[Open generated file](%s)\n", uri.File(vc.VirtualGoFilePath))
	}
}

// findPropsStructInComponent locates the Props struct type in a component.
//
// Takes vc (*annotator_dto.VirtualComponent) which is the component to search.
//
// Returns *goast.StructType which is the Props struct, or nil if not found.
func findPropsStructInComponent(vc *annotator_dto.VirtualComponent) *goast.StructType {
	if vc == nil || vc.Source == nil || vc.Source.Script == nil {
		return nil
	}

	if isPikoNoProps(vc.Source.Script.PropsTypeExpression) {
		return nil
	}

	astFile := getComponentAST(vc)
	if astFile == nil {
		return nil
	}

	return findPropsStructInAST(astFile)
}

// isPikoNoProps checks if a props expression is piko.NoProps.
//
// Takes propsExpr (goast.Expr) which is the expression to check.
//
// Returns bool which is true if the expression is nil or piko.NoProps.
func isPikoNoProps(propsExpr goast.Expr) bool {
	if propsExpr == nil {
		return true
	}

	selectorExpression, ok := propsExpr.(*goast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := selectorExpression.X.(*goast.Ident)
	return ok && x.Name == "piko" && selectorExpression.Sel.Name == "NoProps"
}

// getComponentAST returns the AST for a component, preferring the rewritten
// version.
//
// Takes vc (*annotator_dto.VirtualComponent) which provides the component to
// get the AST from.
//
// Returns *goast.File which is the rewritten AST if available, otherwise the
// original script AST, or nil if neither exists.
func getComponentAST(vc *annotator_dto.VirtualComponent) *goast.File {
	if vc.RewrittenScriptAST != nil {
		return vc.RewrittenScriptAST
	}
	if vc.Source.Script != nil {
		return vc.Source.Script.AST
	}
	return nil
}

// findPropsStructInAST searches an AST file for a struct type named "Props".
//
// Takes file (*goast.File) which is the parsed Go source file to search.
//
// Returns *goast.StructType which is the Props struct if found, or nil if not
// present.
func findPropsStructInAST(file *goast.File) *goast.StructType {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*goast.GenDecl)
		if !ok {
			continue
		}
		if st := findPropsStructInGenDecl(genDecl); st != nil {
			return st
		}
	}
	return nil
}

// findPropsStructInGenDecl searches a GenDecl for a struct type named Props.
//
// Takes genDecl (*goast.GenDecl) which is the declaration to search.
//
// Returns *goast.StructType which is the Props struct if found, or nil.
func findPropsStructInGenDecl(genDecl *goast.GenDecl) *goast.StructType {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*goast.TypeSpec)
		if !ok || typeSpec.Name.Name != "Props" {
			continue
		}
		if st, ok := typeSpec.Type.(*goast.StructType); ok {
			return st
		}
	}
	return nil
}

// extractPropsFromStruct extracts property details from a struct type.
//
// Takes propsStruct (*goast.StructType) which is the struct to extract from.
//
// Returns []propInfo which contains the extracted property details.
func extractPropsFromStruct(propsStruct *goast.StructType) []propInfo {
	if propsStruct.Fields == nil {
		return nil
	}

	props := make([]propInfo, 0, len(propsStruct.Fields.List))
	for _, field := range propsStruct.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		props = append(props, extractPropFromField(field))
	}
	return props
}

// extractPropFromField extracts property details from a single struct field.
//
// Takes field (*goast.Field) which is the struct field to extract from.
//
// Returns propInfo which holds the property name, type, and required status.
func extractPropFromField(field *goast.Field) propInfo {
	fieldName := field.Names[0].Name
	propName := strings.ToLower(fieldName)
	isRequired := false

	if field.Tag != nil {
		tagValue := strings.Trim(field.Tag.Value, "`")
		propName, isRequired = extractPropTagInfo(tagValue, propName)
	}

	return propInfo{
		name:       propName,
		typeName:   goastutil.ASTToTypeString(field.Type),
		isRequired: isRequired,
	}
}

// extractPropTagInfo extracts the property name and required status from struct
// tags.
//
// Takes tagValue (string) which contains the struct tag to parse.
// Takes defaultName (string) which is used when no property name is found.
//
// Returns propName (string) which is the extracted name or the default.
// Returns isRequired (bool) which is true if the validate tag has "required".
func extractPropTagInfo(tagValue, defaultName string) (propName string, isRequired bool) {
	propName = defaultName

	if pName := extractTagValue(tagValue, "prop"); pName != "" {
		if name := strings.Split(pName, ",")[0]; name != "" {
			propName = name
		}
	}

	if validate := extractTagValue(tagValue, "validate"); strings.Contains(validate, "required") {
		isRequired = true
	}

	return propName, isRequired
}

// extractTagValue extracts a value from a struct tag string.
//
// Takes tagString (string) which is the full struct tag string without backticks.
// Takes key (string) which is the tag key to find.
//
// Returns string which is the tag value, or empty if not found.
func extractTagValue(tagString, key string) string {
	pattern := key + `:"`
	index := strings.Index(tagString, pattern)
	if index == -1 {
		return ""
	}

	startPosition := index + len(pattern)
	for i := startPosition; i < len(tagString); i++ {
		if tagString[i] == '"' {
			return tagString[startPosition:i]
		}
	}

	return ""
}

// containsUnquotedTagClose checks if a line contains a > or /> that closes
// an HTML tag, not inside a quoted attribute value.
//
// Takes line (string) which is the text to check.
//
// Returns bool which is true if the line has an unquoted tag close.
func containsUnquotedTagClose(line string) bool {
	inDoubleQuote := false
	inSingleQuote := false

	for i := range len(line) {
		character := line[i]

		if character == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if character == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if character == '>' && !inDoubleQuote && !inSingleQuote {
			return true
		}
	}

	return false
}

// tagNameToElementType maps an HTML tag name to its TypeScript element type.
//
// Takes tagName (string) which is the HTML tag name to look up.
//
// Returns string which is the TypeScript element type for the tag.
func tagNameToElementType(tagName string) string {
	switch strings.ToLower(tagName) {
	case "input":
		return "HTMLInputElement"
	case "button":
		return "HTMLButtonElement"
	case "form":
		return "HTMLFormElement"
	case "a":
		return "HTMLAnchorElement"
	case "img":
		return "HTMLImageElement"
	case "select":
		return "HTMLSelectElement"
	case "textarea":
		return "HTMLTextAreaElement"
	case "canvas":
		return "HTMLCanvasElement"
	case "video":
		return "HTMLVideoElement"
	case "audio":
		return "HTMLAudioElement"
	case "table":
		return "HTMLTableElement"
	case "iframe":
		return "HTMLIFrameElement"
	case "div", "span", "p", "h1", "h2", "h3", "h4", "h5", "h6",
		"header", "footer", "main", "section", "article", "nav", "aside":
		return "HTMLElement"
	default:
		if strings.Contains(tagName, "-") {
			return tagName + "Element"
		}
		return "HTMLElement"
	}
}
