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

//go:build integration && vips

package asset_pipeline_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/image/image_domain"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/registry_blob_adapter"
	"piko.sh/piko/tests/integration/testutil"
	"piko.sh/piko/wdk/media/image_provider_imaging"
	"piko.sh/piko/wdk/media/image_provider_vips"

	_ "golang.org/x/image/webp"
)

type assetPipelineHarness struct {
	*testutil.BaseHarness
	registryService   registry_domain.RegistryService
	metadataStore     registry_domain.MetadataStore
	blobStore         registry_domain.BlobStore
	blobStores        map[string]registry_domain.BlobStore
	tempDir           string
	testPath          string
	cleanup           func()
	imageService      image_domain.Service
	capabilityService capabilities_domain.CapabilityService
	storedArtefacts   []string
}

func runTestCase(t *testing.T, tc testutil.TestCase) {
	base := testutil.NewBaseHarness(t, tc)

	err := base.LoadSpec()
	require.NoError(t, err, "Failed to load testspec.json")

	harness := setupRegistryFixture(t, base, tc.Path)
	defer harness.cleanup()

	ctx := context.Background()

	if len(base.Spec.Assets) > 0 {
		for _, asset := range base.Spec.Assets {
			sourcePath := filepath.Join(tc.Path, asset.SourcePath)

			err := storeAsset(ctx, harness, asset.ExpectedArtefactID, sourcePath)
			require.NoError(t, err, "Failed to store asset: %s", asset.SourcePath)

			harness.storedArtefacts = append(harness.storedArtefacts, asset.ExpectedArtefactID)

			stored, err := harness.registryService.GetArtefact(ctx, asset.ExpectedArtefactID)
			require.NoError(t, err, "Artefact should be retrievable by expected ID: %s", asset.ExpectedArtefactID)
			assert.Equal(t, asset.ExpectedArtefactID, stored.ID, "Stored artefact ID mismatch")
		}
	}

	if len(base.Spec.RenderChecks) > 0 {
		spy := testutil.NewSpyRegistryPort()

		for _, check := range base.Spec.RenderChecks {
			spy.SetSVGData(check.ExpectLookupID, &render_domain.ParsedSvgData{
				InnerHTML: "<path d='test'></path>",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "viewBox", Value: "0 0 24 24"},
				},
			})
		}

		for _, check := range base.Spec.RenderChecks {
			_, err := spy.GetAssetRawSVG(ctx, check.TemplateSrc)
			require.NoError(t, err, "Lookup should succeed for: %s", check.TemplateSrc)

			lookups := spy.GetSVGLookupCalls()
			assert.Contains(t, lookups, check.ExpectLookupID,
				"Render should lookup artefact with ID %q", check.ExpectLookupID)
		}
	}

	if len(base.Spec.Transformations) > 0 {
		for _, transform := range base.Spec.Transformations {
			var sourceArtefactID string
			assetIndex := transform.SourceAssetIndex
			if assetIndex < len(base.Spec.Assets) {
				sourceArtefactID = base.Spec.Assets[assetIndex].ExpectedArtefactID
			}
			require.NotEmpty(t, sourceArtefactID, "No source asset found for transformation")

			outputData, mimeType, err := executeTransformation(ctx, harness, sourceArtefactID, transform)
			require.NoError(t, err, "Transformation failed for profile %s: %v", transform.ProfileName, err)

			verifyTransformationOutput(t, transform, outputData, mimeType)

			if transform.GoldenFile != "" {
				handleGoldenFile(t, tc.Path, transform.GoldenFile, outputData)
			}
		}
	}

	if len(base.Spec.ErrorChecks) > 0 {
		for _, errCheck := range base.Spec.ErrorChecks {
			description := errCheck.Description
			if description == "" {
				description = fmt.Sprintf("%s operation", errCheck.Operation)
			}

			err := executeErrorCheck(ctx, harness, tc.Path, errCheck)

			if errCheck.ExpectError {
				require.Error(t, err, "Expected error for: %s", description)
				if errCheck.ErrorContains != "" {
					assert.Contains(t, err.Error(), errCheck.ErrorContains,
						"Error should contain %q", errCheck.ErrorContains)
				}
			} else {
				require.NoError(t, err, "Expected success for: %s", description)
			}
		}
	}

	if len(base.Spec.DeletionChecks) > 0 {
		for _, delCheck := range base.Spec.DeletionChecks {
			description := delCheck.Description
			if description == "" {
				description = fmt.Sprintf("delete %s", delCheck.ArtefactID)
			}

			err := executeDeletionCheck(ctx, harness, delCheck)

			if delCheck.ExpectError {
				require.Error(t, err, "Expected error for: %s", description)
				if delCheck.ErrorContains != "" {
					assert.Contains(t, err.Error(), delCheck.ErrorContains,
						"Error should contain %q", delCheck.ErrorContains)
				}
			} else {
				require.NoError(t, err, "Expected success for: %s", description)

				if delCheck.ExpectGCHints > 0 {
					hints, hintErr := harness.registryService.PopGCHints(ctx, delCheck.ExpectGCHints+10)
					require.NoError(t, hintErr, "Failed to get GC hints")
					assert.GreaterOrEqual(t, len(hints), delCheck.ExpectGCHints,
						"Expected at least %d GC hints", delCheck.ExpectGCHints)
				}
			}
		}
	}

	if len(base.Spec.DeduplicationChecks) > 0 {
		for _, dedupCheck := range base.Spec.DeduplicationChecks {
			description := dedupCheck.Description
			if description == "" {
				description = fmt.Sprintf("dedup check for %d assets", len(dedupCheck.Assets))
			}

			err := executeDeduplicationCheck(ctx, harness, tc.Path, dedupCheck)
			require.NoError(t, err, "Deduplication check failed: %s", description)
		}
	}

	if len(base.Spec.InvalidationChecks) > 0 {
		for _, invCheck := range base.Spec.InvalidationChecks {
			description := invCheck.Description
			if description == "" {
				description = fmt.Sprintf("invalidate %s", invCheck.ArtefactID)
			}

			err := executeInvalidationCheck(ctx, harness, tc.Path, invCheck)
			require.NoError(t, err, "Invalidation check failed: %s", description)
		}
	}

	if len(base.Spec.BatchChecks) > 0 {
		for _, batchCheck := range base.Spec.BatchChecks {
			description := batchCheck.Description
			if description == "" {
				description = fmt.Sprintf("%s operation", batchCheck.Operation)
			}

			err := executeBatchCheck(ctx, harness, batchCheck)
			require.NoError(t, err, "Batch check failed: %s", description)
		}
	}

	if len(base.Spec.ResponsiveChecks) > 0 {
		for _, respCheck := range base.Spec.ResponsiveChecks {
			err := executeResponsiveCheck(ctx, t, harness, tc.Path, respCheck)
			require.NoError(t, err, "Responsive check failed")
		}
	}

	if len(base.Spec.VariantURLChecks) > 0 {
		for _, urlCheck := range base.Spec.VariantURLChecks {
			err := executeVariantURLCheck(ctx, t, harness, tc.Path, urlCheck)
			require.NoError(t, err, "Variant URL check failed: %s", urlCheck.ProfileName)
		}
	}
}

