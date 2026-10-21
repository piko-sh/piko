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

package lifecycle_adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func desiredProfileNames(artefact *registry_dto.ArtefactMeta) []string {
	names := make([]string, len(artefact.DesiredProfiles))
	for i := range artefact.DesiredProfiles {
		names[i] = artefact.DesiredProfiles[i].Name
	}
	return names
}

func TestBuildWalk_KeepsAnnotationProfiles(t *testing.T) {
	t.Parallel()

	const moduleName = "example.com/site"
	artefactID := moduleName + "/lib/photo.png"

	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(libDir, 0o755))
	photoPath := filepath.Join(libDir, "photo.png")
	require.NoError(t, os.WriteFile(photoPath, []byte("png bytes"), 0o600))

	ctx := context.Background()
	registry := registry_domain.NewRegistryService(
		registry_adapters.NewMockMetadataStore(),
		map[string]registry_domain.BlobStore{localDiskCacheSource: registry_adapters.NewMockBlobStore()},
		nil,
		nil,
	)
	resolver := &resolver_domain.MockResolver{GetModuleNameFunc: func() string { return moduleName }}

	annotationPipeline := lifecycle_domain.NewAssetPipelineOrchestrator(registry, &config.AssetsConfig{}, resolver, nil)
	require.NoError(t, annotationPipeline.ProcessBuildResult(ctx, &annotator_dto.ProjectAnnotationResult{
		FinalAssetManifest: []*annotator_dto.FinalAssetDependency{{
			SourcePath:           artefactID,
			AssetType:            "img",
			TransformationParams: map[string][]string{"placeholder": {"true"}},
		}},
	}))

	annotated, err := registry.GetArtefact(ctx, artefactID)
	require.NoError(t, err)
	require.Contains(t, desiredProfileNames(annotated), "placeholder",
		"the annotation pipeline registers the placeholder profile before the walk")
	require.Empty(t, annotated.ActualVariants, "the annotation pipeline does not seed local source bytes")

	bs := &buildService{
		registryService: registry,
		resolver:        resolver,
		pathsConfig:     lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir},
	}
	bs.handleFileEvent(ctx, lifecycle_dto.FileEvent{Path: photoPath, Type: lifecycle_dto.FileEventTypeCreate})

	walked, err := registry.GetArtefact(ctx, artefactID)
	require.NoError(t, err)

	walkedNames := desiredProfileNames(walked)
	for _, name := range desiredProfileNames(annotated) {
		assert.Contains(t, walkedNames, name, "the walk must keep the annotation pipeline's %q profile", name)
	}
	for _, profile := range lifecycle_domain.GetProfilesForFile(artefactID, moduleName, nil) {
		assert.Contains(t, walkedNames, profile.Name, "the walk adds its file-type %q profile", profile.Name)
	}
	require.Len(t, walked.ActualVariants, 1)
	assert.Equal(t, "source", walked.ActualVariants[0].VariantID, "the walk seeds the source bytes")
	assert.Empty(t, bs.seedFailures.take())
}

func TestBuildWalk_UpsertsWithProfileMerge(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(libDir, 0o755))
	stylePath := filepath.Join(libDir, "style.css")
	require.NoError(t, os.WriteFile(stylePath, []byte("body{}"), 0o600))

	registry := &registry_domain.MockRegistryService{}
	bs := &buildService{
		registryService: registry,
		resolver:        &resolver_domain.MockResolver{GetModuleNameFunc: func() string { return "example.com/site" }},
		pathsConfig:     lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir},
	}

	bs.handleFileEvent(context.Background(), lifecycle_dto.FileEvent{Path: stylePath, Type: lifecycle_dto.FileEventTypeCreate})

	assert.Equal(t, int64(1), registry.UpsertArtefactCallCount.Load())
	assert.Equal(t, int64(1), registry.UpsertArtefactProfileMergeCallCount.Load())
}

func TestSeedExternalAssetEntry_UpsertsWithProfileMerge(t *testing.T) {
	t.Parallel()

	assetDir := t.TempDir()
	iconPath := filepath.Join(assetDir, "arrow.svg")
	require.NoError(t, os.WriteFile(iconPath, []byte("<svg/>"), 0o600))
	entries, err := os.ReadDir(assetDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	registry := &registry_domain.MockRegistryService{}
	bs := &buildService{registryService: registry}

	require.NoError(t, bs.seedExternalAssetEntry(context.Background(), assetDir, "piko.sh/piko/lib/icons", iconPath, entries[0], nil))

	assert.Equal(t, int64(1), registry.UpsertArtefactCallCount.Load())
	assert.Equal(t, int64(1), registry.UpsertArtefactProfileMergeCallCount.Load(),
		"the external asset walk must not drop the annotation pipeline's profiles either")
	assert.Empty(t, bs.seedFailures.take())
}
