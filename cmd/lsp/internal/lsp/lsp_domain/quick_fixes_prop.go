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
	"errors"
	"fmt"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/logger/logger_domain"
)

// propInsertionInfo holds the position where a new prop should be added.
type propInsertionInfo struct {
	// line is the line number where the property should be inserted.
	line uint32

	// char is the character position within the line where text is inserted.
	char uint32

	// isSelfClosing indicates whether the target element is a self-closing tag.
	isSelfClosing bool
}

// generateAddMissingPropFix creates a fix to add a missing required prop.
// This uses AST-based tag detection to find the exact insertion point.
//
// Takes diagnostic (protocol.Diagnostic) which contains the missing
// prop error and associated fix data.
// Takes document (*document) which provides the annotated AST and document
// content for locating the insertion point.
//
// Returns *protocol.CodeAction which is the add-prop fix action, or
// nil if the diagnostic lacks a prop name or the insertion point
// cannot be determined.
func generateAddMissingPropFix(ctx context.Context, diagnostic protocol.Diagnostic, document *document, _ *workspace) *protocol.CodeAction {
	_, l := logger_domain.From(ctx, log)

	fixData, ok := safeExtractData[missingRequiredPropData](diagnostic.Data)
	if !ok || fixData.PropName == "" {
		return nil
	}

	insertInfo, err := findPropInsertionPoint(document, diagnostic.Range.Start)
	if err != nil {
		l.Debug("generateAddMissingPropFix: " + err.Error())
		return nil
	}

	suggestedValue := getDefaultSuggestedValue(fixData.SuggestedValue)
	insertText := fmt.Sprintf(" :%s=%s", fixData.PropName, suggestedValue)

	title := buildPropFixTitle(fixData.PropName, insertInfo.isSelfClosing)

	return &protocol.CodeAction{
		Title:       title,
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		IsPreferred: true,
		Edit:        new(buildInsertionEdit(document.URI, insertInfo.line, insertInfo.char, insertText)),
	}
}

// findPropInsertionPoint finds where to insert a prop in a component tag.
//
// Takes document (*document) which contains the parsed document with annotations.
// Takes position (protocol.Position) which specifies the cursor position.
//
// Returns *propInsertionInfo which contains the insertion point details.
// Returns error when the document has no annotated AST, no component tag is
// found at the position, or the tag end position cannot be found.
func findPropInsertionPoint(document *document, position protocol.Position) (*propInsertionInfo, error) {
	if document.AnnotationResult == nil || document.AnnotationResult.AnnotatedAST == nil || len(document.AnnotationResult.AnnotatedAST.RootNodes) == 0 {
		return nil, errors.New("no annotated AST available")
	}

	componentTag := findComponentTagAtPosition(
		document.AnnotationResult.AnnotatedAST.RootNodes,
		position.Line,
		position.Character,
	)
	if componentTag == nil {
		return nil, errors.New("could not find component tag at position")
	}

	tagEndLine, tagEndChar, isSelfClosing, found := calculateTagEndPosition(document.Content, componentTag)
	if !found {
		return nil, errors.New("could not find tag end position")
	}

	return &propInsertionInfo{
		line:          tagEndLine,
		char:          tagEndChar,
		isSelfClosing: isSelfClosing,
	}, nil
}

// getDefaultSuggestedValue returns the given value or a default placeholder.
//
// Takes suggestedValue (string) which is the value to check.
//
// Returns string which is the original value if not empty, or an empty string
// literal ("") if the input was empty.
func getDefaultSuggestedValue(suggestedValue string) string {
	if suggestedValue == "" {
		return "\"\""
	}
	return suggestedValue
}

// buildInsertionEdit creates a WorkspaceEdit for inserting text at a position.
//
// Takes uri (protocol.DocumentURI) which identifies the document to edit.
// Takes line (uint32) which specifies the line number for the insertion.
// Takes char (uint32) which specifies the character position for the insertion.
// Takes text (string) which contains the text to insert.
//
// Returns protocol.WorkspaceEdit which contains the insertion operation.
func buildInsertionEdit(uri protocol.DocumentURI, line, char uint32, text string) protocol.WorkspaceEdit {
	return protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			uri: {createSimpleTextEdit(line, char, line, char, text)},
		},
	}
}

// buildPropFixTitle creates the title for a quick fix action.
//
// Takes propName (string) which is the name of the missing prop.
// Takes isSelfClosing (bool) which shows if the tag is self-closing.
//
// Returns string which is the title for the quick fix action.
func buildPropFixTitle(propName string, isSelfClosing bool) string {
	if isSelfClosing {
		return fmt.Sprintf("Add missing required prop '%s' (self-closing tag)", propName)
	}
	return fmt.Sprintf("Add missing required prop '%s'", propName)
}
