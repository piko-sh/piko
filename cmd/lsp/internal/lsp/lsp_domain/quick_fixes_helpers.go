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
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safeconv"
)

// parseSFC parses SFC file content and returns the parsed structure.
//
// Takes content ([]byte) which holds the raw SFC file data.
//
// Returns *sfcparser.ParseResult which holds the parsed SFC components.
// Returns error when the content cannot be parsed.
func parseSFC(content []byte) (*sfcparser.ParseResult, error) {
	return sfcparser.Parse(content)
}

// parseGoScript parses Go source code and returns the syntax tree.
//
// Takes scriptContent (string) which contains the Go source code to parse.
//
// Returns *ast.File which is the parsed syntax tree.
// Returns *token.FileSet which provides position information for the tree.
// Returns error when the source code is not valid Go.
func parseGoScript(scriptContent string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "script.go", scriptContent, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing Go script: %w", err)
	}
	return file, fset, nil
}

// formatGoAST turns a Go AST node back into source code text.
//
// Takes fset (*token.FileSet) which holds position data for the node.
// Takes node (ast.Node) which is the AST node to format.
//
// Returns string which is the formatted Go source code.
// Returns error when the AST node cannot be formatted.
func formatGoAST(fset *token.FileSet, node ast.Node) (string, error) {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		return "", fmt.Errorf("formatting Go AST: %w", err)
	}
	return buffer.String(), nil
}

// safeExtractData extracts and converts diagnostic data to a typed struct.
// This prevents runtime panics when accessing diagnostic data fields.
//
// Takes data (any) which is the raw diagnostic data to extract from.
//
// Returns T which is the extracted and converted typed struct.
// Returns bool which indicates whether the extraction was successful.
func safeExtractData[T any](data any) (T, bool) {
	var result T
	if data == nil {
		return result, false
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return result, false
	}

	resultPtr := any(&result)
	switch ptr := resultPtr.(type) {
	case *typeMismatchData:
		extractTypeMismatchData(dataMap, ptr)
		return result, true
	case *undefinedVariableData:
		extractUndefinedVariableData(dataMap, ptr)
		return result, true
	case *undefinedPartialAliasData:
		extractUndefinedPartialAliasData(dataMap, ptr)
		return result, true
	case *missingRequiredPropData:
		extractMissingRequiredPropData(dataMap, ptr)
		return result, true
	case *missingImportData:
		extractMissingImportData(dataMap, ptr)
		return result, true
	}

	return result, false
}

// extractTypeMismatchData extracts type mismatch data from a map into a struct.
//
// Takes dataMap (map[string]any) which holds the raw diagnostic data.
// Takes r (*typeMismatchData) which receives the extracted values.
func extractTypeMismatchData(dataMap map[string]any, r *typeMismatchData) {
	if v, ok := dataMap["can_coerce"].(bool); ok {
		r.CanCoerce = v
	}
	if v, ok := dataMap["prop_def_path"].(string); ok {
		r.PropDefPath = v
	}
	if v, ok := dataMap["prop_def_line"].(float64); ok {
		r.PropDefLine = int(v)
	}
	if v, ok := dataMap["prop_name"].(string); ok {
		r.PropName = v
	}
}

// extractUndefinedVariableData reads undefined variable data from a map and
// stores the values in the result struct.
//
// Takes dataMap (map[string]any) which holds the raw data from a diagnostic.
// Takes r (*undefinedVariableData) which receives the extracted values.
func extractUndefinedVariableData(dataMap map[string]any, r *undefinedVariableData) {
	if v, ok := dataMap["suggestion"].(string); ok {
		r.Suggestion = v
	}
	if v, ok := dataMap["is_prop"].(bool); ok {
		r.IsProp = v
	}
	if v, ok := dataMap["prop_name"].(string); ok {
		r.PropName = v
	}
	if v, ok := dataMap["suggested_type"].(string); ok {
		r.SuggestedType = v
	}
}

// extractUndefinedPartialAliasData reads undefined partial alias data from a
// map and stores the values in the given result struct.
//
// Takes dataMap (map[string]any) which contains the raw data to read from.
// Takes r (*undefinedPartialAliasData) which receives the extracted values.
func extractUndefinedPartialAliasData(dataMap map[string]any, r *undefinedPartialAliasData) {
	if v, ok := dataMap["suggestion"].(string); ok {
		r.Suggestion = v
	}
	if v, ok := dataMap["alias"].(string); ok {
		r.Alias = v
	}
	if v, ok := dataMap["potential_path"].(string); ok {
		r.PotentialPath = v
	}
}

// extractMissingRequiredPropData reads values from a map and stores them in
// the given struct.
//
// Takes dataMap (map[string]any) which contains the raw diagnostic data.
// Takes r (*missingRequiredPropData) which receives the extracted values.
func extractMissingRequiredPropData(dataMap map[string]any, r *missingRequiredPropData) {
	if v, ok := dataMap["prop_name"].(string); ok {
		r.PropName = v
	}
	if v, ok := dataMap["prop_type"].(string); ok {
		r.PropType = v
	}
	if v, ok := dataMap["suggested_value"].(string); ok {
		r.SuggestedValue = v
	}
}