func setupRegistryFixture(t *testing.T, base *testutil.BaseHarness, testPath string) *assetPipelineHarness {
	t.Helper()

	transformerType := base.Spec.Transformer
	if transformerType == "" {
		transformerType = "imaging"
	}

	tempDir, err := os.MkdirTemp("", "asset-pipeline-test-*")
	require.NoError(t, err)

	metaStore, err := registry_otter.NewOtterDAL(registry_otter.Config{Capacity: 100_000})
	require.NoError(t, err)

	blobDir := filepath.Join(tempDir, "blobs")
	diskProvider, err := provider_disk.NewDiskProvider(provider_disk.Config{
		BaseDirectory: blobDir,
	})
	require.NoError(t, err)
	blobStore, err := registry_blob_adapter.NewBlobStoreAdapter(registry_blob_adapter.Config{
		Provider:   diskProvider,
		Repository: "",
	})
	require.NoError(t, err)
	blobStores := map[string]registry_domain.BlobStore{"disk": blobStore}

	service := registry_domain.NewRegistryService(metaStore, blobStores, nil, nil)

	imageConfig := image_domain.DefaultServiceConfig()
	transformers := make(map[string]image_domain.TransformerPort)

	var vipsTransformer *image_provider_vips.Provider

	imagingTransformer := image_provider_imaging.NewProvider(image_provider_imaging.Config{
		ImageServiceConfig: imageConfig,
	})
	transformers["imaging"] = imagingTransformer

	defaultTransformer := "imaging"
	if transformerType == "vips" {

		vipsTransformer, err = image_provider_vips.NewProvider(image_provider_vips.Config{
			ImageServiceConfig: imageConfig,
		})
		require.NoError(t, err, "Failed to create vips transformer - ensure libvips is installed")
		transformers["vips"] = vipsTransformer
		defaultTransformer = "vips"
	}
	transformers["default"] = transformers[defaultTransformer]

	imageService, err := image_domain.NewService(transformers, defaultTransformer, imageConfig)
	require.NoError(t, err)

	capService, err := capabilities.NewServiceWithBuiltins(
		capabilities.WithImageProvider(imageService),
	)
	require.NoError(t, err)

	cleanup := func() {
		_ = metaStore.Close()
		_ = os.RemoveAll(tempDir)
	}

	return &assetPipelineHarness{
		BaseHarness:       base,
		registryService:   service,
		metadataStore:     metaStore,
		blobStore:         blobStore,
		blobStores:        blobStores,
		tempDir:           tempDir,
		testPath:          testPath,
		cleanup:           cleanup,
		imageService:      imageService,
		capabilityService: capService,
	}
}

