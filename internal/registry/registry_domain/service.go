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

package registry_domain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// EventArtefactCreated is the event type sent when a new artefact is created.
	EventArtefactCreated orchestrator_domain.EventType = "artefact.created"

	// EventArtefactUpdated is the event type published when an artefact is modified.
	EventArtefactUpdated orchestrator_domain.EventType = "artefact.updated"

	// EventArtefactDeleted is the event type sent when an artefact is deleted.
	EventArtefactDeleted orchestrator_domain.EventType = "artefact.deleted"

	// TopicArtefactCreated is the event topic for when an artefact is created.
	TopicArtefactCreated = "artefact.created"

	// TopicArtefactUpdated is the event topic for artefact update notifications.
	TopicArtefactUpdated = "artefact.updated"

	// TopicArtefactDeleted is the event topic for when an artefact is deleted.
	TopicArtefactDeleted = "artefact.deleted"
)

// ArtefactTopics contains all artefact event topics. Use this when
// subscribing to all artefact events at once.
var ArtefactTopics = []string{
	TopicArtefactCreated,
	TopicArtefactUpdated,
	TopicArtefactDeleted,
}

// registryService implements RegistryService and provides artefact management
// with metadata storage, blob stores, and event publishing.
type registryService struct {
	// metaStore stores artefact metadata.
	metaStore MetadataStore

	// blobStores maps storage backend IDs to their blob store instances.
	blobStores map[string]BlobStore

	// eventBus publishes lifecycle events when artefacts change.
	eventBus orchestrator_domain.EventBus

	// cache stores often-used metadata to reduce database queries.
	cache MetadataCache

	// loader deduplicates concurrent artefact load requests.
	loader singleflight.Group

	// artefactEventsPublished tracks the number of artefact events published.
	// Used for pipeline flush detection: the daemon waits until the bridge
	// has handled all published events before checking if idle.
	artefactEventsPublished atomic.Int64
}

// All service methods are implemented in separate files for better organisation:
// - artefact_lifecycle.go: UpsertArtefact, AddVariant, DeleteArtefact, processBlobUpdate, publishEvent, helper functions
// - artefact_retrieval.go: GetArtefact, GetMultipleArtefacts, ListAllArtefactIDs, SearchArtefacts, SearchArtefactsByTagValues, FindArtefactByVariantStorageKey
// - blob_operations.go: GetVariantData, GetBlobStore, PopGCHints

var _ RegistryService = (*registryService)(nil)

// Name returns the service identifier for health probe reporting.
// Implements the healthprobe_domain.Probe interface.
//
// Returns string which is the constant "RegistryService".
func (*registryService) Name() string {
	return "RegistryService"
}

// Check implements the healthprobe_domain.Probe interface.
//
// It verifies that the metadata store and blob stores are accessible
// and responsive.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which indicates the health state of the
// service.
func (s *registryService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime)
	}

	return s.checkReadiness(ctx, checkType, startTime)
}

// ArtefactEventsPublished returns the number of artefact events published.
// Used for pipeline flush detection - the daemon waits until the bridge
// has processed this many events before checking if the dispatcher is idle.
//
// Returns int64 which is the count of artefact events published so far.
func (s *registryService) ArtefactEventsPublished() int64 {
	return s.artefactEventsPublished.Load()
}

