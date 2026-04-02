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
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_parser"
	"piko.sh/piko/internal/esbuild/logger"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// refsPrefixLen is the length of the "refs." prefix string.
	refsPrefixLen = 5

	// newlineChar is the newline character used to split document content into lines.
	newlineChar = "\n"

	// patternSearchRadius is the number of characters after the cursor to include
	// when searching for a pattern match.
	patternSearchRadius = 20

	// maxMultiLineSearchLinesNav is the maximum number of lines to search when
	// finding multi-line element attributes.
	maxMultiLineSearchLinesNav = 20
)

// PKDefinitionContext represents the context for a PK-specific definition lookup.
type PKDefinitionContext struct {
	// Name is the identifier being looked up.
	Name string

	// Kind is the type of definition being searched for.
	Kind PKDefinitionKind

	// Position is the cursor location in the document.
	Position protocol.Position
}

// PKDefinitionKind represents the kind of a PK definition.
type PKDefinitionKind int

const (
	// PKDefUnknown is a definition kind that could not be identified.
	PKDefUnknown PKDefinitionKind = iota

	// PKDefHandler is a handler reference in a p-on:click="handlerName" event.
	PKDefHandler

	// PKDefPartial represents a reloadPartial('name') partial reference.
	PKDefPartial

	// PKDefRef represents a refs.refName or p-ref="refName" reference.
	PKDefRef

	// PKDefPikoElement represents a piko:* built-in element tag.
	PKDefPikoElement

	// PKDefDirective represents a p-* directive attribute.
	PKDefDirective

	// PKDefBuiltinFunction represents a builtin function like len, cap, T, etc.
	PKDefBuiltinFunction

	// PKDefTemplateTag represents a <template> tag in an SFC file.
	PKDefTemplateTag

	// PKDefPartialFile represents a piko:partial tag where we want to go to the file.
	PKDefPartialFile

	// PKDefPartialTag represents hover on a piko:partial tag name (shows Props).
	PKDefPartialTag

	// PKDefActionRoot represents the cursor on the "action" keyword in
	// action.email.Contact($form).
	PKDefActionRoot

	// PKDefActionNamespace represents the cursor on a namespace segment like
	// "email" in action.email.Contact($form).
	PKDefActionNamespace

	// PKDefActionName represents the cursor on the action name like "Contact"
	// in action.email.Contact($form).
	PKDefActionName

	// PKDefActionParamKey represents the cursor on a key inside an action
	// call's object literal, e.g. "environment_id" in
	// action.ns.Name({environment_id: ...}).
	PKDefActionParamKey

	// PKDefCSSClass represents a CSS class name reference in a template
	// attribute such as class="foo", p-class:foo, or :class="'foo'".
	PKDefCSSClass

	// PKDefPKCStateProperty represents a state.propName reference in a PKC
	// template expression.
	PKDefPKCStateProperty
)

// GetPKDefinition finds a PK-specific definition at the cursor position.
//
// This handles:
//   - p-on:click="handler" -> Jump to handler function in client script
//   - reloadPartial('name') -> Jump to partial component
//   - refs.refName -> Jump to p-ref="refName" attribute
//
// Takes position (protocol.Position) which specifies the cursor position to check.
//
// Returns []protocol.Location which contains the definition locations found.
// Returns error when the definition lookup fails.
func (d *document) GetPKDefinition(ctx context.Context, position protocol.Position) ([]protocol.Location, error) {
	_, l := logger_domain.From(ctx, log)

	if len(d.Content) == 0 {
		return nil, nil
	}

	defCtx := d.analysePKDefinitionContext(position)
	if defCtx == nil || defCtx.Kind == PKDefUnknown {
		return nil, nil
	}

	l.Debug("GetPKDefinition: Found PK context",
		logger_domain.String("kind", defCtx.kindString()),
		logger_domain.String("name", defCtx.Name))

	switch defCtx.Kind {
	case PKDefHandler:
		if d.isPKCFile() {
			return d.findPKCHandlerDefinition(defCtx.Name)
		}
		return d.findHandlerDefinition(defCtx.Name)
	case PKDefPartial:
		return d.findPartialDefinitionByName(defCtx.Name)
	case PKDefPartialFile:
		return d.findPartialFileByName(ctx, defCtx.Name)
	case PKDefRef:
		if d.isPKCFile() {
			return d.findPKCRefDefinition(defCtx.Name)
		}
		return d.findRefDefinition(defCtx.Name)
	case PKDefActionName:
		return d.findActionDefinition(defCtx.Name)
	case PKDefCSSClass:
		return d.findCSSClassDefinitionLocation(defCtx.Name)
	case PKDefPKCStateProperty:
		return d.findPKCStatePropertyDefinition(defCtx.Name)
	default:
		return nil, nil
	}
}

