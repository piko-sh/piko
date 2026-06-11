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

package daemon_adapters

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	// eventPayloadKeyArtefactID is the payload key holding the affected artefact ID on
	// registry artefact lifecycle events.
	eventPayloadKeyArtefactID = "artefactID"
)

// SubscribeArtefactMetadataInvalidation wires the shared artefact-metadata cache to the
// registry artefact lifecycle events so that an updated, created, or deleted artefact is
// evicted from the cache straight away rather than lingering until the cache TTL expires.
//
// Without this subscription the daemon serves stale variant metadata for an asset (for
// example an svg that was rewritten on disk) until the metadata cache entry naturally
// expires, because the asset-serving handler returns the cached ArtefactMeta on a cache
// hit without re-reading the registry.
//
// Takes eventBus (orchestrator_domain.EventBus) which delivers registry artefact events.
// Takes cache (cache_domain.Cache) which is the shared artefact-metadata cache to evict
// from; a nil cache disables invalidation and returns nil.
//
// Returns error when subscribing to any artefact topic fails.
func SubscribeArtefactMetadataInvalidation(
	ctx context.Context,
	eventBus orchestrator_domain.EventBus,
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta],
) error {
	ctx, l := logger_domain.From(ctx, log)

	if eventBus == nil || cache == nil {
		l.Internal("Artefact metadata cache invalidation not wired",
			logger_domain.Bool("hasEventBus", eventBus != nil),
			logger_domain.Bool("hasCache", cache != nil))
		return nil
	}

	handler := newArtefactMetadataInvalidationHandler(cache)
	for _, topic := range registry_domain.ArtefactTopics {
		if err := eventBus.SubscribeWithHandler(ctx, topic, handler); err != nil {
			return fmt.Errorf("subscribing artefact metadata invalidation to topic %q: %w", topic, err)
		}
	}

	l.Internal("Artefact metadata cache invalidation subscribed",
		logger_domain.Strings("topics", registry_domain.ArtefactTopics))
	return nil
}

// newArtefactMetadataInvalidationHandler builds an event handler that evicts the affected
// artefact from the shared metadata cache on each registry artefact lifecycle event.
//
// Takes cache (cache_domain.Cache) which is the shared artefact-metadata cache to evict
// from.
//
// Returns orchestrator_domain.EventHandler which invalidates the affected artefact ID.
func newArtefactMetadataInvalidationHandler(
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta],
) orchestrator_domain.EventHandler {
	return func(eventCtx context.Context, event orchestrator_domain.Event) error {
		eventCtx, l := logger_domain.From(eventCtx, log)

		artefactID, ok := event.Payload[eventPayloadKeyArtefactID].(string)
		if !ok || artefactID == "" {
			l.Trace("Artefact event without artefact ID, skipping cache invalidation",
				logger_domain.String("eventType", string(event.Type)))
			return nil
		}

		if err := cache.Invalidate(eventCtx, artefactID); err != nil {
			l.Warn("Failed to invalidate artefact metadata cache entry",
				logger_domain.String("artefactID", artefactID),
				logger_domain.String("eventType", string(event.Type)),
				logger_domain.Error(err))
			return nil
		}

		l.Trace("Invalidated artefact metadata cache entry",
			logger_domain.String("artefactID", artefactID),
			logger_domain.String("eventType", string(event.Type)))
		return nil
	}
}