func storeAsset(ctx context.Context, h *assetPipelineHarness, artefactID, sourcePath string) error {

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading source file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(sourcePath))
	mimeType := "application/octet-stream"
	switch ext {
	case ".svg":
		mimeType = "image/svg+xml"
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".bmp":
		mimeType = "image/bmp"
	case ".css":
		mimeType = "text/css"
	case ".js":
		mimeType = "application/javascript"
	}

	desiredProfiles := []registry_dto.NamedProfile{
		{
			Name: "original",
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityNeed,
				CapabilityName: "identity",
				Params:         registry_dto.ProfileParams{},
				ResultingTags: registry_dto.TagsFromMap(map[string]string{
					"mime_type": mimeType,
				}),
			},
		},
	}

	_, err = h.registryService.UpsertArtefact(
		ctx,
		artefactID,
		sourcePath,
		bytes.NewReader(data),
		"disk",
		desiredProfiles,
	)
	if err != nil {
		return fmt.Errorf("upserting artefact: %w", err)
	}

	return nil
}

func executeTransformation(
	ctx context.Context,
	h *assetPipelineHarness,
	artefactID string,
	transform testutil.TransformationCheck,
) ([]byte, string, error) {

	artefact, err := h.registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		return nil, "", fmt.Errorf("getting artefact: %w", err)
	}

	var sourceVariant *registry_dto.Variant
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == "source" {
			sourceVariant = &artefact.ActualVariants[i]
			break
		}
	}
	if sourceVariant == nil {
		return nil, "", fmt.Errorf("source variant not found for artefact %s", artefactID)
	}

	sourceData, err := h.blobStore.Get(ctx, sourceVariant.StorageKey)
	if err != nil {
		return nil, "", fmt.Errorf("getting source data: %w", err)
	}
	defer func() { _ = sourceData.Close() }()

	sourceBytes, err := io.ReadAll(sourceData)
	if err != nil {
		return nil, "", fmt.Errorf("reading source data: %w", err)
	}

	outputReader, err := h.capabilityService.Execute(
		ctx,
		transform.CapabilityName,
		bytes.NewReader(sourceBytes),
		transform.Params,
	)
	if err != nil {
		return nil, "", fmt.Errorf("executing capability: %w", err)
	}

	outputBytes, err := io.ReadAll(outputReader)
	if err != nil {
		return nil, "", fmt.Errorf("reading output: %w", err)
	}

	mimeType := transform.Expected.MimeType
	if mimeType == "" {
		if format, ok := transform.Params["format"]; ok {
			switch strings.ToLower(format) {
			case "jpeg", "jpg":
				mimeType = "image/jpeg"
			case "png":
				mimeType = "image/png"
			case "webp":
				mimeType = "image/webp"
			case "gif":
				mimeType = "image/gif"
			default:
				mimeType = "application/octet-stream"
			}
		}
	}

	return outputBytes, mimeType, nil
}

func verifyTransformationOutput(t *testing.T, transform testutil.TransformationCheck, outputData []byte, mimeType string) {
	t.Helper()

	if transform.Expected.MimeType != "" {
		assert.Equal(t, transform.Expected.MimeType, mimeType,
			"MIME type mismatch for profile %s", transform.ProfileName)
	}

	if transform.Expected.MinSizeBytes > 0 {
		assert.GreaterOrEqual(t, int64(len(outputData)), transform.Expected.MinSizeBytes,
			"Output too small for profile %s", transform.ProfileName)
	}

	if transform.Expected.MaxSizeBytes > 0 {
		assert.LessOrEqual(t, int64(len(outputData)), transform.Expected.MaxSizeBytes,
			"Output too large for profile %s", transform.ProfileName)
	}

	if transform.Expected.ExactSizeBytes > 0 {
		assert.Equal(t, transform.Expected.ExactSizeBytes, int64(len(outputData)),
			"Output size mismatch for profile %s", transform.ProfileName)
	}

	if transform.Expected.StartsWithDataURL != "" {
		outputString := string(outputData)
		assert.True(t, strings.HasPrefix(outputString, transform.Expected.StartsWithDataURL),
			"Output should start with %q for profile %s", transform.Expected.StartsWithDataURL, transform.ProfileName)
	}

	if transform.Expected.OutputNotEmpty {
		assert.NotEmpty(t, outputData, "Output should not be empty for profile %s", transform.ProfileName)
	}

	if needsDimensionCheck(transform.Expected) && !strings.HasPrefix(mimeType, "text/") {
		width, height, err := getImageDimensions(outputData)
		if err != nil {
			t.Logf("    Warning: Could not get image dimensions: %v", err)
		} else {
			t.Logf("    Dimensions: %dx%d", width, height)

			if transform.Expected.ExactWidth > 0 {
				assert.Equal(t, transform.Expected.ExactWidth, width,
					"Width mismatch for profile %s", transform.ProfileName)
			}
			if transform.Expected.ExactHeight > 0 {
				assert.Equal(t, transform.Expected.ExactHeight, height,
					"Height mismatch for profile %s", transform.ProfileName)
			}
			if transform.Expected.MinWidth > 0 {
				assert.GreaterOrEqual(t, width, transform.Expected.MinWidth,
					"Width too small for profile %s", transform.ProfileName)
			}
			if transform.Expected.MaxWidth > 0 {
				assert.LessOrEqual(t, width, transform.Expected.MaxWidth,
					"Width too large for profile %s", transform.ProfileName)
			}
			if transform.Expected.MinHeight > 0 {
				assert.GreaterOrEqual(t, height, transform.Expected.MinHeight,
					"Height too small for profile %s", transform.ProfileName)
			}
			if transform.Expected.MaxHeight > 0 {
				assert.LessOrEqual(t, height, transform.Expected.MaxHeight,
					"Height too large for profile %s", transform.ProfileName)
			}
		}
	}
}

