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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

const (
	invalidatorAssetArtefactID = "module/lib/icon.svg"
	invalidatorAssetURL        = "/_piko/assets/module/lib/icon.svg"
)

func buildArtefactWithBody(version string) *registry_dto.ArtefactMeta {
	tags := registry_dto.Tags{}
	tags.Set(registry_dto.TagEtag, "etag-"+version)
	return &registry_dto.ArtefactMeta{
		ID:     invalidatorAssetArtefactID,
		Status: registry_dto.VariantStatusReady,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:    variantSource,
				MimeType:     "image/svg+xml",
				StorageKey:   "store/" + version,
				SizeBytes:    int64(len("svg-body-" + version)),
				MetadataTags: tags,
				Status:       registry_dto.VariantStatusReady,
			},
		},
	}
}

func newSwitchableRegistry() (*registry_domain.MockRegistryService, *atomic.Pointer[registry_dto.ArtefactMeta]) {
	current := &atomic.Pointer[registry_dto.ArtefactMeta]{}
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
			art := current.Load()
			if art == nil || art.ID != artefactID {
				return nil, registry_domain.ErrArtefactNotFound
			}
			return art, nil
		},
		GetVariantDataFunc: func(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
			version := strings.TrimPrefix(variant.StorageKey, "store/")
			return io.NopCloser(strings.NewReader("svg-body-" + version)), nil
		},
	}
	return registry, current
}

func newRealArtefactCache(t *testing.T) cache_domain.Cache[string, *registry_dto.ArtefactMeta] {
	t.Helper()
	otterCache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
		Namespace:      "invalidator-test-artefact-metadata",
		MaximumEntries: 100,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = otterCache.Close(context.Background()) })
	return otterCache
}

func mountAssetHandler(
	t *testing.T,
	cache cache_domain.Cache[string, *registry_dto.ArtefactMeta],
	registry registry_domain.RegistryService,
) http.Handler {
	t.Helper()
	builder := &HTTPRouterBuilder{artefactCache: cache}
	t.Cleanup(builder.Close)
	router := chi.NewRouter()
	router.Get("/_piko/assets/*", builder.serveArtefact(registry, daemon_domain.OnDemandVariantGenerator(nil), true))
	return router
}

func getServedBody(t *testing.T, handler http.Handler, url string) (int, string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response.StatusCode, string(body)
}

func TestServeArtefact_WithoutInvalidation_ServesStaleBytes(t *testing.T) {
	t.Parallel()

	cache := newRealArtefactCache(t)
	registry, current := newSwitchableRegistry()
	current.Store(buildArtefactWithBody("v1"))

	handler := mountAssetHandler(t, cache, registry)

	status, body := getServedBody(t, handler, invalidatorAssetURL)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "svg-body-v1", body)

	current.Store(buildArtefactWithBody("v2"))

	status, body = getServedBody(t, handler, invalidatorAssetURL)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "svg-body-v1", body,
		"without invalidation the cache must keep serving the stale v1 bytes")
}

func TestServeArtefact_WithInvalidation_ServesFreshBytesAfterEvent(t *testing.T) {
	t.Parallel()

	cache := newRealArtefactCache(t)
	registry, current := newSwitchableRegistry()
	current.Store(buildArtefactWithBody("v1"))

	var handlerMu sync.Mutex
	handlers := map[string]orchestrator_domain.EventHandler{}
	eventBus := &orchestrator_domain.MockEventBus{
		SubscribeWithHandlerFunc: func(_ context.Context, topic string, handler orchestrator_domain.EventHandler) error {
			handlerMu.Lock()
			handlers[topic] = handler
			handlerMu.Unlock()
			return nil
		},
	}

	require.NoError(t, SubscribeArtefactMetadataInvalidation(context.Background(), eventBus, cache))

	assetHandler := mountAssetHandler(t, cache, registry)

	status, body := getServedBody(t, assetHandler, invalidatorAssetURL)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "svg-body-v1", body, "first request must serve v1 and populate the cache")

	current.Store(buildArtefactWithBody("v2"))

	handlerMu.Lock()
	updatedHandler, ok := handlers[registry_domain.TopicArtefactUpdated]
	handlerMu.Unlock()
	require.True(t, ok, "invalidator must subscribe to artefact.updated")

	require.NoError(t, updatedHandler(context.Background(), orchestrator_domain.Event{
		Type: registry_domain.EventArtefactUpdated,
		Payload: map[string]any{
			eventPayloadKeyArtefactID: invalidatorAssetArtefactID,
		},
	}))

	status, body = getServedBody(t, assetHandler, invalidatorAssetURL)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "svg-body-v2", body,
		"after the artefact.updated event the handler must serve the fresh v2 bytes, not the stale v1")
}

func TestSubscribeArtefactMetadataInvalidation_SubscribesAllArtefactTopics(t *testing.T) {
	t.Parallel()

	cache := newRealArtefactCache(t)
	var subscribed []string
	var mu sync.Mutex
	eventBus := &orchestrator_domain.MockEventBus{
		SubscribeWithHandlerFunc: func(_ context.Context, topic string, _ orchestrator_domain.EventHandler) error {
			mu.Lock()
			subscribed = append(subscribed, topic)
			mu.Unlock()
			return nil
		},
	}

	require.NoError(t, SubscribeArtefactMetadataInvalidation(context.Background(), eventBus, cache))

	require.ElementsMatch(t, registry_domain.ArtefactTopics, subscribed)
}

func TestSubscribeArtefactMetadataInvalidation_NilCacheIsNoOp(t *testing.T) {
	t.Parallel()

	eventBus := &orchestrator_domain.MockEventBus{}

	require.NoError(t, SubscribeArtefactMetadataInvalidation(context.Background(), eventBus, nil))
	require.Equal(t, int64(0), eventBus.SubscribeWithHandlerCallCount.Load())
}
