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

package annotator_adapters

import (
	"context"
	"fmt"

	"github.com/maypok86/otter/v2"
	"github.com/maypok86/otter/v2/stats"
	"go.opentelemetry.io/otel/attribute"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// otterComponentCache is a driven adapter that implements ComponentCachePort
// using the high-performance Otter v2 in-memory cache. It uses Otter's loading
// mechanism to provide atomic, stampede-proof parsing of component files.
type otterComponentCache struct {
	// cache stores parsed components, keyed by their identifier string.
	cache *otter.Cache[string, *annotator_dto.ParsedComponent]
}

var _ annotator_domain.ComponentCachePort = (*otterComponentCache)(nil)

// GetOrSet retrieves a parsed component from the Otter cache.
//
// Takes key (string) which is the cache key for the component.
// Takes loader (func(...)) which provides the component if not cached.
//
// Returns *annotator_dto.ParsedComponent which is the cached or loaded component.
// Returns error when the loader function fails.
func (c *otterComponentCache) GetOrSet(
	ctx context.Context,
	key string,
	loader func(ctx context.Context) (*annotator_dto.ParsedComponent, error),
) (*annotator_dto.ParsedComponent, error) {
	ctx, span, l := log.
		With(logger_domain.String("cache.key", key)).
		Span(ctx, "otterComponentCache.GetOrSet")
	defer span.End()

	otterLoader := otter.LoaderFunc[string, *annotator_dto.ParsedComponent](
		func(ctx context.Context, key string) (*annotator_dto.ParsedComponent, error) {
			l.Internal("Component parse cache MISS. Executing loader function.", logger_domain.String("key", key))
			span.SetAttributes(attribute.String("cache.status", "MISS"))
			return loader(ctx)
		},
	)

	component, err := c.cache.Get(ctx, key, otterLoader)
	if err != nil {
		l.ReportError(span, err, "Loader function returned an error.", logger_domain.String("key", key))
		return nil, fmt.Errorf("loading component cache entry for key %q: %w", key, err)
	}

	l.Internal("Successfully retrieved component from cache.", logger_domain.String("key", key))
	return component, nil
}

// Clear removes all entries from the Otter cache.
func (c *otterComponentCache) Clear(ctx context.Context) {
	ctx, span, l := log.Span(ctx, "otterComponentCache.Clear")
	defer span.End()

	c.cache.InvalidateAll()
	l.Internal("Component parse cache cleared.")
}

// NewComponentCache creates a new component cache with default settings.
//
// Returns annotator_domain.ComponentCachePort which is ready for use.
func NewComponentCache() annotator_domain.ComponentCachePort {
	cache := otter.Must(&otter.Options[string, *annotator_dto.ParsedComponent]{
		MaximumSize:     2000,
		InitialCapacity: 100,
		StatsRecorder:   stats.NewCounter(),
	})

	return &otterComponentCache{
		cache: cache,
	}
}
