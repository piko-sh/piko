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

// Hover handles hover requests to show type information for symbols.
//
// Takes params (*protocol.HoverParams) which specifies the document and
// position to get hover information for.
//
// Returns *protocol.Hover which contains the type information to display.
// Returns error when the hover information cannot be found.
func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Hover request received", logger_domain.String("uri", params.TextDocument.URI.Filename()))

	document, err := s.workspace.RunAnalysisForURI(context.WithoutCancel(ctx), params.TextDocument.URI)
	if err != nil {
		l.Error("Hover: analysis failed", logger_domain.Error(err))
		return nil, nil
	}

	return document.GetHoverInfo(ctx, params.Position)
}
