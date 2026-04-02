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

package seo_adapters

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type mockRegistryService struct {
	UpsertArtefactFunc func(
		ctx context.Context,
		artefactID string,
		sourcePath string,
		sourceData io.Reader,
		storageBackendID string,
		desiredProfiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error)
}

func (m *mockRegistryService) UpsertArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	sourceData io.Reader,
	storageBackendID string,
	desiredProfiles []registry_dto.NamedProfile,
) (*registry_dto.ArtefactMeta, error) {
	if m.UpsertArtefactFunc != nil {
		return m.UpsertArtefactFunc(ctx, artefactID, sourcePath, sourceData, storageBackendID, desiredProfiles)
	}
	return &registry_dto.ArtefactMeta{}, nil
}

func (*mockRegistryService) AddVariant(
	_ context.Context,
	_ string,
	_ *registry_dto.Variant,
) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
}

func (*mockRegistryService) DeleteArtefact(_ context.Context, _ string) error {
	return nil
}

func (*mockRegistryService) GetArtefact(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
	return nil, nil
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

func (*mockRegistryService) GetVariantData(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
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

func TestRegistryStorageAdapter_StoreSitemap_Success(t *testing.T) {
	t.Parallel()

	var capturedArtefactID string
	var capturedContent []byte
	var capturedProfiles []registry_dto.NamedProfile
	var capturedStorageBackendID string

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID string,
			_ string,
			sourceData io.Reader,
			storageBackendID string,
			desiredProfiles []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			capturedArtefactID = artefactID
			capturedStorageBackendID = storageBackendID
			capturedProfiles = desiredProfiles

			data, readError := io.ReadAll(sourceData)
			require.NoError(t, readError)
			capturedContent = data

			return &registry_dto.ArtefactMeta{ID: artefactID}, nil
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	sitemapContent := []byte(`<?xml version="1.0" encoding="UTF-8"?><urlset></urlset>`)
	profiles := []registry_dto.NamedProfile{
		{Name: "gzip"},
		{Name: "brotli"},
	}

	err := adapter.StoreSitemap(context.Background(), "sitemap.xml", sitemapContent, profiles)
	require.NoError(t, err)

	assert.Equal(t, "sitemap.xml", capturedArtefactID)
	assert.Equal(t, sitemapContent, capturedContent)
	assert.Equal(t, "default", capturedStorageBackendID)
	assert.Len(t, capturedProfiles, 2)
}

func TestRegistryStorageAdapter_StoreSitemap_Error(t *testing.T) {
	t.Parallel()

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			_ string,
			_ string,
			_ io.Reader,
			_ string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("storage unavailable")
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	err := adapter.StoreSitemap(context.Background(), "sitemap.xml", []byte("<urlset/>"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upserting sitemap artefact")
	assert.Contains(t, err.Error(), "storage unavailable")
}

func TestRegistryStorageAdapter_StoreSitemap_NilProfiles(t *testing.T) {
	t.Parallel()

	var capturedProfiles []registry_dto.NamedProfile

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			_ string,
			_ string,
			_ io.Reader,
			_ string,
			desiredProfiles []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			capturedProfiles = desiredProfiles
			return &registry_dto.ArtefactMeta{}, nil
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	err := adapter.StoreSitemap(context.Background(), "sitemap.xml", []byte("<urlset/>"), nil)
	require.NoError(t, err)
	assert.Nil(t, capturedProfiles)
}

func TestRegistryStorageAdapter_StoreRobotsTxt_Success(t *testing.T) {
	t.Parallel()

	var capturedArtefactID string
	var capturedContent []byte
	var capturedProfiles []registry_dto.NamedProfile

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID string,
			_ string,
			sourceData io.Reader,
			_ string,
			desiredProfiles []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			capturedArtefactID = artefactID
			capturedProfiles = desiredProfiles

			data, readError := io.ReadAll(sourceData)
			require.NoError(t, readError)
			capturedContent = data

			return &registry_dto.ArtefactMeta{ID: artefactID}, nil
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	robotsContent := []byte("User-agent: *\nAllow: /\n")

	err := adapter.StoreRobotsTxt(context.Background(), robotsContent)
	require.NoError(t, err)

	assert.Equal(t, "robots.txt", capturedArtefactID)
	assert.Equal(t, robotsContent, capturedContent)
	assert.Nil(t, capturedProfiles)
}

func TestRegistryStorageAdapter_StoreRobotsTxt_Error(t *testing.T) {
	t.Parallel()

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			_ string,
			_ string,
			_ io.Reader,
			_ string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("disk full")
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	err := adapter.StoreRobotsTxt(context.Background(), []byte("User-agent: *\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upserting robots.txt artefact")
	assert.Contains(t, err.Error(), "disk full")
}

func TestRegistryStorageAdapter_StoreRobotsTxt_EmptyContent(t *testing.T) {
	t.Parallel()

	var capturedContent []byte

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			_ string,
			_ string,
			sourceData io.Reader,
			_ string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			data, readError := io.ReadAll(sourceData)
			require.NoError(t, readError)
			capturedContent = data
			return &registry_dto.ArtefactMeta{}, nil
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	err := adapter.StoreRobotsTxt(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, capturedContent)
}

func TestRegistryStorageAdapter_StoreSitemap_EmptyContent(t *testing.T) {
	t.Parallel()

	var capturedContent []byte

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			_ string,
			_ string,
			sourceData io.Reader,
			_ string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			data, readError := io.ReadAll(sourceData)
			require.NoError(t, readError)
			capturedContent = data
			return &registry_dto.ArtefactMeta{}, nil
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	err := adapter.StoreSitemap(context.Background(), "sitemap.xml", []byte{}, nil)
	require.NoError(t, err)
	assert.Empty(t, capturedContent)
}

func TestRegistryStorageAdapter_StoreSitemap_CustomArtefactID(t *testing.T) {
	t.Parallel()

	var capturedArtefactID string

	mock := &mockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID string,
			_ string,
			_ io.Reader,
			_ string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			capturedArtefactID = artefactID
			return &registry_dto.ArtefactMeta{}, nil
		},
	}

	adapter := NewRegistryStorageAdapter(mock)

	err := adapter.StoreSitemap(context.Background(), "sitemap-part-2.xml", []byte("<urlset/>"), nil)
	require.NoError(t, err)
	assert.Equal(t, "sitemap-part-2.xml", capturedArtefactID)
}