// kindString returns a string representation of the definition kind.
//
// Returns string which is "handler", "partial", "ref", "piko-element",
// "directive", "template-tag", or "unknown".
func (ctx *PKDefinitionContext) kindString() string {
	switch ctx.Kind {
	case PKDefHandler:
		return "handler"
	case PKDefPartial:
		return "partial"
	case PKDefRef:
		return "ref"
	case PKDefPikoElement:
		return "piko-element"
	case PKDefDirective:
		return "directive"
	case PKDefTemplateTag:
		return "template-tag"
	case PKDefPartialFile:
		return "partial-file"
	case PKDefPartialTag:
		return "partial-tag"
	case PKDefActionRoot:
		return "action-root"
	case PKDefActionNamespace:
		return "action-namespace"
	case PKDefActionName:
		return "action-name"
	case PKDefCSSClass:
		return "css-class"
	case PKDefPKCStateProperty:
		return "pkc-state-property"
	default:
		return "unknown"
	}
}

// analysePKDefinitionContext finds the type of PK definition at the cursor.
//
// Takes position (protocol.Position) which specifies the cursor position to check.
//
// Returns *PKDefinitionContext which describes the definition context, or nil
// if the position is not on a known PK definition.
func (d *document) analysePKDefinitionContext(position protocol.Position) *PKDefinitionContext {
	lines := strings.Split(string(d.Content), newlineChar)
	if int(position.Line) >= len(lines) {
		return nil
	}

	line := lines[position.Line]
	cursor := int(position.Character)

	if ctx := d.checkActionDefinitionContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkEventHandlerContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkPartialReloadContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkIsAttributeDefinitionContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkPikoPartialTagDefinitionContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if ctx := d.checkRefsAccessContext(line, cursor, position); ctx != nil {
		return ctx
	}

	if d.isPKCFile() {
		if ctx := d.checkPKCStatePropertyContext(line, cursor, position); ctx != nil {
			return ctx
		}
	}

	return d.checkCSSClassDefinitionContext(line, cursor, position)
}

// checkIsAttributeDefinitionContext checks if cursor is on an is="..."
// attribute for go-to-definition.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKDefinitionContext which provides definition context if the
// cursor is on an is="..." attribute, or nil if no match.
func (*document) checkIsAttributeDefinitionContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	patterns := []string{`is="`, `is='`}

	for _, pattern := range patterns {
		if ctx := tryExtractIsAttributeContext(line, cursor, position, pattern); ctx != nil {
			return ctx
		}
	}

	return nil
}

// checkPikoPartialTagDefinitionContext checks if cursor is on a <piko:partial>
// tag name for go-to-definition.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKDefinitionContext which provides definition context if the cursor
// is on a piko:partial tag name, or nil if no match.
func (d *document) checkPikoPartialTagDefinitionContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	for _, pattern := range []string{"<piko:partial", "</piko:partial"} {
		index := strings.LastIndex(line, pattern)
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

		return &PKDefinitionContext{
			Kind:     PKDefPartialFile,
			Name:     partialName,
			Position: position,
		}
	}

	return nil
}

