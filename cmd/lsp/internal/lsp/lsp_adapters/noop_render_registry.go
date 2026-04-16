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

package lsp_adapters

import (
	"context"
	"errors"
	"io"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/render/render_dto"
)

// ErrNoopRegistry is returned when operations are attempted on the noop
// registry which doesn't support actual registry operations.
var ErrNoopRegistry = errors.New("noop render registry: operation not supported in LSP mode")

// NoopRenderRegistry implements render_domain.RegistryPort with no-op methods,
// letting the LSP bootstrap without requiring database connectivity for the
// render registry, which is not needed for LSP operations.
type NoopRenderRegistry struct{}

// NewNoopRenderRegistry creates a new no-op render registry for LSP use.
//
// Returns *NoopRenderRegistry which provides no-op implementations of render
// methods.
func NewNoopRenderRegistry() *NoopRenderRegistry {
	return &NoopRenderRegistry{}
}

// GetComponentMetadata returns nil and an error as the noop registry cannot
// retrieve component metadata.
//
// Returns *render_dto.ComponentMetadata which is always nil.
// Returns error when called, as this is a noop implementation.
func (*NoopRenderRegistry) GetComponentMetadata(_ context.Context, _ string) (*render_dto.ComponentMetadata, error) {
	return nil, ErrNoopRegistry
}

// GetAssetRawSVG returns an error as the noop registry cannot retrieve SVG
// assets.
//
// Returns *render_domain.ParsedSvgData which is always nil for this
// implementation.
// Returns error when called, as the noop registry does not support asset
// retrieval.
func (*NoopRenderRegistry) GetAssetRawSVG(_ context.Context, _ string) (*render_domain.ParsedSvgData, error) {
	return nil, ErrNoopRegistry
}

// BulkGetAssetRawSVG returns an error as the noop registry cannot retrieve
// SVG assets.
//
// Returns map[string]*render_domain.ParsedSvgData which is always nil.
// Returns error when called, as this is a noop implementation.
func (*NoopRenderRegistry) BulkGetAssetRawSVG(_ context.Context, _ []string) (map[string]*render_domain.ParsedSvgData, error) {
	return nil, ErrNoopRegistry
}

// BulkGetComponentMetadata returns nil as the noop registry cannot retrieve
// component metadata.
//
// Returns map[string]*render_dto.ComponentMetadata which is always nil.
// Returns error when called, as this is a noop implementation.
func (*NoopRenderRegistry) BulkGetComponentMetadata(_ context.Context, _ []string) (map[string]*render_dto.ComponentMetadata, error) {
	return nil, ErrNoopRegistry
}

// GetStats returns empty statistics.
//
// Returns render_domain.RegistryAdapterStats which is always a zero value.
func (*NoopRenderRegistry) GetStats() render_domain.RegistryAdapterStats {
	return render_domain.RegistryAdapterStats{}
}

// ClearComponentCache is a no-op as there's no cache to clear.
func (*NoopRenderRegistry) ClearComponentCache(_ context.Context, _ string) {}

// ClearSvgCache is a no-op as there's no cache to clear.
func (*NoopRenderRegistry) ClearSvgCache(_ context.Context, _ string) {}

// GetArtefactServePath returns an empty string as the noop registry has no
// artefacts to serve.
//
// Returns string which is always empty for this implementation.
func (*NoopRenderRegistry) GetArtefactServePath(_ context.Context, _ string) string {
	return ""
}

// UpsertArtefact returns nil metadata as the noop registry cannot store
// artefacts.
//
// Returns *registry_dto.ArtefactMeta which is always nil.
// Returns error when called, as the noop registry does not support storage.
func (*NoopRenderRegistry) UpsertArtefact(
	_ context.Context,
	_ string,
	_ string,
	_ io.Reader,
	_ string,
	_ []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	return nil, ErrNoopRegistry
}

var _ render_domain.RegistryPort = (*NoopRenderRegistry)(nil)
