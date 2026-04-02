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
	"go/ast"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
)

// generateUndefinedVariableFixes creates fix options for undefined variables.
// This includes typo corrections and adding new props to the component.
//
// Takes diagnostic (protocol.Diagnostic) which contains the error details.
// Takes document (*document) which provides the source document context.
// Takes ws (*workspace) which gives access to workspace resources.
//
// Returns []protocol.CodeAction which contains the available fix actions.
func generateUndefinedVariableFixes(ctx context.Context, diagnostic protocol.Diagnostic, document *document, ws *workspace) []protocol.CodeAction {
	actions := []protocol.CodeAction{}

	fixData, ok := safeExtractData[undefinedVariableData](diagnostic.Data)
	if !ok {
		return actions
	}

	if fixData.Suggestion != "" {
		actions = append(actions, createTypoCorrectionAction(document.URI, diagnostic, fixData.Suggestion))
	}

	if fixData.IsProp && fixData.PropName != "" {
		if action := generateAddToPropsEdit(ctx, diagnostic, document, ws, fixData); action != nil {
			actions = append(actions, *action)
		}
	}

	return actions
}

// generateAddToPropsEdit creates a fix to add a new field to the component's
// Props struct. This modifies the current document's <script> block to add the
// missing prop.
//
// Takes diagnostic (protocol.Diagnostic) which contains the undefined
// variable error details.
// Takes document (*document) which provides the document content and URI.
// Takes fixData (undefinedVariableData) which provides the prop name
// and suggested type for the new field.
//
// Returns *protocol.CodeAction which is the add-to-props fix action,
// or nil if the modification cannot be prepared.
func generateAddToPropsEdit(ctx context.Context, diagnostic protocol.Diagnostic, document *document, _ *workspace, fixData undefinedVariableData) *protocol.CodeAction {
	_, l := logger_domain.From(ctx, log)

	modifiedCode, goScript, err := prepareAddPropModification(document.Content, fixData)
	if err != nil {
		l.Debug("generateAddToPropsEdit: failed to prepare modification", logger_domain.Error(err))
		return nil
	}

	fieldType := getFieldType(fixData.SuggestedType)

	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Add '%s %s' to component Props", fixData.PropName, fieldType),
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		IsPreferred: false,
		Edit:        new(buildScriptBlockEdit(document.URI, goScript, modifiedCode)),
	}
}

// prepareAddPropModification parses an SFC file and modifies its AST to add a
// new prop field.
//
// Takes content ([]byte) which contains the SFC file content to parse.
// Takes fixData (undefinedVariableData) which provides the prop field details.
//
// Returns string which is the formatted Go code with the new prop field.
// Returns *sfcparser.Script which is the Go script block from the SFC.
// Returns error when parsing fails or the SFC has no Go script block.
func prepareAddPropModification(content []byte, fixData undefinedVariableData) (string, *sfcparser.Script, error) {
	sfcResult, err := parseSFC(content)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse SFC: %w", err)
	}

	goScript, found := sfcResult.GoScript()
	if !found {
		return "", nil, errors.New("no Go script block found")
	}

	goFile, fset, err := parseGoScript(goScript.Content)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse Go script: %w", err)
	}

	if err := addPropFieldToAST(goFile, fixData); err != nil {
		return "", nil, err
	}

	formattedCode, err := formatGoAST(fset, goFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to format AST: %w", err)
	}

	return formattedCode, goScript, nil
}

// addPropFieldToAST adds a new field to the Props struct in the AST.
//
// Takes goFile (*ast.File) which contains the AST to modify.
// Takes fixData (undefinedVariableData) which provides the field name and type.
//
// Returns error when the Props struct cannot be found in the file.
func addPropFieldToAST(goFile *ast.File, fixData undefinedVariableData) error {
	_, propsStruct, found := findPropsStruct(goFile)
	if !found {
		return errors.New("props struct not found")
	}

	fieldType := getFieldType(fixData.SuggestedType)
	newField := &ast.Field{
		Names: []*ast.Ident{{Name: fixData.PropName}},
		Type:  &ast.Ident{Name: fieldType},
	}
	propsStruct.Fields.List = append(propsStruct.Fields.List, newField)

	return nil
}

// getFieldType returns the field type to use.
//
// Takes suggestedType (string) which is the preferred type.
//
// Returns string which is the suggested type, or "string" if empty.
func getFieldType(suggestedType string) string {
	if suggestedType == "" {
		return "string"
	}
	return suggestedType
}