// extractIsAttributeValueMultiLine extracts the is="..." attribute value from
// an element, searching across multiple lines if needed.
//
// Takes startLine (uint32) which is the line number where the element starts.
//
// Returns string which is the attribute value, or empty string if not found.
func (d *document) extractIsAttributeValueMultiLine(startLine uint32) string {
	lines := strings.Split(string(d.Content), newlineChar)

	var elementText strings.Builder

	for i := int(startLine); i < len(lines) && i < int(startLine)+maxMultiLineSearchLinesNav; i++ {
		elementText.WriteString(lines[i])
		elementText.WriteString(" ")

		if containsUnquotedTagClose(lines[i]) {
			break
		}
	}

	return extractIsAttributeValue(elementText.String())
}

var (
	// eventHandlerPatterns defines the directive patterns to search for event
	// handler context.
	eventHandlerPatterns = []string{
		`p-on:click="`, `p-on:change="`, `p-on:submit="`, `p-on:input="`,
		`p-on:focus="`, `p-on:blur="`, `p-on:keydown="`, `p-on:keyup="`,
		`p-on:mouseenter="`, `p-on:mouseleave="`, `p-on:scroll="`,
	}

	// partialReloadPatterns are the patterns to search for partial reload context.
	partialReloadPatterns = []string{`reloadPartial('`, `reloadPartial("`, `partial('`, `partial("`}
)

// checkEventHandlerContext checks if the cursor is on a handler name in a
// p-on:*="handler" attribute.
//
// Takes line (string) which contains the text of the current line.
// Takes cursor (int) which specifies the cursor position within the line.
// Takes position (protocol.Position) which provides the LSP
// position in the document.
//
// Returns *PKDefinitionContext which contains the handler context, or nil if
// the cursor is not on an event handler.
func (*document) checkEventHandlerContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	for _, pattern := range eventHandlerPatterns {
		if ctx := tryExtractEventHandlerContext(line, cursor, position, pattern); ctx != nil {
			return ctx
		}
	}
	return nil
}

// checkPartialReloadContext checks if the cursor is on a partial name in
// reloadPartial().
//
// Takes line (string) which is the current line of text being analysed.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKDefinitionContext which provides the definition context if
// found, or nil if the cursor is not on a partial reload pattern.
func (*document) checkPartialReloadContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	for _, pattern := range partialReloadPatterns {
		if ctx := tryExtractPartialReloadContext(line, cursor, position, pattern); ctx != nil {
			return ctx
		}
	}
	return nil
}

// checkRefsAccessContext checks if the cursor is on a ref name in a
// refs.refName pattern.
//
// Takes line (string) which contains the text to search within.
// Takes cursor (int) which specifies the cursor position in the line.
// Takes position (protocol.Position) which provides the position for the result.
//
// Returns *PKDefinitionContext which contains the ref definition context, or
// nil if the cursor is not on a valid ref name.
func (*document) checkRefsAccessContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	name, _, _, ok := scanIdentifierAfterPrefix(line, cursor, "refs.", patternSearchRadius)
	if !ok {
		return nil
	}

	return &PKDefinitionContext{
		Kind:     PKDefRef,
		Name:     name,
		Position: position,
	}
}

// findHandlerDefinition finds the definition of a handler function in the
// client script. Uses the shared function extraction from
// extractPKCFunctionsFromAST for consistent handling of regular functions,
// arrow functions, function expressions, and default exports.
//
// Takes handlerName (string) which specifies the name of the handler to find.
//
// Returns []protocol.Location which contains the location of the handler in
// the source code.
// Returns error when the definition cannot be found.
func (d *document) findHandlerDefinition(handlerName string) ([]protocol.Location, error) {
	sfcResult := d.getSFCResult()
	if sfcResult == nil {
		return nil, nil
	}

	clientScript, found := sfcResult.ClientScript()
	if !found || clientScript.Content == "" {
		return nil, nil
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
		parserOptions(),
	)

	if !ok {
		return nil, nil
	}

	baseLineOffset := clientScript.ContentLocation.Line - 1
	baseColOffset := clientScript.ContentLocation.Column - 1

	meta := &pkcMetadata{
		Functions: make(map[string]*pkcFunction),
	}
	extractPKCFunctionsFromAST(&tree, clientScript.Content, baseLineOffset, baseColOffset, meta)

	function, exists := meta.Functions[handlerName]
	if !exists {
		return nil, nil
	}

	return d.pkcSymbolLocation(function.Line, function.Column, len(function.Name)), nil
}

