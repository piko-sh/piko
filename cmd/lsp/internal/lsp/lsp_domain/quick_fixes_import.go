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
	"piko.sh/piko/internal/sfcparser"
)

// generateAddImportFix creates a fix to add a missing import.
// This modifies the <script> block to add the import statement.
//
// Takes diagnostic (protocol.Diagnostic) which contains the missing
// import error and associated fix data.
// Takes document (*document) which provides the document content and URI.
//
// Returns *protocol.CodeAction which is the add-import fix action, or
// nil if the diagnostic lacks the required alias and import path or
// the modification cannot be prepared.
func generateAddImportFix(ctx context.Context, diagnostic protocol.Diagnostic, document *document, _ *workspace) *protocol.CodeAction {
	_, l := logger_domain.From(ctx, log)

	fixData, ok := safeExtractData[missingImportData](diagnostic.Data)
	if !ok || fixData.Alias == "" || fixData.ImportPath == "" {
		return nil
	}

	modifiedCode, goScript, err := prepareAddImportModification(document.Content, fixData)
	if err != nil {
		l.Debug("generateAddImportFix: failed to prepare modification", logger_domain.Error(err))
		return nil
	}

	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Add import %s %q", fixData.Alias, fixData.ImportPath),
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diagnostic},
		IsPreferred: true,
		Edit:        new(buildScriptBlockEdit(document.URI, goScript, modifiedCode)),
	}
}

// prepareAddImportModification parses an SFC file and adds an import to its Go
// script block.
//
// Takes content ([]byte) which is the raw SFC file content to parse.
// Takes fixData (missingImportData) which specifies the import to add.
//
// Returns string which is the formatted Go code with the import added.
// Returns *sfcparser.Script which is the original Go script block.
// Returns error when parsing fails or no Go script block is found.
func prepareAddImportModification(content []byte, fixData missingImportData) (string, *sfcparser.Script, error) {
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

	addImportToAST(goFile, fixData.Alias, fixData.ImportPath)

	formattedCode, err := formatGoAST(fset, goFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to format AST: %w", err)
	}

	return formattedCode, goScript, nil
}