func needsDimensionCheck(exp testutil.VariantExpectation) bool {
	return exp.ExactWidth > 0 || exp.ExactHeight > 0 ||
		exp.MinWidth > 0 || exp.MaxWidth > 0 ||
		exp.MinHeight > 0 || exp.MaxHeight > 0
}

func getImageDimensions(data []byte) (int, int, error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, fmt.Errorf("decoding image config: %w", err)
	}
	return img.Width, img.Height, nil
}

func handleGoldenFile(t *testing.T, testPath, goldenFileName string, outputData []byte) {
	t.Helper()

	goldenDir := filepath.Join(testPath, "golden")
	goldenPath := filepath.Join(goldenDir, goldenFileName)

	if *testutil.UpdateGolden {

		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatalf("Failed to create golden directory: %v", err)
		}

		if err := os.WriteFile(goldenPath, outputData, 0o644); err != nil {
			t.Fatalf("Failed to write golden file %s: %v", goldenPath, err)
		}
	} else {

		expectedData, err := os.ReadFile(goldenPath)
		if err != nil {
			if os.IsNotExist(err) {
				t.Fatalf("Golden file not found: %s (run with -update to create)", goldenPath)
			}
			t.Fatalf("Failed to read golden file: %v", err)
		}

		if !bytes.Equal(outputData, expectedData) {
			t.Errorf("Golden file mismatch for %s: got %d bytes, expected %d bytes",
				goldenFileName, len(outputData), len(expectedData))
		}
	}
}

func executeErrorCheck(ctx context.Context, h *assetPipelineHarness, testPath string, check testutil.ErrorCheck) error {
	switch check.Operation {
	case "store":

		artefactID := check.ArtefactID
		if artefactID == "" {
			artefactID = ""
		}
		sourcePath := check.SourcePath
		if sourcePath != "" {
			sourcePath = filepath.Join(testPath, sourcePath)
		}

		if sourcePath == "" {
			return storeAsset(ctx, h, artefactID, sourcePath)
		}
		return storeAsset(ctx, h, artefactID, sourcePath)

	case "get":

		artefactID := check.ArtefactID
		_, err := h.registryService.GetArtefact(ctx, artefactID)
		return err

	case "delete":

		artefactID := check.ArtefactID
		return h.registryService.DeleteArtefact(ctx, artefactID)

	case "transform":

		if len(h.storedArtefacts) == 0 {
			return errors.New("no stored artefacts to transform")
		}
		artefactID := h.storedArtefacts[0]
		transform := testutil.TransformationCheck{
			ProfileName:    "error-test",
			CapabilityName: check.Params["capability"],
			Params:         check.Params,
		}
		_, _, err := executeTransformation(ctx, h, artefactID, transform)
		return err

	default:
		return fmt.Errorf("unknown error check operation: %s", check.Operation)
	}
}

func executeDeletionCheck(ctx context.Context, h *assetPipelineHarness, check testutil.DeletionCheck) error {
	return h.registryService.DeleteArtefact(ctx, check.ArtefactID)
}

func executeDeduplicationCheck(ctx context.Context, h *assetPipelineHarness, testPath string, check testutil.DeduplicationCheck) error {

	storageKeys := make(map[string]bool)

	for _, asset := range check.Assets {
		sourcePath := filepath.Join(testPath, asset.SourcePath)
		err := storeAsset(ctx, h, asset.ExpectedArtefactID, sourcePath)
		if err != nil {
			return fmt.Errorf("storing asset %s: %w", asset.ExpectedArtefactID, err)
		}

		artefact, err := h.registryService.GetArtefact(ctx, asset.ExpectedArtefactID)
		if err != nil {
			return fmt.Errorf("getting artefact %s: %w", asset.ExpectedArtefactID, err)
		}

		for _, variant := range artefact.ActualVariants {
			if variant.VariantID == "source" {
				storageKeys[variant.StorageKey] = true
				break
			}
		}
	}

	if check.ExpectSingleBlob {
		if len(storageKeys) != 1 {
			return fmt.Errorf("expected single blob, got %d unique storage keys: %v",
				len(storageKeys), storageKeys)
		}
	}

	if check.ExpectRefCount > 0 {
		for key := range storageKeys {
			refCount, err := h.metadataStore.GetBlobRefCount(ctx, key)
			if err != nil {
				return fmt.Errorf("getting ref count for %s: %w", key, err)
			}
			if refCount != check.ExpectRefCount {
				return fmt.Errorf("expected ref count %d for %s, got %d",
					check.ExpectRefCount, key, refCount)
			}
		}
	}

	return nil
}