// findPartialDefinitionByName finds where a partial component is defined.
// It first looks for the import line in the script block, then falls back to
// opening the partial file itself.
//
// Takes partialName (string) which specifies the partial component to find.
//
// Returns []protocol.Location which contains the definition location.
// Returns error when the search fails.
func (d *document) findPartialDefinitionByName(partialName string) ([]protocol.Location, error) {
	if location := d.findPartialImportLine(partialName); location != nil {
		return []protocol.Location{*location}, nil
	}

	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return nil, nil
	}

	currentComponent := d.findCurrentComponent()
	if currentComponent == nil {
		return nil, nil
	}

	for _, imp := range currentComponent.Source.PikoImports {
		if imp.Alias == partialName {
			for _, vc := range d.AnnotationResult.VirtualModule.ComponentsByHash {
				if vc.Source != nil && strings.HasSuffix(vc.Source.SourcePath, imp.Path+".pk") {
					return []protocol.Location{{
						URI:   protocol.DocumentURI("file://" + vc.Source.SourcePath),
						Range: protocol.Range{},
					}}, nil
				}
			}
		}
	}

	return nil, nil
}

// findPartialFileByName finds the source file of a partial component by name.
// Unlike findPartialDefinitionByName, this returns the partial file itself,
// not the import line.
//
// Takes partialName (string) which specifies the name of the partial to find.
//
// Returns []protocol.Location which contains the partial file location.
// Returns error when the lookup fails.
func (d *document) findPartialFileByName(ctx context.Context, partialName string) ([]protocol.Location, error) {
	_, l := logger_domain.From(ctx, log)

	if d.AnnotationResult == nil || d.AnnotationResult.VirtualModule == nil {
		return nil, nil
	}

	currentComponent := d.findCurrentComponent()
	if currentComponent == nil {
		return nil, nil
	}

	for _, imp := range currentComponent.Source.PikoImports {
		if imp.Alias == partialName {
			for _, vc := range d.AnnotationResult.VirtualModule.ComponentsByHash {
				if vc.Source != nil && vc.Source.ModuleImportPath == imp.Path {
					l.Debug("findPartialFileByName: Found partial file",
						logger_domain.String("alias", partialName),
						logger_domain.String("path", vc.Source.SourcePath))
					return []protocol.Location{{
						URI:   protocol.DocumentURI("file://" + vc.Source.SourcePath),
						Range: protocol.Range{},
					}}, nil
				}
			}
		}
	}

	l.Debug("findPartialFileByName: Partial not found", logger_domain.String("alias", partialName))
	return nil, nil
}

// findPartialImportLine finds the line where a partial is imported.
//
// Takes partialName (string) which is the alias of the partial to find.
//
// Returns *protocol.Location which points to the import line, or nil if not
// found.
func (d *document) findPartialImportLine(partialName string) *protocol.Location {
	content := string(d.Content)
	lines := strings.Split(content, newlineChar)

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, partialName) {
			continue
		}

		afterAlias := trimmed[len(partialName):]
		if len(afterAlias) == 0 {
			continue
		}

		afterAlias = strings.TrimLeft(afterAlias, " \t")
		if len(afterAlias) > 0 && (afterAlias[0] == '"' || afterAlias[0] == '`') {
			column := strings.Index(line, partialName)
			return &protocol.Location{
				URI: d.URI,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      safeconv.IntToUint32(lineNum),
						Character: safeconv.IntToUint32(column),
					},
					End: protocol.Position{
						Line:      safeconv.IntToUint32(lineNum),
						Character: safeconv.IntToUint32(column + len(partialName)),
					},
				},
			}
		}
	}

	return nil
}

