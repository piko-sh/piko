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

package asset_resolver

import (
	"context"
	"io"
	"sync/atomic"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type mockRegistryService struct {
	GetArtefactFunc         func(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)
	GetVariantDataFunc      func(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error)
	GetArtefactCallCount    int64
	GetVariantDataCallCount int64
}

var _ registry_domain.RegistryService = (*mockRegistryService)(nil)

func (m *mockRegistryService) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.GetArtefactCallCount, 1)
	if m.GetArtefactFunc != nil {
		return m.GetArtefactFunc(ctx, artefactID)
	}
	return nil, nil
}

func (m *mockRegistryService) GetVariantData(ctx context.Context, variant *registry_dto.Variant) (io.ReadCloser, error) {
	atomic.AddInt64(&m.GetVariantDataCallCount, 1)
	if m.GetVariantDataFunc != nil {
		return m.GetVariantDataFunc(ctx, variant)
	}
	return nil, nil
}

func (*mockRegistryService) UpsertArtefact(
	_ context.Context, _ string, _ string, _ io.Reader, _ string, _ []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) AddVariant(_ context.Context, _ string, _ *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) DeleteArtefact(_ context.Context, _ string) error {
	return nil
}

func (*mockRegistryService) GetMultipleArtefacts(_ context.Context, _ []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) ListAllArtefactIDs(_ context.Context) ([]string, error) {
	return nil, nil
}

func (*mockRegistryService) SearchArtefacts(_ context.Context, _ registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) SearchArtefactsByTagValues(_ context.Context, _ string, _ []string) ([]*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) FindArtefactByVariantStorageKey(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) GetVariantChunk(_ context.Context, _ *registry_dto.Variant, _ string) (io.ReadCloser, error) {
	return nil, nil
}

func (*mockRegistryService) GetVariantDataRange(_ context.Context, _ *registry_dto.Variant, _ int64, _ int64) (io.ReadCloser, error) {
	return nil, nil
}

func (*mockRegistryService) GetBlobStore(_ string) (registry_domain.BlobStore, error) {
	return nil, nil
}

func (*mockRegistryService) PopGCHints(_ context.Context, _ int) ([]registry_dto.GCHint, error) {
	return nil, nil
}

func (*mockRegistryService) ArtefactEventsPublished() int64 {
	return 0
}

func (*mockRegistryService) ListBlobStoreIDs() []string {
	return nil
}

func makeVariant(variantID string, mimeType string, tags registry_dto.Tags) registry_dto.Variant {
	return registry_dto.Variant{
		VariantID:    variantID,
		MimeType:     mimeType,
		StorageKey:   "storage/" + variantID,
		MetadataTags: tags,
		SizeBytes:    1024,
	}
}

func makeTags(width string, density string) registry_dto.Tags {
	var tags registry_dto.Tags
	if width != "" {
		tags.Set(registry_dto.TagWidth, width)
	}
	if density != "" {
		tags.Set(registry_dto.TagDensity, density)
	}
	return tags
}