func executeInvalidationCheck(ctx context.Context, h *assetPipelineHarness, testPath string, check testutil.InvalidationCheck) error {

	_, err := h.registryService.GetArtefact(ctx, check.ArtefactID)
	if err != nil {
		return fmt.Errorf("artefact %s not found: %w", check.ArtefactID, err)
	}

	if check.ModifySourcePath != "" {
		sourcePath := filepath.Join(testPath, check.ModifySourcePath)
		err := storeAsset(ctx, h, check.ArtefactID, sourcePath)
		if err != nil {
			return fmt.Errorf("updating artefact %s: %w", check.ArtefactID, err)
		}
	}

	artefact, err := h.registryService.GetArtefact(ctx, check.ArtefactID)
	if err != nil {
		return fmt.Errorf("getting updated artefact %s: %w", check.ArtefactID, err)
	}

	variantStatuses := make(map[string]string)
	for _, v := range artefact.ActualVariants {
		variantStatuses[v.VariantID] = string(v.Status)
	}

	for _, variantID := range check.ExpectStaleVariants {
		status, exists := variantStatuses[variantID]
		if !exists {
			return fmt.Errorf("expected stale variant %s not found", variantID)
		}
		if status != "STALE" {
			return fmt.Errorf("expected variant %s to be STALE, got %s", variantID, status)
		}
	}

	for _, variantID := range check.ExpectReadyVariants {
		status, exists := variantStatuses[variantID]
		if !exists {
			return fmt.Errorf("expected ready variant %s not found", variantID)
		}
		if status != "READY" {
			return fmt.Errorf("expected variant %s to be READY, got %s", variantID, status)
		}
	}

	return nil
}

func executeBatchCheck(ctx context.Context, h *assetPipelineHarness, check testutil.BatchCheck) error {
	switch check.Operation {
	case "getMultiple":
		artefacts, err := h.registryService.GetMultipleArtefacts(ctx, check.ArtefactIDs)
		if err != nil {
			return err
		}

		if check.ExpectCount > 0 && len(artefacts) != check.ExpectCount {
			return fmt.Errorf("expected %d artefacts, got %d", check.ExpectCount, len(artefacts))
		}

		foundIDs := make(map[string]bool)
		for _, a := range artefacts {
			foundIDs[a.ID] = true
		}
		for _, expectedID := range check.ExpectIDs {
			if !foundIDs[expectedID] {
				return fmt.Errorf("expected artefact %s not found in results", expectedID)
			}
		}
		return nil

	case "listAll":
		ids, err := h.registryService.ListAllArtefactIDs(ctx)
		if err != nil {
			return err
		}

		if check.ExpectCount > 0 && len(ids) != check.ExpectCount {
			return fmt.Errorf("expected %d artefact IDs, got %d", check.ExpectCount, len(ids))
		}

		idSet := make(map[string]bool)
		for _, id := range ids {
			idSet[id] = true
		}
		for _, expectedID := range check.ExpectIDs {
			if !idSet[expectedID] {
				return fmt.Errorf("expected artefact %s not found in list", expectedID)
			}
		}
		return nil

	case "search":
		if len(check.SearchTags) == 0 {
			return errors.New("search operation requires searchTags")
		}

		var tagKeys []string
		var tagValues []string
		for k, v := range check.SearchTags {
			tagKeys = append(tagKeys, k)
			tagValues = append(tagValues, v)
		}

		results, err := h.registryService.SearchArtefactsByTagValues(ctx, tagKeys[0], tagValues)
		if err != nil {
			return err
		}

		if check.ExpectCount > 0 && len(results) != check.ExpectCount {
			return fmt.Errorf("expected %d search results, got %d", check.ExpectCount, len(results))
		}
		return nil

	default:
		return fmt.Errorf("unknown batch operation: %s", check.Operation)
	}
}

