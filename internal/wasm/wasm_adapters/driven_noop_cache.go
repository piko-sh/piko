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

package wasm_adapters

import (
	"context"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

// NoOpComponentCache implements ComponentCachePort without caching.
// Each call executes the loader function directly, which is suitable for WASM
// contexts where each generation request is independent.
type NoOpComponentCache struct{}

var _ annotator_domain.ComponentCachePort = (*NoOpComponentCache)(nil)

// NewNoOpComponentCache creates a new no-op component cache.
//
// Returns *NoOpComponentCache which is ready for use.
func NewNoOpComponentCache() *NoOpComponentCache {
	return &NoOpComponentCache{}
}

// GetOrSet always executes the loader function directly without
// caching.
//
// Takes ctx (context.Context) which is the request context.
// Takes loader (func(ctx context.Context)
// (*annotator_dto.ParsedComponent, error)) which fetches the
// component.
//
// Returns *annotator_dto.ParsedComponent which is the result from
// the loader.
// Returns error when the loader fails.
func (*NoOpComponentCache) GetOrSet(
	ctx context.Context,
	_ string,
	loader func(ctx context.Context) (*annotator_dto.ParsedComponent, error),
) (*annotator_dto.ParsedComponent, error) {
	return loader(ctx)
}

// Clear is a no-op since there is no cache to clear.
func (*NoOpComponentCache) Clear(_ context.Context) {
}
