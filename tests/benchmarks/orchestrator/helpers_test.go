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

// This file provides fake (mock) implementations of external dependencies for use in tests and benchmarks.
// These fakes are designed to be simple, in-memory, and safe for concurrent use in parallel tests.
package orchestrator_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type FakeBlobStore struct {
	storage sync.Map
}

func NewFakeBlobStore() *FakeBlobStore {
	return &FakeBlobStore{}
}

func (f *FakeBlobStore) Put(_ context.Context, key string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.storage.Store(key, data)
	return nil
}

func (f *FakeBlobStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := f.storage.Load(key)
	if !ok {
		return nil, fmt.Errorf("blob not found: %s", key)
	}

	byteSlice, ok := data.([]byte)
	if !ok {
		return nil, fmt.Errorf("internal error: blob for key '%s' is not a []byte", key)
	}
	return io.NopCloser(bytes.NewReader(byteSlice)), nil
}

func (f *FakeBlobStore) Delete(_ context.Context, key string) error {
	f.storage.Delete(key)
	return nil
}

func (f *FakeBlobStore) Rename(_ context.Context, oldKey, newKey string) error {
	data, ok := f.storage.LoadAndDelete(oldKey)
	if !ok {
		return fmt.Errorf("blob not found: %s", oldKey)
	}
	f.storage.Store(newKey, data)
	return nil
}

func (f *FakeBlobStore) Exists(_ context.Context, key string) (bool, error) {
	_, ok := f.storage.Load(key)
	return ok, nil
}

func (f *FakeBlobStore) RangeGet(_ context.Context, key string, offset int64, length int64) (io.ReadCloser, error) {
	data, ok := f.storage.Load(key)
	if !ok {
		return nil, fmt.Errorf("blob not found: %s", key)
	}
	byteSlice, ok := data.([]byte)
	if !ok {
		return nil, fmt.Errorf("internal error: blob for key '%s' is not a []byte", key)
	}

	if offset >= int64(len(byteSlice)) {
		return nil, errors.New("offset beyond file size")
	}

	end := min(offset+length, int64(len(byteSlice)))

	rangeData := byteSlice[offset:end]
	return io.NopCloser(bytes.NewReader(rangeData)), nil
}

func (f *FakeBlobStore) ListKeys(_ context.Context) ([]string, error) {
	var keys []string
	f.storage.Range(func(key, _ any) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys, nil
}

type FakeCapabilityService struct{}

func (f *FakeCapabilityService) Register(name string, capabilityFunction capabilities_domain.CapabilityFunc) error {
	return nil
}

func (f *FakeCapabilityService) Execute(_ context.Context, _ string, input io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
	inputData, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	outputData := append([]byte("COMPILED: "), inputData...)
	return bytes.NewReader(outputData), nil
}

type lockedArtefact struct {
	meta registry_dto.ArtefactMeta
	mu   sync.RWMutex
}

type FakeRegistry struct {
	blobStore            registry_domain.BlobStore
	bus                  orchestrator_domain.EventBus
	CompiledVariantAdded chan string
	artefacts            sync.Map
}

func NewFakeRegistry(bus orchestrator_domain.EventBus, blobStore registry_domain.BlobStore) *FakeRegistry {
	return &FakeRegistry{
		blobStore:            blobStore,
		bus:                  bus,
		CompiledVariantAdded: make(chan string, 128),
	}
}

func (f *FakeRegistry) UpsertArtefact(ctx context.Context, artefactID string, sourcePath string, sourceData io.Reader, storageBackendID string, desiredProfiles []registry_dto.NamedProfile) (*registry_dto.ArtefactMeta, error) {
	data, err := io.ReadAll(sourceData)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("source/%s", artefactID)
	if err := f.blobStore.Put(ctx, key, bytes.NewReader(data)); err != nil {
		return nil, err
	}

	artefact := &lockedArtefact{
		meta: registry_dto.ArtefactMeta{
			ID:         artefactID,
			SourcePath: sourcePath,
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:        "source-variant",
					StorageBackendID: storageBackendID,
					StorageKey:       key,
				},
			},
			DesiredProfiles: desiredProfiles,
		},
	}
	f.artefacts.Store(artefact.meta.ID, artefact)

	event := orchestrator_domain.Event{
		Type:    registry_domain.EventArtefactCreated,
		Payload: map[string]any{"artefactID": artefact.meta.ID},
	}
	if f.bus != nil {
		if err := f.bus.Publish(ctx, "artefact.created", event); err != nil {
			return nil, err
		}
	}

	return new(artefact.meta), nil
}

func (f *FakeRegistry) GetArtefact(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
	value, ok := f.artefacts.Load(id)
	if !ok {
		return nil, fmt.Errorf("artefact not found: %s", id)
	}

	la, ok := value.(*lockedArtefact)
	if !ok {
		return nil, fmt.Errorf("internal error: artefact for id '%s' is not a *lockedArtefact", id)
	}
	la.mu.RLock()
	defer la.mu.RUnlock()

	return new(la.meta), nil
}

func (f *FakeRegistry) AddVariant(ctx context.Context, artefactID string, variant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
	value, ok := f.artefacts.Load(artefactID)
	if !ok {
		return nil, fmt.Errorf("artefact not found: %s", artefactID)
	}

	la, ok := value.(*lockedArtefact)
	if !ok {
		return nil, fmt.Errorf("internal error: artefact for id '%s' is not a *lockedArtefact", artefactID)
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	la.meta.ActualVariants = append(la.meta.ActualVariants, *variant)
	if variant.VariantID == "compiled-profile" {
		f.CompiledVariantAdded <- artefactID
	}

	return new(la.meta), nil
}

func (f *FakeRegistry) GetVariantData(_ context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
	return f.blobStore.Get(context.Background(), variant.StorageKey)
}

func (f *FakeRegistry) GetBlobStore(id string) (registry_domain.BlobStore, error) {
	return f.blobStore, nil
}

func (f *FakeRegistry) DeleteArtefact(ctx context.Context, artefactID string) error {
	f.artefacts.Delete(artefactID)
	return nil
}

func (f *FakeRegistry) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, errors.New("unimplemented")
}

func (f *FakeRegistry) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	return nil, errors.New("unimplemented")
}

func (f *FakeRegistry) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	return nil, errors.New("unimplemented")
}

func (f *FakeRegistry) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	return nil, errors.New("unimplemented")
}
func (f *FakeRegistry) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	return nil, errors.New("unimplemented")
}

func (f *FakeRegistry) GetVariantChunk(ctx context.Context, variant *registry_dto.Variant, chunkID string) (io.ReadCloser, error) {
	return nil, errors.New("unimplemented: chunks not supported in fake registry")
}

func (f *FakeRegistry) GetVariantDataRange(ctx context.Context, variant *registry_dto.Variant, offset int64, length int64) (io.ReadCloser, error) {
	return f.blobStore.RangeGet(ctx, variant.StorageKey, offset, length)
}

func (f *FakeRegistry) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, errors.New("unimplemented")
}

func (f *FakeRegistry) ArtefactEventsPublished() int64 {
	return 0
}

func (f *FakeRegistry) ListBlobStoreIDs() []string {
	return nil
}