func executeResponsiveCheck(ctx context.Context, t *testing.T, h *assetPipelineHarness, testPath string, check testutil.ResponsiveCheck) error {
	t.Helper()

	if check.SourceAssetIndex >= len(h.storedArtefacts) {
		return fmt.Errorf("source asset index %d out of range (have %d)", check.SourceAssetIndex, len(h.storedArtefacts))
	}
	artefactID := h.storedArtefacts[check.SourceAssetIndex]

	artefact, err := h.registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		return fmt.Errorf("getting source artefact: %w", err)
	}

	var sourceVariant *registry_dto.Variant
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == "source" {
			sourceVariant = &artefact.ActualVariants[i]
			break
		}
	}
	if sourceVariant == nil {
		return fmt.Errorf("source variant not found for artefact %s", artefactID)
	}

	sourceReader, err := h.registryService.GetVariantData(ctx, sourceVariant)
	if err != nil {
		return fmt.Errorf("getting source data: %w", err)
	}
	sourceData, err := io.ReadAll(sourceReader)
	_ = sourceReader.Close()
	if err != nil {
		return fmt.Errorf("reading source data: %w", err)
	}

	format := check.Format
	if format == "" {
		format = "webp"
	}
	quality := check.Quality
	if quality == 0 {
		quality = 85
	}

	for _, expectedVariant := range check.ExpectedVariants {
		density := expectedVariant.Density
		targetWidth := expectedVariant.ExpectedWidth

		transform := testutil.TransformationCheck{
			ProfileName:    fmt.Sprintf("responsive-%s", density),
			CapabilityName: "image-transform",
			Params: map[string]string{
				"width":   fmt.Sprintf("%d", targetWidth),
				"format":  format,
				"quality": fmt.Sprintf("%d", quality),
				"fit":     "cover",
			},
		}

		if expectedVariant.ExpectedHeight > 0 {
			transform.Params["height"] = fmt.Sprintf("%d", expectedVariant.ExpectedHeight)
			transform.Params["fit"] = "fill"
		}

		outputReader, err := h.capabilityService.Execute(
			ctx,
			"image-transform",
			bytes.NewReader(sourceData),
			transform.Params,
		)
		if err != nil {
			return fmt.Errorf("transforming for density %s: %w", density, err)
		}

		outputData, err := io.ReadAll(outputReader)
		if err != nil {
			return fmt.Errorf("reading output for density %s: %w", density, err)
		}

		if len(outputData) == 0 {
			return fmt.Errorf("empty output for density %s", density)
		}

		width, height, err := h.imageService.GetDimensions(ctx, bytes.NewReader(outputData))
		if err != nil {
			return fmt.Errorf("getting dimensions for density %s: %w", density, err)
		}

		if width != targetWidth {
			return fmt.Errorf("density %s: expected width %d, got %d", density, targetWidth, width)
		}

		if expectedVariant.ExpectedHeight > 0 && height != expectedVariant.ExpectedHeight {
			return fmt.Errorf("density %s: expected height %d, got %d", density, expectedVariant.ExpectedHeight, height)
		}

		outputSize := int64(len(outputData))
		if expectedVariant.MinSizeBytes > 0 && outputSize < expectedVariant.MinSizeBytes {
			return fmt.Errorf("density %s: output size %d < min %d", density, outputSize, expectedVariant.MinSizeBytes)
		}
		if expectedVariant.MaxSizeBytes > 0 && outputSize > expectedVariant.MaxSizeBytes {
			return fmt.Errorf("density %s: output size %d > max %d", density, outputSize, expectedVariant.MaxSizeBytes)
		}

		if check.GoldenPrefix != "" {
			goldenName := fmt.Sprintf("%s-%s.%s", check.GoldenPrefix, density, format)
			handleGoldenFile(t, testPath, goldenName, outputData)
		}

	}

	return nil
}

func executeVariantURLCheck(ctx context.Context, t *testing.T, h *assetPipelineHarness, testPath string, check testutil.VariantURLCheck) error {
	t.Helper()

	if check.SourceAssetIndex >= len(h.storedArtefacts) {
		return fmt.Errorf("source asset index %d out of range (have %d)", check.SourceAssetIndex, len(h.storedArtefacts))
	}
	artefactID := h.storedArtefacts[check.SourceAssetIndex]

	artefact, err := h.registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		return fmt.Errorf("getting source artefact: %w", err)
	}

	var sourceVariant *registry_dto.Variant
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == "source" {
			sourceVariant = &artefact.ActualVariants[i]
			break
		}
	}
	if sourceVariant == nil {
		return fmt.Errorf("source variant not found for artefact %s", artefactID)
	}

	sourceReader, err := h.registryService.GetVariantData(ctx, sourceVariant)
	if err != nil {
		return fmt.Errorf("getting source data: %w", err)
	}
	sourceData, err := io.ReadAll(sourceReader)
	_ = sourceReader.Close()
	if err != nil {
		return fmt.Errorf("reading source data: %w", err)
	}

	outputReader, err := h.capabilityService.Execute(
		ctx,
		"image-transform",
		bytes.NewReader(sourceData),
		check.TransformParams,
	)
	if err != nil {
		return fmt.Errorf("transforming image: %w", err)
	}

	outputData, err := io.ReadAll(outputReader)
	if err != nil {
		return fmt.Errorf("reading transformed output: %w", err)
	}

	if check.Expected.MinSizeBytes > 0 && int64(len(outputData)) < check.Expected.MinSizeBytes {
		return fmt.Errorf("output size %d < min %d", len(outputData), check.Expected.MinSizeBytes)
	}

	if check.Expected.ExactWidth > 0 || check.Expected.ExactHeight > 0 {
		width, height, err := h.imageService.GetDimensions(ctx, bytes.NewReader(outputData))
		if err != nil {
			return fmt.Errorf("getting dimensions: %w", err)
		}
		if check.Expected.ExactWidth > 0 && width != check.Expected.ExactWidth {
			return fmt.Errorf("width %d != expected %d", width, check.Expected.ExactWidth)
		}
		if check.Expected.ExactHeight > 0 && height != check.Expected.ExactHeight {
			return fmt.Errorf("height %d != expected %d", height, check.Expected.ExactHeight)
		}
	}

	expectedURL := fmt.Sprintf("/_piko/assets/%s?v=%s", artefactID, check.ProfileName)

	if check.ExpectedURLPattern != "" {
		expectedPattern := strings.ReplaceAll(check.ExpectedURLPattern, "{artefactID}", artefactID)
		expectedPattern = strings.ReplaceAll(expectedPattern, "{profile}", check.ProfileName)
		if expectedURL != expectedPattern {
			return fmt.Errorf("URL mismatch: got %s, expected pattern %s", expectedURL, expectedPattern)
		}
	}

	if check.GoldenFile != "" {
		handleGoldenFile(t, testPath, check.GoldenFile, outputData)
	}

	return nil
}