// extractMissingImportData reads import data from a map into a result struct.
//
// Takes dataMap (map[string]any) which contains the raw diagnostic data.
// Takes r (*missingImportData) which receives the alias and import path.
func extractMissingImportData(dataMap map[string]any, r *missingImportData) {
	if v, ok := dataMap["alias"].(string); ok {
		r.Alias = v
	}
	if v, ok := dataMap["import_path"].(string); ok {
		r.ImportPath = v
	}
}

// readFileContent reads a file using the workspace's document cache.
// This reads the in-memory editor version rather than the
// on-disk version.
//
// Takes ws (*workspace) which provides access to the document cache.
// Takes uri (protocol.DocumentURI) which identifies the file to read.
//
// Returns []byte which contains the file content from the cache.
// Returns error when the workspace or cache is nil, or the file is not
// found in the cache.
func readFileContent(ws *workspace, uri protocol.DocumentURI) ([]byte, error) {
	if ws == nil || ws.docCache == nil {
		return nil, errors.New("workspace or docCache is nil")
	}

	content, found := ws.docCache.Get(uri)
	if !found {
		return nil, fmt.Errorf("file not found in cache: %s", uri)
	}

	return content, nil
}

// createSimpleTextEdit creates a TextEdit that replaces a range with new text.
//
// Takes startLine (uint32) which is the starting line of the range.
// Takes startChar (uint32) which is the starting character position.
// Takes endLine (uint32) which is the ending line of the range.
// Takes endChar (uint32) which is the ending character position.
// Takes newText (string) which is the text to insert.
//
// Returns protocol.TextEdit which holds the range and new text.
func createSimpleTextEdit(startLine, startChar, endLine, endChar uint32, newText string) protocol.TextEdit {
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: startLine, Character: startChar},
			End:   protocol.Position{Line: endLine, Character: endChar},
		},
		NewText: newText,
	}
}

// createTypoCorrectionAction creates a quick fix action that suggests a
// corrected spelling for a misspelled identifier.
//
// Takes uri (protocol.DocumentURI) which identifies the document to modify.
// Takes diagnostic (protocol.Diagnostic) which provides the error location.
// Takes suggestion (string) which is the correct spelling to suggest.
//
// Returns protocol.CodeAction which replaces the misspelled identifier with
// the suggested correction.
func createTypoCorrectionAction(uri protocol.DocumentURI, diagnostic protocol.Diagnostic, suggestion string) protocol.CodeAction {
	edit := protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			uri: {
				createSimpleTextEdit(
					diagnostic.Range.Start.Line,
					diagnostic.Range.Start.Character,
					diagnostic.Range.End.Line,
					diagnostic.Range.End.Character,
					suggestion,
				),
			},
		},
	}

	return protocol.CodeAction{
		Title:       fmt.Sprintf("Did you mean '%s'?", suggestion),
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		IsPreferred: true,
		Edit:        &edit,
	}
}

// buildScriptBlockEdit creates a WorkspaceEdit that replaces a script block's
// content. This is used by quick fixes that modify the Go script within an SFC
// file.
//
// Takes targetURI (protocol.DocumentURI) which identifies the file to edit.
// Takes goScript (*sfcparser.Script) which provides the script block location.
// Takes newContent (string) which is the replacement content for the block.
//
// Returns protocol.WorkspaceEdit which contains the text edit for the script
// block replacement.
func buildScriptBlockEdit(targetURI protocol.DocumentURI, goScript *sfcparser.Script, newContent string) protocol.WorkspaceEdit {
	scriptRange := calculateScriptBlockRange(goScript)

	return protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			targetURI: {
				{
					Range:   scriptRange,
					NewText: newContent,
				},
			},
		},
	}
}

// calculateScriptBlockRange calculates the LSP range for a script block's
// content.
//
// Takes goScript (*sfcparser.Script) which contains the parsed script block
// with its content and location information.
//
// Returns protocol.Range which spans from the script's start position to the
// end of its content.
func calculateScriptBlockRange(goScript *sfcparser.Script) protocol.Range {
	scriptStartLine := safeconv.IntToUint32(goScript.ContentLocation.Line - 1)
	scriptStartChar := safeconv.IntToUint32(goScript.ContentLocation.Column - 1)

	originalLines := strings.Split(goScript.Content, "\n")
	scriptEndLine := scriptStartLine + safeconv.IntToUint32(len(originalLines)) - 1

	var scriptEndChar uint32
	if len(originalLines) > 0 {
		scriptEndChar = safeconv.IntToUint32(len(originalLines[len(originalLines)-1]))
		if len(originalLines) == 1 {
			scriptEndChar += scriptStartChar
		}
	}

	return protocol.Range{
		Start: protocol.Position{Line: scriptStartLine, Character: scriptStartChar},
		End:   protocol.Position{Line: scriptEndLine, Character: scriptEndChar},
	}
}