// findRefDefinition finds where a p-ref attribute is defined in the template.
//
// Takes refName (string) which specifies the reference name to find.
//
// Returns []protocol.Location which contains the definition location, or nil
// if not found.
// Returns error when the lookup fails.
func (d *document) findRefDefinition(refName string) ([]protocol.Location, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return nil, nil
	}

	var location *protocol.Location

	d.AnnotationResult.AnnotatedAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.DirRef != nil && node.DirRef.RawExpression == refName {
			if node.Location.Line > 0 {
				location = &protocol.Location{
					URI: d.URI,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      safeconv.IntToUint32(node.Location.Line - 1),
							Character: safeconv.IntToUint32(node.Location.Column - 1),
						},
						End: protocol.Position{
							Line:      safeconv.IntToUint32(node.Location.Line - 1),
							Character: safeconv.IntToUint32(node.Location.Column - 1 + len(node.TagName) + 1),
						},
					},
				}
				return false
			}
		}
		return true
	})

	if location != nil {
		return []protocol.Location{*location}, nil
	}

	return nil, nil
}

// GetPKReferences finds all references to a PK symbol (handler, partial,
// ref).
//
// Takes position (protocol.Position) which specifies the cursor
// position to analyse.
//
// Returns []protocol.Location which contains all locations referencing the
// symbol at the given position.
// Returns error when the reference search fails.
func (d *document) GetPKReferences(ctx context.Context, position protocol.Position) ([]protocol.Location, error) {
	_, l := logger_domain.From(ctx, log)

	if len(d.Content) == 0 {
		return nil, nil
	}

	defCtx := d.analysePKDefinitionContext(position)
	if defCtx == nil || defCtx.Kind == PKDefUnknown {
		return nil, nil
	}

	l.Debug("GetPKReferences: Found PK context",
		logger_domain.String("kind", defCtx.kindString()),
		logger_domain.String("name", defCtx.Name))

	switch defCtx.Kind {
	case PKDefHandler:
		return d.findHandlerReferences(defCtx.Name)
	case PKDefPartial:
		return d.findPartialReferences(defCtx.Name)
	case PKDefRef:
		return d.findRefReferences(defCtx.Name)
	default:
		return nil, nil
	}
}

// findHandlerReferences finds all p-on:* attributes that use a handler.
//
// Takes handlerName (string) which is the handler name to search for.
//
// Returns []protocol.Location which contains the locations of all references.
// Returns error when the search fails.
func (d *document) findHandlerReferences(handlerName string) ([]protocol.Location, error) {
	locations := make([]protocol.Location, 0)
	content := string(d.Content)
	lines := strings.Split(content, newlineChar)

	for lineNum, line := range lines {
		if !strings.Contains(line, handlerName) {
			continue
		}

		patterns := []string{`p-on:click="`, `p-on:change="`, `p-on:submit="`, `p-on:input="`}
		for _, pattern := range patterns {
			index := strings.Index(line, pattern)
			if index == -1 {
				continue
			}

			afterPattern := line[index+len(pattern):]
			if strings.HasPrefix(afterPattern, handlerName) {
				column := index + len(pattern)
				locations = append(locations, protocol.Location{
					URI: d.URI,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      safeconv.IntToUint32(lineNum),
							Character: safeconv.IntToUint32(column),
						},
						End: protocol.Position{
							Line:      safeconv.IntToUint32(lineNum),
							Character: safeconv.IntToUint32(column + len(handlerName)),
						},
					},
				})
			}
		}
	}

	return locations, nil
}