func TestDynamicAssetRegistration_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	tc := testutil.TestCase{
		Name: "dynamic_asset_registration",
		Path: "./testdata/12_artefact_deletion",
	}

	base := testutil.NewBaseHarness(t, tc)

	_ = base.LoadSpec()
	if base.Spec == nil {

		base.Spec = &testutil.TestSpec{
			Description: "Dynamic asset registration test",
			Transformer: "imaging",
		}
	}

	harness := setupRegistryFixture(t, base, tc.Path)
	defer harness.cleanup()

	ctx := context.Background()

	t.Run("Phase1_MetadataOnlyRegistration", func(t *testing.T) {

		artefactID := "test/dynamic/product-123.jpg"
		desiredProfiles := []registry_dto.NamedProfile{
			{
				Name: "image_w640_webp",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image.transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"format": "webp",
						"width":  "640",
					}),
					ResultingTags: registry_dto.TagsFromMap(map[string]string{
						"format": "webp",
						"width":  "640",
					}),
				},
			},
			{
				Name: "image_w1280_webp",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image-transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"format": "webp",
						"width":  "1280",
					}),
					ResultingTags: registry_dto.TagsFromMap(map[string]string{
						"format": "webp",
						"width":  "1280",
					}),
				},
			},
		}

		artefact, err := harness.registryService.UpsertArtefact(
			ctx,
			artefactID,
			artefactID,
			nil,
			"default",
			desiredProfiles,
		)
		require.NoError(t, err, "Metadata-only registration should succeed")
		require.NotNil(t, artefact, "Artefact should be returned")

		assert.Equal(t, registry_dto.VariantStatusPending, artefact.ComputeStatus(),
			"Artefact should be PENDING with no actual variants")

		retrieved, err := harness.registryService.GetArtefact(ctx, artefactID)
		require.NoError(t, err, "Should be able to retrieve PENDING artefact")
		assert.Equal(t, artefactID, retrieved.ID)
		assert.Equal(t, 2, len(retrieved.DesiredProfiles), "Should have 2 desired profiles")
		assert.Equal(t, 0, len(retrieved.ActualVariants), "Should have 0 actual variants")
	})

	t.Run("Phase2_ProfileConversion", func(t *testing.T) {

		artefactID := "test/dynamic/profile-test.jpg"
		testAssetPath := filepath.Join(tc.Path, "src/lib/images/jersey.jpg")
		err := storeAsset(ctx, harness, artefactID, testAssetPath)
		require.NoError(t, err)

		desiredProfiles := []registry_dto.NamedProfile{
			{
				Name: "image_w640_webp",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image.transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"format": "webp",
						"width":  "640",
					}),
					ResultingTags: registry_dto.TagsFromMap(map[string]string{
						"format": "webp",
						"width":  "640",
					}),
				},
			},
			{
				Name: "image_w1280_avif",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image-transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"format": "avif",
						"width":  "1280",
					}),
					ResultingTags: registry_dto.TagsFromMap(map[string]string{
						"format": "avif",
						"width":  "1280",
					}),
				},
			},
		}

		artefact, err := harness.registryService.UpsertArtefact(
			ctx,
			artefactID,
			artefactID,
			nil,
			"default",
			desiredProfiles,
		)
		require.NoError(t, err)

		assert.Equal(t, 2, len(artefact.DesiredProfiles))
		webpProfile, foundWebp := artefact.GetProfile("image_w640_webp")
		assert.True(t, foundWebp, "image_w640_webp profile should exist")
		avifProfile, foundAvif := artefact.GetProfile("image_w1280_avif")
		assert.True(t, foundAvif, "image_w1280_avif profile should exist")
		_ = avifProfile

		assert.Equal(t, "image.transform", webpProfile.CapabilityName)
		formatVal, _ := webpProfile.Params.GetByName("format")
		assert.Equal(t, "webp", formatVal)
		widthVal, _ := webpProfile.Params.GetByName("width")
		assert.Equal(t, "640", widthVal)
	})

	t.Run("Phase3_HTTPLazyVariantGeneration", func(t *testing.T) {

		artefactID := "test/dynamic/lazy-test.jpg"
		testAssetPath := filepath.Join(tc.Path, "src/lib/images/jersey.jpg")

		err := storeAsset(ctx, harness, artefactID, testAssetPath)
		require.NoError(t, err)

		desiredProfiles := []registry_dto.NamedProfile{
			{
				Name: "image_w400_webp",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "image-transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"format": "webp",
						"width":  "400",
					}),
					ResultingTags: registry_dto.TagsFromMap(map[string]string{
						"format": "webp",
						"width":  "400",
					}),
				},
			},
		}

		artefact, err := harness.registryService.UpsertArtefact(
			ctx,
			artefactID,
			artefactID,
			nil,
			"default",
			desiredProfiles,
		)
		require.NoError(t, err)

		onDemandGen := harness.createOnDemandGenerator()

		variant, err := onDemandGen.GenerateVariant(ctx, artefact, "image_w400_webp")
		require.NoError(t, err, "Lazy variant generation should succeed")
		require.NotNil(t, variant, "Variant should be created")

		refreshed, err := harness.registryService.GetArtefact(ctx, artefactID)
		require.NoError(t, err)

		found := false
		for i := range refreshed.ActualVariants {
			if refreshed.ActualVariants[i].VariantID == "image_w400_webp" {
				found = true
				break
			}
		}
		assert.True(t, found, "Generated variant should be in registry")
	})
}

