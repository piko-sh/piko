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
	"piko.sh/piko/internal/logger/logger_domain"
)

// Completion handles completion requests and provides context-aware suggestions.
//
// Takes params (*protocol.CompletionParams) which specifies the cursor position
// and document for which to provide completions.
//
// Returns *protocol.CompletionList which contains the available completion items.
// Returns error when the completions cannot be retrieved.
func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Completion request received", logger_domain.String("uri", params.TextDocument.URI.Filename()))

	if result := s.tryFastPathCompletion(ctx, params); result != nil {
		l.Debug("Completion: fast-path succeeded",
			logger_domain.String("uri", params.TextDocument.URI.Filename()),
			logger_domain.Int("item_count", len(result.Items)))
		return result, nil
	}

	document, err := s.workspace.GetDocumentForCompletion(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Info("Completion: analysis had errors, attempting text-based completion",
			logger_domain.Error(err))
		document, _ = s.workspace.GetDocument(params.TextDocument.URI)
	}

	if document == nil {
		l.Error("Completion: no document available")
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	return document.GetCompletions(ctx, params.Position)
}