// findPartialReferences finds all reloadPartial and partial calls that
// reference the given partial name.
//
// Takes partialName (string) which is the name of the partial to search for.
//
// Returns []protocol.Location which contains the locations of all references.
// Returns error when the search fails.
func (d *document) findPartialReferences(partialName string) ([]protocol.Location, error) {
	locations := make([]protocol.Location, 0)
	content := string(d.Content)
	lines := strings.Split(content, newlineChar)

	patterns := []string{
		`reloadPartial('` + partialName + `'`,
		`reloadPartial("` + partialName + `"`,
		`partial('` + partialName + `'`,
		`partial("` + partialName + `"`,
	}

	for lineNum, line := range lines {
		for _, pattern := range patterns {
			index := strings.Index(line, pattern)
			if index != -1 {
				nameIndex := strings.Index(pattern, partialName)
				column := index + nameIndex

				locations = append(locations, protocol.Location{
					URI: d.URI,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      safeconv.IntToUint32(lineNum),
							Character: safeconv.IntToUint32(column),
						},
						End: protocol.Position{
							Line:      safeconv.IntToUint32(lineNum),
							Character: safeconv.IntToUint32(column + len(partialName)),
						},
					},
				})
			}
		}
	}

	return locations, nil
}

// findRefReferences finds all refs.refName usages in the document.
//
// Takes refName (string) which specifies the reference name to search for.
//
// Returns []protocol.Location which contains all locations where the
// reference is used.
// Returns error when the search fails.
func (d *document) findRefReferences(refName string) ([]protocol.Location, error) {
	locations := make([]protocol.Location, 0)
	content := string(d.Content)
	lines := strings.Split(content, newlineChar)

	pattern := "refs." + refName

	for lineNum, line := range lines {
		d.findRefReferencesInLine(line, lineNum, pattern, refName, &locations)
	}

	return locations, nil
}

// findRefReferencesInLine finds all ref references in a single line of text.
//
// Takes line (string) which is the text content to search.
// Takes lineNum (int) which is the line number for reporting locations.
// Takes pattern (string) which is the ref pattern to match.
// Takes refName (string) which is the reference name for the location.
// Takes locations (*[]protocol.Location) which collects the found locations.
func (d *document) findRefReferencesInLine(line string, lineNum int, pattern, refName string, locations *[]protocol.Location) {
	index := strings.Index(line, pattern)
	for index != -1 {
		endIndex := index + len(pattern)

		if isPartOfLongerIdentifier(line, endIndex) {
			index = findNextOccurrence(line, pattern, endIndex)
			continue
		}

		column := index + refsPrefixLen
		*locations = append(*locations, d.createRefLocation(lineNum, column, refName))

		index = findNextOccurrence(line, pattern, endIndex)
	}
}

// createRefLocation creates a protocol.Location for a ref reference.
//
// Takes lineNum (int) which specifies the line number in the document.
// Takes column (int) which specifies the column position on the line.
// Takes refName (string) which provides the reference name to determine the
// end position.
//
// Returns protocol.Location which represents the location spanning the
// reference name.
func (d *document) createRefLocation(lineNum, column int, refName string) protocol.Location {
	return protocol.Location{
		URI: d.URI,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(lineNum),
				Character: safeconv.IntToUint32(column),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(lineNum),
				Character: safeconv.IntToUint32(column + len(refName)),
			},
		},
	}
}

// tryExtractIsAttributeContext tries to extract definition context from an
// is="..." pattern.
//
// Takes line (string) which contains the text to search within.
// Takes cursor (int) which specifies the current cursor position.
// Takes position (protocol.Position) which provides the document position.
// Takes pattern (string) which defines the attribute pattern to match.
//
// Returns *PKDefinitionContext which contains the extracted context, or nil
// when no valid context is found.
func tryExtractIsAttributeContext(line string, cursor int, position protocol.Position, pattern string) *PKDefinitionContext {
	index := strings.LastIndex(line[:min(cursor+patternSearchRadius, len(line))], pattern)
	if index == -1 || index > cursor {
		return nil
	}

	quoteChar := pattern[len(pattern)-1]
	startPosition := index + len(pattern)
	endPosition := findQuoteEndPositionNav(line, startPosition, quoteChar)

	if endPosition == -1 || cursor < index || cursor > endPosition+1 {
		return nil
	}

	partialName := line[startPosition:endPosition]
	if partialName == "" {
		return nil
	}

	return &PKDefinitionContext{
		Kind:     PKDefPartial,
		Name:     partialName,
		Position: position,
	}
}

