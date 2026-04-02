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

package annotator_domain

import (
	"context"

	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// TypeInspectorBuilderAdapter adapts inspector_domain.TypeBuilder to implement
// TypeInspectorBuilderPort. The AnnotatorService can then depend on an
// interface rather than a concrete type, improving testability.
type TypeInspectorBuilderAdapter struct {
	// builder provides the underlying type inspection implementation.
	builder *inspector_domain.TypeBuilder
}

var _ TypeInspectorBuilderPort = (*TypeInspectorBuilderAdapter)(nil)

// NewTypeInspectorBuilderAdapter creates a new adapter wrapping the given
// TypeBuilder.
//
// Takes builder (*inspector_domain.TypeBuilder) which is the builder to wrap.
//
// Returns *TypeInspectorBuilderAdapter which implements
// TypeInspectorBuilderPort.
func NewTypeInspectorBuilderAdapter(builder *inspector_domain.TypeBuilder) *TypeInspectorBuilderAdapter {
	return &TypeInspectorBuilderAdapter{builder: builder}
}

// SetConfig configures the type inspector with base directory and module info.
//
// Takes config (inspector_dto.Config) which specifies the base directory and
// module information.
func (a *TypeInspectorBuilderAdapter) SetConfig(config inspector_dto.Config) {
	a.builder.SetConfig(config)
}

// Build processes Go source files to build type information.
//
// Takes sourceOverlay (map[string][]byte) which provides in-memory file
// contents that override files on disk.
// Takes scriptHashes (map[string]string) which maps script paths to their
// content hashes for cache invalidation.
//
// Returns error when the underlying builder fails to process the sources.
func (a *TypeInspectorBuilderAdapter) Build(ctx context.Context, sourceOverlay map[string][]byte, scriptHashes map[string]string) error {
	return a.builder.Build(ctx, sourceOverlay, scriptHashes)
}

// GetQuerier returns the type querier after Build completes successfully.
// The returned TypeInspectorPort is the underlying *TypeQuerier which
// implements the interface.
//
// Returns TypeInspectorPort which provides type inspection capabilities.
// Returns bool which indicates whether a valid querier is available.
func (a *TypeInspectorBuilderAdapter) GetQuerier() (TypeInspectorPort, bool) {
	querier, ok := a.builder.GetQuerier()
	if !ok {
		return nil, false
	}
	return querier, true
}
