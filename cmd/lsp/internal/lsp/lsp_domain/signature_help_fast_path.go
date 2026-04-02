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

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// signatureCallContext holds the parsed function call context from text.
type signatureCallContext struct {
	// FunctionName is the name of the function being called.
	FunctionName string

	// BaseExpression is the receiver for method calls (e.g., "state.user").
	// Empty for standalone function calls.
	BaseExpression string

	// ActiveParameter is the zero-based index of the current parameter.
	ActiveParameter int

	// IsMethodCall indicates whether this is a method call (base.method()).
	IsMethodCall bool
}

// tryFastPathSignatureHelp attempts to provide signature help using cached
// analysis data without waiting for a full rebuild.
//
// Takes params (*protocol.SignatureHelpParams) which specifies the signature
// help request parameters.
//
// Returns *protocol.SignatureHelp which contains the signature help if the
// fast-path succeeded, or nil if it should fall back to the normal path.
func (s *Server) tryFastPathSignatureHelp(ctx context.Context, params *protocol.SignatureHelpParams) *protocol.SignatureHelp {
	_, l := logger_domain.From(ctx, log)

	uri := params.TextDocument.URI
	position := params.Position

	document, exists := s.workspace.GetDocument(uri)
	if !exists {
		l.Trace("Fast-path signature help: no document found",
			logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	if !document.hasSignatureHelpPrerequisites() {
		l.Trace("Fast-path signature help: document lacks analysis data",
			logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	content, ok := s.workspace.docCache.Get(uri)
	if !ok {
		l.Trace("Fast-path signature help: no content in cache",
			logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	callCtx := analyseSignatureContextFromContent(content, position)
	if callCtx == nil {
		return nil
	}

	l.Debug("Fast-path signature help: attempting",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.String("function", callCtx.FunctionName),
		logger_domain.String("base", callCtx.BaseExpression),
		logger_domain.Int("activeParam", callCtx.ActiveParameter))

	result := document.getSignatureHelpFast(ctx, callCtx, position)
	if result == nil || len(result.Signatures) == 0 {
		l.Debug("Fast-path signature help: resolution failed",
			logger_domain.String("function", callCtx.FunctionName))
		return nil
	}

	l.Debug("Fast-path signature help: success",
		logger_domain.String("function", callCtx.FunctionName),
		logger_domain.Int("activeParam", callCtx.ActiveParameter))

	return result
}

// hasSignatureHelpPrerequisites checks if the document has the data needed
// for fast-path signature help.
//
// Returns bool which is true when all required fields are present.
func (d *document) hasSignatureHelpPrerequisites() bool {
	return d.AnnotationResult != nil &&
		d.AnnotationResult.AnnotatedAST != nil &&
		d.AnalysisMap != nil &&
		d.TypeInspector != nil
}

// getSignatureHelpFast provides signature help using cached analysis data.
//
// Takes callCtx (*signatureCallContext) which describes the function call
// detected from text.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *protocol.SignatureHelp which contains the signature help, or nil
// if resolution fails.
func (d *document) getSignatureHelpFast(ctx context.Context, callCtx *signatureCallContext, position protocol.Position) *protocol.SignatureHelp {
	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil {
		return nil
	}

	analysisCtx, exists := d.AnalysisMap[targetNode]
	if !exists || analysisCtx == nil {
		return nil
	}

	var funcSig *inspector_dto.FunctionSignature

	if callCtx.IsMethodCall {
		funcSig = d.resolveMethodSignatureFast(ctx, callCtx, analysisCtx, position)
	} else {
		funcSig = d.resolveFunctionSignatureFast(callCtx, analysisCtx)
	}

	if funcSig == nil {
		return nil
	}

	return d.buildSignatureHelp(callCtx.FunctionName, funcSig, callCtx.ActiveParameter)
}

// resolveFunctionSignatureFast looks up a standalone function signature.
//
// Takes callCtx (*signatureCallContext) which contains the function name.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides the
// current package and source context.
//
// Returns *inspector_dto.FunctionSignature which is the resolved signature,
// or nil if not found.
func (d *document) resolveFunctionSignatureFast(
	callCtx *signatureCallContext,
	analysisCtx *annotator_domain.AnalysisContext,
) *inspector_dto.FunctionSignature {
	if analysisCtx.Symbols != nil {
		if symbol, found := analysisCtx.Symbols.Find(callCtx.FunctionName); found && symbol.TypeInfo != nil {
			return d.TypeInspector.FindFuncSignature(
				symbol.TypeInfo.PackageAlias,
				callCtx.FunctionName,
				analysisCtx.CurrentGoFullPackagePath,
				analysisCtx.CurrentGoSourcePath,
			)
		}
	}

	funcSig := d.TypeInspector.FindFuncSignature(
		"",
		callCtx.FunctionName,
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)
	if funcSig != nil {
		return funcSig
	}

	return nil
}

// resolveMethodSignatureFast finds the signature for a method call.
//
// Takes callCtx (*signatureCallContext) which holds the base expression and
// method name.
// Takes analysisCtx (*annotator_domain.AnalysisContext) which provides the
// current package context.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *inspector_dto.FunctionSignature which is the method signature, or
// nil if not found.
func (d *document) resolveMethodSignatureFast(
	ctx context.Context,
	callCtx *signatureCallContext,
	analysisCtx *annotator_domain.AnalysisContext,
	position protocol.Position,
) *inspector_dto.FunctionSignature {
	baseType := d.resolveExpressionFromText(ctx, callCtx.BaseExpression, position)
	if baseType == nil || baseType.TypeExpression == nil {
		return nil
	}

	return d.TypeInspector.FindMethodSignature(
		baseType.TypeExpression,
		callCtx.FunctionName,
		analysisCtx.CurrentGoFullPackagePath,
		analysisCtx.CurrentGoSourcePath,
	)
}

// parenScanState tracks position while scanning for an opening parenthesis.
type parenScanState struct {
	// depth tracks nesting level of parentheses; 0 means at the outermost level.
	depth int

	// commaCount tracks the number of commas seen at depth zero.
	commaCount int

	// inString indicates whether the scanner is inside a quoted string literal.
	inString bool

	// inRawString indicates whether the scanner is inside a raw string literal.
	inRawString bool

	// escapeNext indicates that the next character should be skipped due to a
	// backslash escape sequence in a string.
	escapeNext bool
}

// processChar processes a single character during paren scanning.
//
// Takes text ([]byte) which contains the text being scanned.
// Takes i (int) which is the index of the character to process.
//
// Returns found (bool) which is true if the opening paren was found.
// Returns parenIndex (int) which is the index of the opening paren if found.
// Returns commaCount (int) which is the current comma count.
func (s *parenScanState) processChar(text []byte, i int) (found bool, parenIndex int, commaCount int) {
	character := text[i]

	if s.escapeNext {
		s.escapeNext = false
		return false, 0, 0
	}

	if s.handleStringDelimiters(character, i) {
		return false, 0, 0
	}

	if s.inRawString || s.inString {
		return false, 0, 0
	}

	return s.handleParensAndCommas(character, i)
}

// handleStringDelimiters handles raw strings and regular string delimiters.
//
// Takes character (byte) which is the character to check for string delimiters.
// Takes i (int) which is the current position in the input.
//
// Returns bool which is true if the character was consumed by string handling.
func (s *parenScanState) handleStringDelimiters(character byte, i int) bool {
	if character == '`' {
		s.inRawString = !s.inRawString
		return true
	}
	if s.inRawString {
		return true
	}

	if character == '"' || character == '\'' {
		s.inString = !s.inString
		return true
	}
	if s.inString {
		if character == '\\' && i > 0 {
			s.escapeNext = true
		}
		return true
	}

	return false
}

// handleParensAndCommas processes parentheses and comma characters.
//
// Takes character (byte) which is the character to process.
// Takes i (int) which is the current index in the string.
//
// Returns found (bool) which is true if the target opening paren was found.
// Returns parenIndex (int) which is the index if found.
// Returns commaCount (int) which is the current comma count.
func (s *parenScanState) handleParensAndCommas(character byte, i int) (found bool, parenIndex int, commaCount int) {
	switch character {
	case ')':
		s.depth++
	case '(':
		if s.depth == 0 {
			return true, i, s.commaCount
		}
		s.depth--
	case ',':
		if s.depth == 0 {
			s.commaCount++
		}
	}
	return false, 0, 0
}

// analyseSignatureContextFromContent checks the text around the cursor to
// find function call context.
//
// Takes content ([]byte) which is the raw document content.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *signatureCallContext which describes the found function call,
// or nil if not inside a function call.
func analyseSignatureContextFromContent(content []byte, position protocol.Position) *signatureCallContext {
	_, lineFound := getLineAtPosition(content, position.Line)
	if !lineFound {
		return nil
	}

	cursorOffset := positionToByteOffset(content, position)
	if cursorOffset < 0 {
		return nil
	}

	return findEnclosingCallFromBytes(content[:cursorOffset])
}

// findEnclosingCallFromBytes finds the function call that surrounds the cursor
// by scanning backwards through the text before the cursor.
//
// Takes textBeforeCursor ([]byte) which contains all text from the document
// start up to the cursor position.
//
// Returns *signatureCallContext which describes the surrounding call, or nil
// if no call is found.
func findEnclosingCallFromBytes(textBeforeCursor []byte) *signatureCallContext {
	if len(textBeforeCursor) == 0 {
		return nil
	}

	parenPos, activeParam := findOpeningParen(textBeforeCursor)
	if parenPos < 0 {
		return nil
	}

	functionName, baseExpr := extractCalleeFromText(textBeforeCursor[:parenPos])
	if functionName == "" {
		return nil
	}

	return &signatureCallContext{
		FunctionName:    functionName,
		BaseExpression:  baseExpr,
		ActiveParameter: activeParam,
		IsMethodCall:    baseExpr != "",
	}
}

// findEnclosingCallFromText finds the function call that surrounds the cursor
// by scanning backwards through the text.
//
// Takes lines ([][]byte) which contains the document lines.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *signatureCallContext which describes the surrounding call, or nil
// if no call is found.
func findEnclosingCallFromText(lines [][]byte, position protocol.Position) *signatureCallContext {
	textBeforeCursor := buildTextBeforeCursor(lines, position)
	if len(textBeforeCursor) == 0 {
		return nil
	}

	parenPos, activeParam := findOpeningParen(textBeforeCursor)
	if parenPos < 0 {
		return nil
	}

	functionName, baseExpr := extractCalleeFromText(textBeforeCursor[:parenPos])
	if functionName == "" {
		return nil
	}

	return &signatureCallContext{
		FunctionName:    functionName,
		BaseExpression:  baseExpr,
		ActiveParameter: activeParam,
		IsMethodCall:    baseExpr != "",
	}
}

// buildTextBeforeCursor joins all lines from the start of the document up to
// the cursor position.
//
// Takes lines ([][]byte) which contains the document lines.
// Takes position (protocol.Position) which specifies where the cursor is.
//
// Returns []byte which contains the text from the document start to the cursor.
func buildTextBeforeCursor(lines [][]byte, position protocol.Position) []byte {
	var result []byte

	for i := uint32(0); i < position.Line && int(i) < len(lines); i++ {
		result = append(result, lines[i]...)
		result = append(result, '\n')
	}

	if int(position.Line) < len(lines) {
		line := lines[position.Line]
		endChar := min(int(position.Character), len(line))
		result = append(result, line[:endChar]...)
	}

	return result
}

// findOpeningParen scans text backwards to find the opening parenthesis of a
// function call. It tracks nesting to handle calls within calls.
//
// Takes text ([]byte) which is the text to scan.
//
// Returns parenIndex (int) which is the position of the opening parenthesis,
// or -1 if not found.
// Returns paramIndex (int) which is the active parameter index based on comma
// count.
func findOpeningParen(text []byte) (parenIndex int, paramIndex int) {
	state := &parenScanState{}

	for i := len(text) - 1; i >= 0; i-- {
		if found, index, count := state.processChar(text, i); found {
			return index, count
		}
	}

	return -1, 0
}

// extractCalleeFromText extracts the function name and base expression from
// text that appears before an opening parenthesis.
//
// Takes text ([]byte) which is the bytes before the opening parenthesis.
//
// Returns functionName (string) which is the function or method name.
// Returns baseExpr (string) which is the base expression, or empty for
// standalone functions.
func extractCalleeFromText(text []byte) (functionName string, baseExpr string) {
	if len(text) == 0 {
		return "", ""
	}

	end := len(text)
	for end > 0 && end <= len(text) && isWhitespace(text[end-1]) { // #nosec G602 -- end is always in bounds; explicit check satisfies static analysis
		end--
	}
	if end == 0 {
		return "", ""
	}

	expressionStart := findExpressionStart(text[:end])
	fullExpr := string(text[expressionStart:end])
	if fullExpr == "" {
		return "", ""
	}

	lastDot := -1
	for i := len(fullExpr) - 1; i >= 0; i-- {
		if fullExpr[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot > 0 {
		return fullExpr[lastDot+1:], fullExpr[:lastDot]
	}

	return fullExpr, ""
}
