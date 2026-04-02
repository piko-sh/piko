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
	goast "go/ast"
	"go/token"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/sfcparser"
)

// generateCoerceFix creates a quick fix to add coerce:"true" tag for type
// mismatches. This fix modifies the Props struct in the partial component's
// file to add the coerce tag.
//
// Takes diagnostic (protocol.Diagnostic) which contains the type
// mismatch error and associated fix data.
// Takes ws (*workspace) which provides access to workspace files for
// reading the target component.
//
// Returns *protocol.CodeAction which is the coerce tag fix action, or
// nil if the diagnostic lacks the required data or the modification
// cannot be prepared.
func generateCoerceFix(ctx context.Context, diagnostic protocol.Diagnostic, _ *document, ws *workspace) *protocol.CodeAction {
	_, l := logger_domain.From(ctx, log)

	fixData, ok := safeExtractData[typeMismatchData](diagnostic.Data)
	if !ok || !fixData.CanCoerce {
		return nil
	}

	if fixData.PropDefPath == "" || fixData.PropName == "" {
		l.Debug("generateCoerceFix: missing required data",
			logger_domain.String("propDefPath", fixData.PropDefPath),
			logger_domain.String("propName", fixData.PropName))
		return nil
	}

	targetURI := protocol.DocumentURI("file://" + fixData.PropDefPath)
	modifiedCode, goScript, err := prepareCoerceTagModification(ws, targetURI, fixData.PropName)
	if err != nil {
		l.Debug("generateCoerceFix: failed to prepare modification", logger_domain.Error(err))
		return nil
	}

	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Add coerce:\"true\" tag to '%s' prop", fixData.PropName),
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		IsPreferred: true,
		Edit:        new(buildScriptBlockEdit(targetURI, goScript, modifiedCode)),
	}
}

// prepareCoerceTagModification handles all parsing and AST modification for
// adding a coerce tag.
//
// Takes ws (*workspace) which provides access to workspace resources.
// Takes targetURI (protocol.DocumentURI) which specifies the document to
// modify.
// Takes propName (string) which identifies the property to add the coerce tag
// to.
//
// Returns string which contains the formatted Go code after modification.
// Returns *sfcparser.Script which is the parsed script for further processing.
// Returns error when parsing fails, the property is not found, or formatting
// fails.
func prepareCoerceTagModification(ws *workspace, targetURI protocol.DocumentURI, propName string) (string, *sfcparser.Script, error) {
	goScript, goFile, fset, err := parsePropsFromURI(ws, targetURI)
	if err != nil {
		return "", nil, err
	}

	field, err := findPropsField(goFile, propName)
	if err != nil {
		return "", nil, err
	}

	addCoerceTagToField(field)

	formattedCode, err := formatGoAST(fset, goFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to format AST: %w", err)
	}

	return formattedCode, goScript, nil
}

// parsePropsFromURI parses a file URI and returns the Go script block and AST.
//
// Takes ws (*workspace) which provides access to workspace files.
// Takes targetURI (protocol.DocumentURI) which specifies the file to parse.
//
// Returns *sfcparser.Script which is the parsed Go script block from the SFC.
// Returns *goast.File which is the parsed Go AST.
// Returns *token.FileSet which is the file set for position information.
// Returns error when the file cannot be read, the SFC cannot be parsed, no Go
// script block is found, or the Go script cannot be parsed.
func parsePropsFromURI(ws *workspace, targetURI protocol.DocumentURI) (*sfcparser.Script, *goast.File, *token.FileSet, error) {
	targetContent, err := readFileContent(ws, targetURI)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read target file: %w", err)
	}

	sfcResult, err := parseSFC(targetContent)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse SFC: %w", err)
	}

	goScript, found := sfcResult.GoScript()
	if !found {
		return nil, nil, nil, errors.New("no Go script block found")
	}

	goFile, fset, err := parseGoScript(goScript.Content)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse Go script: %w", err)
	}

	return goScript, goFile, fset, nil
}

// findPropsField finds a named field within the Props struct.
//
// Takes goFile (*goast.File) which is the parsed Go source file to search.
// Takes propName (string) which is the name of the field to find.
//
// Returns *goast.Field which is the matching field from the Props struct.
// Returns error when the Props struct cannot be found or the field does not
// exist.
func findPropsField(goFile *goast.File, propName string) (*goast.Field, error) {
	_, propsStruct, found := findPropsStruct(goFile)
	if !found {
		return nil, errors.New("props struct not found")
	}

	field, _, found := findFieldInStruct(propsStruct, propName)
	if !found {
		return nil, fmt.Errorf("field '%s' not found in props struct", propName)
	}

	return field, nil
}
