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

package orchestrator_adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

var _ orchestrator_domain.TaskExecutor = (*gcExecutor)(nil)

// ExecutorNameBlobGC is the executor name for blob garbage collection tasks.
const ExecutorNameBlobGC = "blob.gc"

const (
	// gcModeHints processes queued GC hints produced by variant
	// replacement and artefact deletion.
	gcModeHints = "hints"

	// gcModeOrphans scans blob storage for keys not referenced by any
	// artefact and removes them.
	gcModeOrphans = "orphans"

	// gcPayloadKeyMode is the payload key for the GC execution mode.
	gcPayloadKeyMode = "mode"

	// gcPayloadKeyBatchSize is the payload key for the maximum number of items
	// to process per GC run.
	gcPayloadKeyBatchSize = "batch_size"

	// gcPayloadKeyRescheduleSeconds is the payload key for the delay in seconds
	// before the next GC run is scheduled.
	gcPayloadKeyRescheduleSeconds = "reschedule_seconds"

	// gcDefaultBatchSize is the default maximum number of items to process per
	// GC run when no batch size is specified in the payload.
	gcDefaultBatchSize = 100

	// gcDefaultRescheduleSeconds is the default delay in seconds before the
	// next GC run when no reschedule interval is specified in the payload.
	gcDefaultRescheduleSeconds = 30

	// logKeyBackendID is the logging field for blob store backend identifiers.
	logKeyBackendID = "backend_id"

	// logKeyStorageKey is the logging field for blob storage keys.
	logKeyStorageKey = "storage_key"

	// logKeyMode is the logging field for the GC execution mode.
	logKeyMode = "mode"
)

// gcExecutor processes blob garbage collection tasks. It supports two modes:
// hint-based GC (fast, frequent) and orphan scanning (slower, infrequent).
type gcExecutor struct {
	// registryService provides access to GC hints, artefact metadata, and blob
	// stores.
	registryService registry_domain.RegistryService

	// orchestratorService is used to self-reschedule the next GC run.
	orchestratorService orchestrator_domain.OrchestratorService
}

// NewGCExecutor creates a new garbage collection executor.
//
// Takes registry (registry_domain.RegistryService) which provides access to
// GC hints, artefact metadata, and blob stores.
// Takes orchestrator (orchestrator_domain.OrchestratorService) which is used
// to self-reschedule the next GC run.
//
// Returns orchestrator_domain.TaskExecutor which executes GC tasks.
func NewGCExecutor(
	registry registry_domain.RegistryService,
	orchestrator orchestrator_domain.OrchestratorService,
) orchestrator_domain.TaskExecutor {
	return &gcExecutor{
		registryService:     registry,
		orchestratorService: orchestrator,
	}
}

// Execute runs a garbage collection task.
//
// Takes payload (map[string]any) which contains:
//   - "mode": "hints" or "orphans"
//   - "batch_size": max items to process per run (default 100)
//   - "reschedule_seconds": delay before next run (default 30)
//
// Returns map[string]any which contains "deleted_count" and "mode" fields.
// Returns error when the payload is invalid or the GC mode is unknown.
func (e *gcExecutor) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	ctx, l := logger_domain.From(ctx, log)

	mode := gcPayloadString(payload, gcPayloadKeyMode, gcModeHints)
	batchSize := gcPayloadInt(payload, gcPayloadKeyBatchSize, gcDefaultBatchSize)
	rescheduleSeconds := gcPayloadInt(payload, gcPayloadKeyRescheduleSeconds, gcDefaultRescheduleSeconds)

	l.Trace("Starting GC task",
		logger_domain.String(logKeyMode, mode),
		logger_domain.Int("batch_size", batchSize),
		logger_domain.Int("reschedule_seconds", rescheduleSeconds))

	var deletedCount int
	var err error

	switch mode {
	case gcModeHints:
		deletedCount, err = e.processHints(ctx, batchSize, rescheduleSeconds, payload)
	case gcModeOrphans:
		deletedCount, err = e.processOrphans(ctx, rescheduleSeconds, payload)
	default:
		return nil, orchestrator_domain.NewFatalError(fmt.Errorf("unknown GC mode: %q", mode))
	}

	if err != nil {
		return nil, err
	}

	l.Trace("GC task complete",
		logger_domain.String(logKeyMode, mode),
		logger_domain.Int("deleted_count", deletedCount))

	return map[string]any{
		"mode":          mode,
		"deleted_count": deletedCount,
	}, nil
}