func (h *assetPipelineHarness) createOnDemandGenerator() *OnDemandVariantGeneratorWrapper {
	return &OnDemandVariantGeneratorWrapper{
		registryService:   h.registryService,
		blobStore:         h.blobStore,
		capabilityService: h.capabilityService,
	}
}

type OnDemandVariantGeneratorWrapper struct {
	registryService   registry_domain.RegistryService
	blobStore         registry_domain.BlobStore
	capabilityService capabilities_domain.CapabilityService
}

func (g *OnDemandVariantGeneratorWrapper) GenerateVariant(
	ctx context.Context,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
) (*registry_dto.Variant, error) {

	var sourceVariant *registry_dto.Variant
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == "source" {
			sourceVariant = &artefact.ActualVariants[i]
			break
		}
	}
	if sourceVariant == nil {
		return nil, fmt.Errorf("source variant not found for artefact %s", artefact.ID)
	}

	profile, ok := artefact.GetProfile(profileName)
	if !ok {
		return nil, fmt.Errorf("profile %q not found in desired profiles", profileName)
	}

	sourceStream, err := g.blobStore.Get(ctx, sourceVariant.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("getting source blob: %w", err)
	}
	defer func() { _ = sourceStream.Close() }()

	sourceData, err := io.ReadAll(sourceStream)
	if err != nil {
		return nil, fmt.Errorf("reading source data: %w", err)
	}

	outputReader, err := g.capabilityService.Execute(
		ctx,
		profile.CapabilityName,
		bytes.NewReader(sourceData),
		profile.Params.ToMap(),
	)
	if err != nil {
		return nil, fmt.Errorf("executing capability: %w", err)
	}

	outputData, err := io.ReadAll(outputReader)
	if err != nil {
		return nil, fmt.Errorf("reading output: %w", err)
	}

	mimeType := "application/octet-stream"
	if format, ok := profile.Params.GetByName("format"); ok {
		switch strings.ToLower(format) {
		case "jpeg", "jpg":
			mimeType = "image/jpeg"
		case "png":
			mimeType = "image/png"
		case "webp":
			mimeType = "image/webp"
		case "avif":
			mimeType = "image/avif"
		case "gif":
			mimeType = "image/gif"
		}
	}

	variantKey := fmt.Sprintf("generated/%s_%s", artefact.ID, profileName)
	err = g.blobStore.Put(ctx, variantKey, bytes.NewReader(outputData))
	if err != nil {
		return nil, fmt.Errorf("storing variant blob: %w", err)
	}

	newVariant := registry_dto.Variant{
		VariantID:        profileName,
		StorageKey:       variantKey,
		StorageBackendID: "default",
		MimeType:         mimeType,
		Status:           registry_dto.VariantStatusReady,
		SizeBytes:        int64(len(outputData)),
		MetadataTags:     registry_dto.Tags{},
		CreatedAt:        time.Now(),
	}

	_, err = g.registryService.AddVariant(ctx, artefact.ID, &newVariant)
	if err != nil {
		return nil, fmt.Errorf("adding variant to registry: %w", err)
	}

	return &newVariant, nil
}
