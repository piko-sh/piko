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
	"fmt"

	"go.lsp.dev/protocol"
)

// generateUndefinedPartialAliasFixes creates fixes for undefined partial aliases.
// This includes typo corrections and offering to add missing imports.
//
// Takes diagnostic (protocol.Diagnostic) which contains the undefined
// partial alias error and associated fix data.
// Takes document (*document) which provides the document URI for text edits.
//
// Returns []protocol.CodeAction which contains the available fix
// actions such as typo corrections and import suggestions, or an
// empty slice if no fixes apply.
func generateUndefinedPartialAliasFixes(diagnostic protocol.Diagnostic, document *document, _ *workspace) []protocol.CodeAction {
	actions := []protocol.CodeAction{}

	fixData, ok := safeExtractData[undefinedPartialAliasData](diagnostic.Data)
	if !ok {
		return actions
	}

	if fixData.Suggestion != "" {
		actions = append(actions, createTypoCorrectionAction(document.URI, diagnostic, fixData.Suggestion))
	}

	if fixData.Alias != "" && fixData.PotentialPath != "" {
		actions = append(actions, protocol.CodeAction{
			Title:       fmt.Sprintf("Add import for '%s' from '%s'", fixData.Alias, fixData.PotentialPath),
			Kind:        protocol.QuickFix,
			Diagnostics: []protocol.Diagnostic{diagnostic},
			IsPreferred: false,
		})
	}

	return actions
}