// processHints pops GC hints, deletes the corresponding blobs, and
// reschedules the next run (immediately when a full batch is returned,
// otherwise after the configured delay).
//
// Takes batchSize (int) which is the maximum number of hints to pop.
// Takes rescheduleSeconds (int) which is the delay before the next run when
// fewer than batchSize hints are returned.
// Takes payload (map[string]any) which is forwarded to the rescheduled task.
//
// Returns int which is the number of blobs successfully deleted.
// Returns error when popping GC hints fails.
func (e *gcExecutor) processHints(
	ctx context.Context,
	batchSize int,
	rescheduleSeconds int,
	payload map[string]any,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)

	hints, err := e.registryService.PopGCHints(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("popping GC hints: %w", err)
	}

	deletedCount := 0
	for _, hint := range hints {
		store, storeErr := e.registryService.GetBlobStore(hint.BackendID)
		if storeErr != nil {
			l.Warn("Blob store not found for GC hint, skipping",
				logger_domain.String(logKeyBackendID, hint.BackendID),
				logger_domain.String(logKeyStorageKey, hint.StorageKey),
				logger_domain.Error(storeErr))
			continue
		}

		if deleteErr := store.Delete(ctx, hint.StorageKey); deleteErr != nil {
			l.Warn("Failed to delete blob for GC hint, skipping",
				logger_domain.String(logKeyBackendID, hint.BackendID),
				logger_domain.String(logKeyStorageKey, hint.StorageKey),
				logger_domain.Error(deleteErr))
			continue
		}

		deletedCount++
		l.Trace("Deleted blob via GC hint",
			logger_domain.String(logKeyBackendID, hint.BackendID),
			logger_domain.String(logKeyStorageKey, hint.StorageKey))
	}

	delay := time.Duration(rescheduleSeconds) * time.Second
	if len(hints) >= batchSize {
		delay = 0
	}

	e.reschedule(ctx, payload, delay)

	return deletedCount, nil
}

// processOrphans scans all blob stores for keys not referenced by any artefact
// and deletes them.
//
// Takes rescheduleSeconds (int) which is the delay before the next orphan scan
// is scheduled.
// Takes payload (map[string]any) which is forwarded to the rescheduled task.
//
// Returns int which is the total number of orphaned blobs deleted.
// Returns error when collecting referenced keys fails.
func (e *gcExecutor) processOrphans(
	ctx context.Context,
	rescheduleSeconds int,
	payload map[string]any,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)

	referencedKeys, err := e.collectReferencedKeys(ctx)
	if err != nil {
		return 0, fmt.Errorf("collecting referenced keys: %w", err)
	}

	l.Trace("Collected referenced storage keys",
		logger_domain.Int("referenced_key_count", len(referencedKeys)))

	deletedCount := 0
	for _, backendID := range e.registryService.ListBlobStoreIDs() {
		deleted := e.scanBlobStore(ctx, l, backendID, referencedKeys)
		deletedCount += deleted
	}

	delay := time.Duration(rescheduleSeconds) * time.Second
	e.reschedule(ctx, payload, delay)

	return deletedCount, nil
}

