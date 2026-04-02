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

// Definition handles go-to-definition requests for symbols in the document.
//
// Takes params (*protocol.DefinitionParams) which specifies the document and
// position to find the definition for.
//
// Returns []protocol.Location which contains the definition locations found.
// Returns error when the definition lookup fails.
func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Definition request received", logger_domain.String("uri", params.TextDocument.URI.Filename()))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Error("Definition: analysis failed", logger_domain.Error(err))
		return nil, nil
	}

	return document.GetDefinition(ctx, params.Position)
}