// findQuoteEndPositionNav finds the position of the closing quote character in a
// line of text.
//
// Takes line (string) which is the text to search within.
// Takes startPosition (int) which is the position to start searching from.
// Takes quoteChar (byte) which is the quote character to find.
//
// Returns int which is the position of the closing quote, or -1 if not found.
func findQuoteEndPositionNav(line string, startPosition int, quoteChar byte) int {
	for i := startPosition; i < len(line); i++ {
		if line[i] == quoteChar {
			return i
		}
	}
	return -1
}

// extractIsAttributeValue extracts the value from an is="..." attribute on the
// given line.
//
// Takes line (string) which contains the tag with the is attribute.
//
// Returns string which is the attribute value, or empty if not found.
func extractIsAttributeValue(line string) string {
	for _, pattern := range []string{`is="`, `is='`} {
		index := strings.Index(line, pattern)
		if index == -1 {
			continue
		}

		quoteChar := pattern[len(pattern)-1]
		startPosition := index + len(pattern)
		for i := startPosition; i < len(line); i++ {
			if line[i] == quoteChar {
				return line[startPosition:i]
			}
		}
	}

	return ""
}

// tryExtractEventHandlerContext tries to extract an event handler context
// for a single pattern.
//
// Takes line (string) which is the source line to search.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position for the result.
// Takes pattern (string) which is the event handler pattern to match.
//
// Returns *PKDefinitionContext which contains the handler context, or nil if
// no valid handler is found at the cursor position.
func tryExtractEventHandlerContext(line string, cursor int, position protocol.Position, pattern string) *PKDefinitionContext {
	index := strings.LastIndex(line[:min(cursor+patternSearchRadius, len(line))], pattern)
	if index == -1 || index > cursor {
		return nil
	}

	startPosition := index + len(pattern)
	endPosition := findEventHandlerEndPosition(line, startPosition)
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

	return &PKDefinitionContext{
		Kind:     PKDefHandler,
		Name:     handlerName,
		Position: position,
	}
}

// findEventHandlerEndPosition finds the end position of a handler name.
//
// Takes line (string) which contains the text to search.
// Takes startPosition (int) which is the offset to begin searching from.
//
// Returns int which is the position of the closing quote or opening bracket
// relative to startPosition, or -1 if neither is found.
func findEventHandlerEndPosition(line string, startPosition int) int {
	endPosition := strings.Index(line[startPosition:], `"`)
	if endPosition == -1 {
		endPosition = strings.Index(line[startPosition:], `(`)
	}
	return endPosition
}

// tryExtractQuotedValue searches for a pattern in the line near the cursor and
// extracts the quoted value that follows it. The last character of the pattern
// is treated as the opening quote character, and the function searches for the
// corresponding closing quote.
//
// Takes line (string) which is the source text to search.
// Takes cursor (int) which is the cursor position within the line.
// Takes pattern (string) which is the pattern whose last character is the quote.
//
// Returns name (string) which is the extracted quoted value.
// Returns startPosition (int) which is the position immediately after the pattern.
// Returns endPosition (int) which is the position of the closing quote.
// Returns ok (bool) which is true when a valid quoted value was found at the
// cursor position.
func tryExtractQuotedValue(line string, cursor int, pattern string) (name string, startPosition, endPosition int, ok bool) {
	index := strings.LastIndex(line[:min(cursor+30, len(line))], pattern)
	if index == -1 || index > cursor {
		return "", 0, 0, false
	}

	quoteChar := pattern[len(pattern)-1]
	startPosition = index + len(pattern)
	endPosition = findQuoteEndPosition(line, startPosition, quoteChar)

	if endPosition == -1 {
		return "", 0, 0, false
	}

	if cursor < startPosition || cursor > endPosition {
		return "", 0, 0, false
	}

	value := line[startPosition:endPosition]
	if value == "" {
		return "", 0, 0, false
	}

	return value, startPosition, endPosition, true
}