// scanBlobStore scans a single blob store for orphaned keys and deletes them.
//
// Takes l (logger_domain.Logger) which logs warnings and trace messages.
// Takes backendID (string) which identifies the blob store to scan.
// Takes referencedKeys (map[string]struct{}) which holds the set of storage
// keys still referenced by artefact variants.
//
// Returns int which is the number of orphaned blobs deleted from this store.
func (e *gcExecutor) scanBlobStore(
	ctx context.Context,
	l logger_domain.Logger,
	backendID string,
	referencedKeys map[string]struct{},
) int {
	store, storeErr := e.registryService.GetBlobStore(backendID)
	if storeErr != nil {
		l.Warn("Failed to get blob store for orphan scan",
			logger_domain.String(logKeyBackendID, backendID),
			logger_domain.Error(storeErr))
		return 0
	}

	keys, listErr := store.ListKeys(ctx)
	if listErr != nil {
		l.Trace("Blob store does not support key listing, skipping orphan scan",
			logger_domain.String(logKeyBackendID, backendID))
		return 0
	}

	deletedCount := 0
	for _, key := range keys {
		if strings.HasPrefix(key, "tmp/") {
			continue
		}
		if _, referenced := referencedKeys[key]; referenced {
			continue
		}

		if deleteErr := store.Delete(ctx, key); deleteErr != nil {
			l.Warn("Failed to delete orphaned blob",
				logger_domain.String(logKeyBackendID, backendID),
				logger_domain.String(logKeyStorageKey, key),
				logger_domain.Error(deleteErr))
			continue
		}

		deletedCount++
		l.Trace("Deleted orphaned blob",
			logger_domain.String(logKeyBackendID, backendID),
			logger_domain.String(logKeyStorageKey, key))
	}
	return deletedCount
}

// collectReferencedKeys builds a set of all storage keys that are currently
// referenced by artefact variants and their chunks.
//
// Returns map[string]struct{} which is the set of referenced storage keys.
// Returns error when listing or fetching artefacts fails.
func (e *gcExecutor) collectReferencedKeys(ctx context.Context) (map[string]struct{}, error) {
	ids, err := e.registryService.ListAllArtefactIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing artefact IDs: %w", err)
	}

	artefacts, err := e.registryService.GetMultipleArtefacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetching artefacts: %w", err)
	}

	referenced := make(map[string]struct{}, len(artefacts)*2)
	for _, artefact := range artefacts {
		if artefact == nil {
			continue
		}
		collectArtefactKeys(artefact, referenced)
	}

	return referenced, nil
}

// collectArtefactKeys adds all storage keys from an artefact's variants and
// chunks to the referenced set.
//
// Takes artefact (*registry_dto.ArtefactMeta) which provides the variants and
// chunks whose storage keys should be collected.
// Takes referenced (map[string]struct{}) which is the set to add the storage
// keys to.
func collectArtefactKeys(artefact *registry_dto.ArtefactMeta, referenced map[string]struct{}) {
	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		if v.StorageKey != "" {
			referenced[v.StorageKey] = struct{}{}
		}
		for j := range v.Chunks {
			if v.Chunks[j].StorageKey != "" {
				referenced[v.Chunks[j].StorageKey] = struct{}{}
			}
		}
	}
}

// reschedule creates and schedules the next GC task with the same payload.
//
// Takes payload (map[string]any) which is forwarded as the rescheduled task's
// payload.
// Takes delay (time.Duration) which is the delay before the rescheduled task
// executes.
func (e *gcExecutor) reschedule(ctx context.Context, payload map[string]any, delay time.Duration) {
	_, l := logger_domain.From(ctx, log)

	mode := gcPayloadString(payload, gcPayloadKeyMode, gcModeHints)

	task := orchestrator_domain.NewTask(ExecutorNameBlobGC, payload)
	task.DeduplicationKey = "blob.gc." + mode
	task.Config.Priority = orchestrator_domain.PriorityLow

	executeAt := time.Now().Add(delay)
	if _, err := e.orchestratorService.Schedule(ctx, task, executeAt); err != nil {
		l.Warn("Failed to reschedule GC task",
			logger_domain.String(logKeyMode, mode),
			logger_domain.Error(err))
	}
}

// gcPayloadString extracts a string value from the payload with a fallback.
//
// Takes payload (map[string]any) which holds the raw task data.
// Takes key (string) which is the payload key to look up.
// Takes fallback (string) which is returned when the key is missing or not a
// string.
//
// Returns string which is the extracted value or the fallback.
func gcPayloadString(payload map[string]any, key string, fallback string) string {
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// gcPayloadInt extracts an integer value from the payload with a fallback.
// Handles both int and float64 (JSON deserialisation produces float64).
//
// Takes payload (map[string]any) which holds the raw task data.
// Takes key (string) which is the payload key to look up.
// Takes fallback (int) which is returned when the key is missing or has an
// unsupported type.
//
// Returns int which is the extracted value or the fallback.
func gcPayloadInt(payload map[string]any, key string, fallback int) int {
	v, ok := payload[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return fallback
	}
}
