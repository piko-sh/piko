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

package bootstrap

// This file provides integration of the collection system into the Piko build pipeline.

import (
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
	_ "piko.sh/piko/internal/collection/collection_adapters/cache_factory" // registers hybrid-collections cache blueprint
	"piko.sh/piko/internal/collection/collection_adapters/driver_markdown"
	"piko.sh/piko/internal/collection/collection_adapters/driver_registry"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/wdk/safedisk"
)

// createCollectionService creates and sets up the collection service with
// its providers. Call this once during bootstrap.
//
// The collection service handles:
//   - Processing GetCollection() calls found by the type resolver
//   - Working with collection providers (markdown, headless CMS, and others)
//   - Creating Go code annotations for the generator
//
// Takes c (*Container) which gives access to settings and dependencies.
//
// Returns collection_domain.CollectionService which is the ready service for
// processing collection requests.
// Returns error when sandbox creation fails or provider setup fails.
func createCollectionService(c *Container) (collection_domain.CollectionService, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating collection service...")

	serverConfig := c.config.ServerConfig

	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	sandboxFactory, err := c.GetSandboxFactory()
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox factory: %w", err)
	}
	contentSandbox, err := sandboxFactory.Create("collection-content", baseDir, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to create content sandbox: %w", err)
	}

	registry := driver_registry.NewMemoryRegistry()

	markdownParser := c.GetMarkdownParser()

	if markdownParser != nil {
		highlighter := c.GetHighlighter()
		renderer := c.GetRenderer()

		resolver, err := c.GetResolver()
		if err != nil {
			return nil, fmt.Errorf("failed to get resolver for collection service: %w", err)
		}

		mdService := markdown_domain.NewMarkdownService(markdownParser, highlighter)
		mdProvider := driver_markdown.NewMarkdownProvider(
			"markdown",
			contentSandbox,
			mdService,
			renderer,
			driver_markdown.WithModuleResolver(resolver),
		)

		if err := registry.Register(mdProvider); err != nil {
			return nil, fmt.Errorf("failed to register markdown provider: %w", err)
		}

		if highlighter != nil {
			l.Internal("Registered markdown collection provider with syntax highlighting")
		} else {
			l.Internal("Registered markdown collection provider")
		}
	} else {
		l.Internal("No markdown parser configured - skipping markdown collection provider")
	}

	collectionService := collection_domain.NewCollectionService(c.GetAppContext(), registry)

	l.Internal("Collection service initialised")

	return collectionService, nil
}

// initHybridCache swaps the hybrid collection cache from the startup otter
// instance (created during package init) to a service-managed instance so it
// appears in the dev widget's cache panel.
//
// Takes c (*Container) which provides access to the cache service.
func initHybridCache(c *Container) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	cacheService, err := c.GetCacheService()
	if err != nil {
		l.Warn("Cache service not available, hybrid cache will use startup otter instance",
			logger_domain.Error(err))
		return
	}

	hybridCache, err := cache_domain.NewCacheBuilder[string, collection_domain.HybridCacheValue](cacheService).
		FactoryBlueprint("hybrid-collections").
		Namespace("hybrid-collections").
		MaximumSize(10_000).
		Build(c.GetAppContext())
	if err != nil {
		l.Warn("Failed to create service-managed hybrid cache, continuing with startup instance",
			logger_domain.Error(err))
		return
	}

	collection_domain.InitHybridCache(hybridCache)
	l.Internal("Hybrid collection cache migrated to cache service")
}