// tryExtractPartialReloadContext tries to find a partial reload context for
// a single pattern.
//
// Takes line (string) which contains the source line to search.
// Takes cursor (int) which specifies the cursor position within the line.
// Takes position (protocol.Position) which provides the LSP position information.
// Takes pattern (string) which defines the pattern to match against.
//
// Returns *PKDefinitionContext which contains the extracted context, or nil
// if no valid partial reload context can be found.
func tryExtractPartialReloadContext(line string, cursor int, position protocol.Position, pattern string) *PKDefinitionContext {
	name, _, _, ok := tryExtractQuotedValue(line, cursor, pattern)
	if !ok {
		return nil
	}

	return &PKDefinitionContext{
		Kind:     PKDefPartial,
		Name:     name,
		Position: position,
	}
}

// scanIdentifierAfterPrefix finds an identifier name following a prefix
// pattern near the cursor position. It searches backwards from the cursor
// (plus searchRadius) for the prefix, then scans forward to collect the
// identifier characters (letters, digits, underscore).
//
// Takes line (string) which is the source text to search.
// Takes cursor (int) which is the cursor position within the line.
// Takes prefix (string) which is the prefix pattern to look for.
// Takes searchRadius (int) which is how far past the cursor to include when
// searching for the prefix.
//
// Returns name (string) which is the identifier found after the prefix.
// Returns startPosition (int) which is the position of the
// first identifier character.
// Returns endPosition (int) which is the position after the
// last identifier character.
// Returns ok (bool) which is true when a non-empty identifier was found at the
// cursor position.
func scanIdentifierAfterPrefix(line string, cursor int, prefix string, searchRadius int) (name string, startPosition, endPosition int, ok bool) {
	index := strings.LastIndex(line[:min(cursor+searchRadius, len(line))], prefix)
	if index == -1 || index > cursor {
		return "", 0, 0, false
	}

	startPosition = index + len(prefix)
	endPosition = startPosition

	for i := startPosition; i < len(line); i++ {
		character := line[i]
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '_' {
			break
		}
		endPosition = i + 1
	}

	if cursor < startPosition || cursor > endPosition {
		return "", 0, 0, false
	}

	value := line[startPosition:endPosition]
	if value == "" {
		return "", 0, 0, false
	}

	return value, startPosition, endPosition, true
}

// isPartOfLongerIdentifier checks if the match is part of a longer identifier.
//
// Takes line (string) which contains the text being checked.
// Takes endIndex (int) which is the position after the matched text.
//
// Returns bool which is true if the character at endIndex is a letter, digit, or
// underscore. This means the match is part of a larger identifier.
func isPartOfLongerIdentifier(line string, endIndex int) bool {
	if endIndex >= len(line) {
		return false
	}
	nextChar := line[endIndex]
	return (nextChar >= 'a' && nextChar <= 'z') ||
		(nextChar >= 'A' && nextChar <= 'Z') ||
		(nextChar >= '0' && nextChar <= '9') ||
		nextChar == '_'
}

// findNextOccurrence finds the next match of a pattern after a given position.
//
// Takes line (string) which is the text to search within.
// Takes pattern (string) which is the text to find.
// Takes afterIndex (int) which is the position to start searching from.
//
// Returns int which is the position of the pattern, or -1 if not found.
func findNextOccurrence(line, pattern string, afterIndex int) int {
	index := strings.Index(line[afterIndex:], pattern)
	if index != -1 {
		return index + afterIndex
	}
	return -1
}
