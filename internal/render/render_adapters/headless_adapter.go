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

package render_adapters

import (
	"context"

	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/wasm/wasm_domain"
)

// HeadlessRendererAdapter adapts RenderOrchestrator to implement
// wasm_domain.HeadlessRendererPort. It allows the WASM module to use the main
// render orchestrator for headless rendering without HTTP context.
type HeadlessRendererAdapter struct {
	// orchestrator provides the rendering logic for converting AST to string.
	orchestrator *render_domain.RenderOrchestrator
}

var _ wasm_domain.HeadlessRendererPort = (*HeadlessRendererAdapter)(nil)

// NewHeadlessRendererAdapter creates a new headless renderer adapter.
//
// Takes orchestrator (*render_domain.RenderOrchestrator) which provides the
// rendering implementation.
//
// Returns *HeadlessRendererAdapter which implements
// wasm_domain.HeadlessRendererPort.
func NewHeadlessRendererAdapter(orchestrator *render_domain.RenderOrchestrator) *HeadlessRendererAdapter {
	return &HeadlessRendererAdapter{orchestrator: orchestrator}
}

// RenderASTToString renders an AST to an HTML string without HTTP context.
//
// Takes opts (wasm_domain.HeadlessRenderOptions) which configures the
// rendering.
//
// Returns string which contains the rendered HTML.
// Returns error when rendering fails.
func (a *HeadlessRendererAdapter) RenderASTToString(
	ctx context.Context,
	opts wasm_domain.HeadlessRenderOptions,
) (string, error) {
	return a.orchestrator.RenderASTToString(ctx, render_domain.RenderASTToStringOptions{
		Template:               opts.Template,
		Metadata:               opts.Metadata,
		Styling:                opts.Styling,
		IncludeDocumentWrapper: opts.IncludeDocumentWrapper,
	})
}