// checkLiveness performs a basic liveness check verifying the service is
// initialised.
//
// Takes startTime (time.Time) which marks when the check began for duration
// calculation.
//
// Returns healthprobe_dto.Status which contains the health state and message.
func (s *registryService) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "Registry service is running"

	if s.metaStore == nil {
		state = healthprobe_dto.StateUnhealthy
		message = "Metadata store is not initialised"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// checkReadiness performs a full readiness check on the metadata store and all
// blob stores.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes startTime (time.Time) which marks when the check began for duration
// calculation.
//
// Returns healthprobe_dto.Status which contains the overall readiness state
// and status of all dependencies.
func (s *registryService) checkReadiness(
	ctx context.Context,
	checkType healthprobe_dto.CheckType,
	startTime time.Time,
) healthprobe_dto.Status {
	dependencies := make([]*healthprobe_dto.Status, 0, len(s.blobStores)+1)
	overallState := healthprobe_dto.StateHealthy

	metaStoreStatus := s.checkMetadataStore(ctx, startTime)
	dependencies = append(dependencies, &metaStoreStatus)
	if metaStoreStatus.State == healthprobe_dto.StateUnhealthy {
		overallState = healthprobe_dto.StateUnhealthy
	}

	blobStoreIDs := make([]string, 0, len(s.blobStores))
	for id := range s.blobStores {
		blobStoreIDs = append(blobStoreIDs, id)
	}
	slices.Sort(blobStoreIDs)

	for _, backendID := range blobStoreIDs {
		blobStore := s.blobStores[backendID]
		blobStatus, blobState := s.checkBlobStore(ctx, checkType, backendID, blobStore)
		dependencies = append(dependencies, &blobStatus)
		overallState = aggregateState(overallState, blobState)
	}

	message := fmt.Sprintf("Registry operational with %d blob store(s)", len(s.blobStores))
	if overallState != healthprobe_dto.StateHealthy {
		message = "Registry has storage issues"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        overallState,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// checkMetadataStore checks that the metadata store can be reached.
//
// Takes startTime (time.Time) which marks when the health check began.
//
// Returns healthprobe_dto.Status which shows the health of the store.
func (s *registryService) checkMetadataStore(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	checkCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second,
		errors.New("registry health check exceeded 2s timeout"))
	defer cancel()

	_, err := s.metaStore.ListAllArtefactIDs(checkCtx)

	status := healthprobe_dto.Status{
		Name:         "MetadataStore",
		State:        healthprobe_dto.StateHealthy,
		Message:      "Metadata store responsive",
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}

	if err != nil {
		status.State = healthprobe_dto.StateUnhealthy
		status.Message = fmt.Sprintf("Metadata store query failed: %v", err)
	}

	return status
}

// checkBlobStore verifies a single blob store is accessible.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes backendID (string) which identifies the blob store backend.
// Takes blobStore (BlobStore) which is the store to check.
//
// Returns healthprobe_dto.Status which contains the health check result.
// Returns healthprobe_dto.State which indicates the overall health state.
func (*registryService) checkBlobStore(
	ctx context.Context,
	checkType healthprobe_dto.CheckType,
	backendID string,
	blobStore BlobStore,
) (healthprobe_dto.Status, healthprobe_dto.State) {
	probe, ok := blobStore.(interface {
		Name() string
		Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
	})
	if !ok {
		return healthprobe_dto.Status{
			Name:         fmt.Sprintf("BlobStore (%s)", backendID),
			State:        healthprobe_dto.StateHealthy,
			Message:      "Blob store does not support health checks (skipped)",
			Timestamp:    time.Now(),
			Duration:     "",
			Dependencies: nil,
		}, healthprobe_dto.StateHealthy
	}

	status := probe.Check(ctx, checkType)
	return status, status.State
}

// NewRegistryService creates a new registry service with the given dependencies.
// The service manages artefact lifecycle, blob storage, and event publishing.
//
// Takes metaStore (MetadataStore) which provides access to artefact metadata.
// Takes blobStores (map[string]BlobStore) which maps storage names to blob
// stores for artefact data.
// Takes eventBus (orchestrator_domain.EventBus) which publishes artefact
// lifecycle events.
// Takes cache (MetadataCache) which caches metadata lookups for performance.
//
// Returns RegistryService which is ready to manage registry operations.
func NewRegistryService(
	metaStore MetadataStore,
	blobStores map[string]BlobStore,
	eventBus orchestrator_domain.EventBus,
	cache MetadataCache,
) RegistryService {
	return &registryService{
		metaStore:               metaStore,
		blobStores:              blobStores,
		eventBus:                eventBus,
		cache:                   cache,
		loader:                  singleflight.Group{},
		artefactEventsPublished: atomic.Int64{},
	}
}

// aggregateState combines health states, preferring the worst state.
//
// Takes current (healthprobe_dto.State) which is the existing aggregate state.
// Takes incoming (healthprobe_dto.State) which is the new state to merge.
//
// Returns healthprobe_dto.State which is the combined state, where unhealthy
// takes precedence over degraded, which takes precedence over healthy.
func aggregateState(current, incoming healthprobe_dto.State) healthprobe_dto.State {
	if incoming == healthprobe_dto.StateUnhealthy {
		return healthprobe_dto.StateUnhealthy
	}
	if incoming == healthprobe_dto.StateDegraded && current == healthprobe_dto.StateHealthy {
		return healthprobe_dto.StateDegraded
	}
	return current
}
