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

package daemon_domain

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/wdk/clock"
)

func TestNewOnDemandVariantGenerator_UsesRealClock_WhenNil(t *testing.T) {
	t.Parallel()

	mockRegistry := &registry_domain.MockRegistryService{}
	mockCapability := &capabilities_domain.MockCapabilityService{}
	config := DefaultOnDemandGeneratorConfig()
	config.Clock = nil

	generator := NewOnDemandVariantGenerator(mockRegistry, mockCapability, config)

	if generator == nil {
		t.Fatal("NewOnDemandVariantGenerator returned nil")
	}

	impl := mustAsGeneratorImpl(t, generator)
	if impl.clock == nil {
		t.Error("Expected clock to be set to RealClock when nil provided")
	}
}

func TestNewOnDemandVariantGenerator_UsesProvidedClock(t *testing.T) {
	t.Parallel()

	mockRegistry := &registry_domain.MockRegistryService{}
	mockCapability := &capabilities_domain.MockCapabilityService{}
	mockClock := clock.NewMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	config := DefaultOnDemandGeneratorConfig()
	config.Clock = mockClock

	generator := NewOnDemandVariantGenerator(mockRegistry, mockCapability, config)
	impl := mustAsGeneratorImpl(t, generator)

	if impl.clock != mockClock {
		t.Error("Expected generator to use provided mock clock")
	}
}

func TestDefaultOnDemandGeneratorConfig_ReturnsValidDefaults(t *testing.T) {
	t.Parallel()

	config := DefaultOnDemandGeneratorConfig()

	if config.Clock != nil {
		t.Error("Expected Clock to be nil by default")
	}
	if config.StorageBackendID != "local_disk_cache" {
		t.Errorf("Expected StorageBackendID = 'local_disk_cache', got %q", config.StorageBackendID)
	}
	if config.MaxWidth != 4096 {
		t.Errorf("Expected MaxWidth = 4096, got %d", config.MaxWidth)
	}
	if config.MinWidth != 1 {
		t.Errorf("Expected MinWidth = 1, got %d", config.MinWidth)
	}
	if config.DefaultQuality != 80 {
		t.Errorf("Expected DefaultQuality = 80, got %d", config.DefaultQuality)
	}
	if len(config.AllowedFormats) == 0 {
		t.Error("Expected AllowedFormats to contain at least one format")
	}
}

func TestParseProfileName(t *testing.T) {
	t.Parallel()

	generator := createTestGenerator(t)

	testCases := []struct {
		name          string
		profileName   string
		expectedFmt   string
		expectedWidth int
		expectedNil   bool
	}{
		{
			name:          "valid webp profile",
			profileName:   "image_w240_webp",
			expectedNil:   false,
			expectedWidth: 240,
			expectedFmt:   "webp",
		},
		{
			name:          "valid jpeg profile",
			profileName:   "image_w1024_jpeg",
			expectedNil:   false,
			expectedWidth: 1024,
			expectedFmt:   "jpeg",
		},
		{
			name:          "valid jpg profile",
			profileName:   "image_w800_jpg",
			expectedNil:   false,
			expectedWidth: 800,
			expectedFmt:   "jpg",
		},
		{
			name:          "valid avif profile",
			profileName:   "image_w512_avif",
			expectedNil:   false,
			expectedWidth: 512,
			expectedFmt:   "avif",
		},
		{
			name:          "valid png profile",
			profileName:   "image_w100_png",
			expectedNil:   false,
			expectedWidth: 100,
			expectedFmt:   "png",
		},
		{
			name:        "invalid format gif",
			profileName: "image_w240_gif",
			expectedNil: true,
		},
		{
			name:        "width too small (0)",
			profileName: "image_w0_webp",
			expectedNil: true,
		},
		{
			name:        "width too large (exceeds max)",
			profileName: "image_w9999_webp",
			expectedNil: true,
		},
		{
			name:        "malformed pattern - missing w prefix",
			profileName: "image_240_webp",
			expectedNil: true,
		},
		{
			name:        "missing image prefix",
			profileName: "w240_webp",
			expectedNil: true,
		},
		{
			name:        "empty string",
			profileName: "",
			expectedNil: true,
		},
		{
			name:        "uppercase format not allowed",
			profileName: "image_w240_WEBP",
			expectedNil: true,
		},
		{
			name:        "negative width",
			profileName: "image_w-100_webp",
			expectedNil: true,
		},
		{
			name:        "non-numeric width",
			profileName: "image_wabc_webp",
			expectedNil: true,
		},
		{
			name:        "extra underscore",
			profileName: "image_w240_webp_extra",
			expectedNil: true,
		},
		{
			name:        "missing format",
			profileName: "image_w240_",
			expectedNil: true,
		},
		{
			name:        "missing width",
			profileName: "image_w_webp",
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := generator.ParseProfileName(tc.profileName)

			if tc.expectedNil {
				if result != nil {
					t.Errorf("Expected nil for profileName %q, got %+v", tc.profileName, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected non-nil result for profileName %q", tc.profileName)
			}

			if result.Width != tc.expectedWidth {
				t.Errorf("Width = %d, want %d", result.Width, tc.expectedWidth)
			}
			if result.Format != tc.expectedFmt {
				t.Errorf("Format = %q, want %q", result.Format, tc.expectedFmt)
			}
			if result.Quality != 80 {
				t.Errorf("Quality = %d, want 80 (default)", result.Quality)
			}
		})
	}
}

func TestGenerateVariant_ReturnsError_ForInvalidProfile(t *testing.T) {
	t.Parallel()

	generator := createTestGenerator(t)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "invalid_profile_name")

	if err == nil {
		t.Fatal("Expected error for invalid profile name")
	}
	if !strings.Contains(err.Error(), "invalid or disallowed profile name") {
		t.Errorf("Expected error message about invalid profile, got: %v", err)
	}
}

func TestGenerateVariant_ReturnsExisting_WhenAlreadyGenerated(t *testing.T) {
	t.Parallel()

	existingVariant := registry_dto.Variant{
		VariantID:  "image_w240_webp",
		StorageKey: "existing/key.webp",
		Status:     registry_dto.VariantStatusReady,
	}

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
					existingVariant,
				},
			}, nil
		},
	}

	generator := createTestGeneratorWithRegistry(t, mockRegistry)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	result, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.VariantID != existingVariant.VariantID {
		t.Errorf("Expected to return existing variant, got %+v", result)
	}
}

func TestGenerateVariant_ReturnsError_WhenSourceVariantMissing(t *testing.T) {
	t.Parallel()

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:             "test-artefact",
				ActualVariants: []registry_dto.Variant{},
			}, nil
		},
	}

	generator := createTestGeneratorWithRegistry(t, mockRegistry)
	artefact := &registry_dto.ArtefactMeta{
		ID:             "test-artefact",
		SourcePath:     "images/test.png",
		ActualVariants: []registry_dto.Variant{},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when source variant is missing")
	}
	if !strings.Contains(err.Error(), "source variant not found") {
		t.Errorf("Expected error about missing source variant, got: %v", err)
	}
}

func TestGenerateVariant_ReturnsError_WhenTransformFails(t *testing.T) {
	t.Parallel()

	transformErr := errors.New("transform failed")

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return nil, transformErr
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when transform fails")
	}
	if !strings.Contains(err.Error(), "image transform failed") {
		t.Errorf("Expected error about transform failure, got: %v", err)
	}
}

func TestGenerateVariant_GeneratesNewVariant_Successfully(t *testing.T) {
	t.Parallel()

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return &registry_domain.MockBlobStore{
				PutFunc: func(_ context.Context, _ string, data io.Reader) error {
					_, _ = io.Copy(io.Discard, data)
					return nil
				},
			}, nil
		},
		AddVariantFunc: func(_ context.Context, _ string, variant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
					*variant,
				},
			}, nil
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("transformed image data")), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	result, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.VariantID != "image_w240_webp" {
		t.Errorf("VariantID = %q, want %q", result.VariantID, "image_w240_webp")
	}
	if result.MimeType != "image/webp" {
		t.Errorf("MimeType = %q, want %q", result.MimeType, "image/webp")
	}
	if result.SizeBytes == 0 {
		t.Error("Expected SizeBytes > 0")
	}
}

func TestGenerateVariant_ReturnsError_WhenBlobStoreFails(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("blob store failed")

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return nil, storeErr
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("transformed")), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when blob store fails")
	}
	if !strings.Contains(err.Error(), "failed to get blob store") {
		t.Errorf("Expected error about blob store failure, got: %v", err)
	}
}

func TestIsAllowedFormat(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	testCases := []struct {
		format   string
		expected bool
	}{
		{format: "webp", expected: true},
		{format: "jpeg", expected: true},
		{format: "jpg", expected: true},
		{format: "png", expected: true},
		{format: "avif", expected: true},
		{format: "WEBP", expected: true},
		{format: "Jpeg", expected: true},
		{format: "gif", expected: false},
		{format: "bmp", expected: false},
		{format: "tiff", expected: false},
		{format: "", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()
			result := generator.isAllowedFormat(tc.format)
			if result != tc.expected {
				t.Errorf("isAllowedFormat(%q) = %v, want %v", tc.format, result, tc.expected)
			}
		})
	}
}

func TestFindSourceVariant_ReturnsSourceVariant(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	artefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
			{VariantID: "image_w240_webp", StorageKey: "w240.webp"},
		},
	}

	result := generator.findSourceVariant(artefact)

	if result == nil {
		t.Fatal("Expected to find source variant")
	}
	if result.VariantID != "source" {
		t.Errorf("VariantID = %q, want %q", result.VariantID, "source")
	}
}

func TestFindSourceVariant_ReturnsNil_WhenNoSource(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	artefact := &registry_dto.ArtefactMeta{
		ID: "test-artefact",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "image_w240_webp", StorageKey: "w240.webp"},
		},
	}

	result := generator.findSourceVariant(artefact)

	if result != nil {
		t.Errorf("Expected nil when no source variant, got %+v", result)
	}
}

func TestGetExtensionForFormat(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	testCases := []struct {
		format   string
		expected string
	}{
		{format: "webp", expected: ".webp"},
		{format: "jpeg", expected: ".jpeg"},
		{format: "jpg", expected: ".jpeg"},
		{format: "png", expected: ".png"},
		{format: "avif", expected: ".avif"},
		{format: "unknown", expected: ".img"},
		{format: "WEBP", expected: ".webp"},
		{format: "JPEG", expected: ".jpeg"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()
			result := generator.getExtensionForFormat(tc.format)
			if result != tc.expected {
				t.Errorf("getExtensionForFormat(%q) = %q, want %q", tc.format, result, tc.expected)
			}
		})
	}
}

func TestGetMimeTypeForFormat(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	testCases := []struct {
		format   string
		expected string
	}{
		{format: "webp", expected: "image/webp"},
		{format: "jpeg", expected: "image/jpeg"},
		{format: "jpg", expected: "image/jpeg"},
		{format: "png", expected: "image/png"},
		{format: "avif", expected: "image/avif"},
		{format: "unknown", expected: "application/octet-stream"},
		{format: "WEBP", expected: "image/webp"},
		{format: "PNG", expected: "image/png"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()
			result := generator.getMimeTypeForFormat(tc.format)
			if result != tc.expected {
				t.Errorf("getMimeTypeForFormat(%q) = %q, want %q", tc.format, result, tc.expected)
			}
		})
	}
}

func TestGenerateTempKey_IncludesTimestamp(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2024, 6, 15, 12, 30, 0, 123456789, time.UTC))

	mockRegistry := &registry_domain.MockRegistryService{}
	mockCapability := &capabilities_domain.MockCapabilityService{}
	config := DefaultOnDemandGeneratorConfig()
	config.Clock = mockClock

	generator := mustAsGeneratorImpl(t, NewOnDemandVariantGenerator(mockRegistry, mockCapability, config))

	key := generator.generateTempKey("artefact-id", "image_w240_webp")

	if !strings.HasPrefix(key, "tmp/") {
		t.Errorf("Expected temp key to start with 'tmp/', got %q", key)
	}
	if !strings.Contains(key, "artefact-id") {
		t.Errorf("Expected temp key to contain artefact ID, got %q", key)
	}
	if !strings.Contains(key, "image_w240_webp") {
		t.Errorf("Expected temp key to contain profile name, got %q", key)
	}
}

func TestGenerateFinalStorageKey_IncludesHash(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))
	hash := []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x11, 0x22}

	key := generator.generateFinalStorageKey("images/photo.png", hash, "webp")

	if !strings.HasPrefix(key, "generated/") {
		t.Errorf("Expected final key to start with 'generated/', got %q", key)
	}
	if !strings.Contains(key, "123456789abcdef0") {
		t.Errorf("Expected final key to contain hash, got %q", key)
	}
	if !strings.HasSuffix(key, ".webp") {
		t.Errorf("Expected final key to end with .webp, got %q", key)
	}
}

func TestGetOrCreateVariantMutex_CreatesNew(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	mu := generator.getOrCreateVariantMutex("new-key")

	if mu == nil {
		t.Error("Expected non-nil mutex")
	}
}

func TestGetOrCreateVariantMutex_ReturnsExisting(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	mu1 := generator.getOrCreateVariantMutex("same-key")
	mu2 := generator.getOrCreateVariantMutex("same-key")

	if mu1 != mu2 {
		t.Error("Expected same mutex for same key")
	}
}

func TestCleanupVariantMutex_RemovesMutex(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	_ = generator.getOrCreateVariantMutex("test-key")

	generator.cleanupVariantMutex("test-key")

	generator.inProgressMutex.Lock()
	_, exists := generator.inProgress["test-key"]
	generator.inProgressMutex.Unlock()

	if exists {
		t.Error("Expected mutex to be removed after cleanup")
	}
}

func TestCountingHashReader_CountsBytes(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	reader := bytes.NewReader(data)
	var byteCount int64

	chr := &countingHashReader{
		reader:    reader,
		hasher:    io.Discard,
		byteCount: &byteCount,
	}

	buffer := make([]byte, 100)
	n, err := chr.Read(buffer)

	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("Unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Read %d bytes, want %d", n, len(data))
	}
	if byteCount != int64(len(data)) {
		t.Errorf("byteCount = %d, want %d", byteCount, len(data))
	}
}

func TestCountingHashReader_PropagatesEOF(t *testing.T) {
	t.Parallel()

	reader := bytes.NewReader([]byte{})
	var byteCount int64

	chr := &countingHashReader{
		reader:    reader,
		hasher:    io.Discard,
		byteCount: &byteCount,
	}

	buffer := make([]byte, 100)
	_, err := chr.Read(buffer)

	if !errors.Is(err, io.EOF) {
		t.Errorf("Expected io.EOF, got %v", err)
	}
}

func createTestGenerator(t *testing.T) OnDemandVariantGenerator {
	t.Helper()

	mockRegistry := &registry_domain.MockRegistryService{}
	mockCapability := &capabilities_domain.MockCapabilityService{}
	config := DefaultOnDemandGeneratorConfig()
	config.Clock = clock.NewMockClock(time.Now())

	return NewOnDemandVariantGenerator(mockRegistry, mockCapability, config)
}

func createTestGeneratorWithRegistry(t *testing.T, registry *registry_domain.MockRegistryService) OnDemandVariantGenerator {
	t.Helper()

	mockCapability := &capabilities_domain.MockCapabilityService{}
	config := DefaultOnDemandGeneratorConfig()
	config.Clock = clock.NewMockClock(time.Now())

	return NewOnDemandVariantGenerator(registry, mockCapability, config)
}

func createTestGeneratorWithDeps(t *testing.T, registry *registry_domain.MockRegistryService, capability *capabilities_domain.MockCapabilityService) OnDemandVariantGenerator {
	t.Helper()

	config := DefaultOnDemandGeneratorConfig()
	config.Clock = clock.NewMockClock(time.Now())

	return NewOnDemandVariantGenerator(registry, capability, config)
}

func mustAsGeneratorImpl(t *testing.T, generator OnDemandVariantGenerator) *onDemandVariantGeneratorImpl {
	t.Helper()

	impl, ok := generator.(*onDemandVariantGeneratorImpl)
	if !ok {
		t.Fatalf("expected *onDemandVariantGeneratorImpl, got %T", generator)
	}
	return impl
}

func TestGenerateVariant_ReturnsError_WhenWriteToBlobStoreFails(t *testing.T) {
	t.Parallel()

	putErr := errors.New("write failed")
	mockBlobStore := &registry_domain.MockBlobStore{
		PutFunc: func(_ context.Context, _ string, _ io.Reader) error {
			return putErr
		},
	}

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return mockBlobStore, nil
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("transformed")), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when write to blob store fails")
	}
	if !strings.Contains(err.Error(), "failed to write blob") {
		t.Errorf("Expected error about write failure, got: %v", err)
	}
}

func TestGenerateVariant_ReturnsError_WhenOutputIsZeroBytes(t *testing.T) {
	t.Parallel()

	mockBlobStore := &registry_domain.MockBlobStore{}

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return mockBlobStore, nil
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {

			return bytes.NewReader([]byte{}), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when output is zero bytes")
	}
	if !strings.Contains(err.Error(), "zero bytes") {
		t.Errorf("Expected error about zero bytes, got: %v", err)
	}
}

func TestGenerateVariant_ReturnsError_WhenRenameBlobFails(t *testing.T) {
	t.Parallel()

	renameErr := errors.New("rename failed")
	mockBlobStore := &registry_domain.MockBlobStore{
		PutFunc: func(_ context.Context, _ string, data io.Reader) error {
			_, _ = io.Copy(io.Discard, data)
			return nil
		},
		RenameFunc: func(_ context.Context, _, _ string) error {
			return renameErr
		},
	}

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return mockBlobStore, nil
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("transformed image data")), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when rename blob fails")
	}
	if !strings.Contains(err.Error(), "failed to rename blob") {
		t.Errorf("Expected error about rename failure, got: %v", err)
	}
}

func TestGenerateVariant_ReturnsError_WhenAddVariantFails(t *testing.T) {
	t.Parallel()

	addVariantErr := errors.New("add variant failed")
	mockBlobStore := &registry_domain.MockBlobStore{
		PutFunc: func(_ context.Context, _ string, data io.Reader) error {
			_, _ = io.Copy(io.Discard, data)
			return nil
		},
	}

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("image data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return mockBlobStore, nil
		},
		AddVariantFunc: func(_ context.Context, _ string, _ *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
			return nil, addVariantErr
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("transformed image data")), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when add variant fails")
	}
	if !strings.Contains(err.Error(), "failed to add variant") {
		t.Errorf("Expected error about add variant failure, got: %v", err)
	}
}

func TestBuildVariantRecord_SetsCorrectFields(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	generator := &onDemandVariantGeneratorImpl{
		clock: mockClock,
		config: OnDemandGeneratorConfig{
			StorageBackendID: "test-backend",
		},
	}

	profile := &ParsedImageProfile{
		Width:   240,
		Format:  "webp",
		Quality: 80,
	}

	hash := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	variant := generator.buildVariantRecord("image_w240_webp", "generated/test_01020304.webp", profile, hash, 1024)

	if variant.VariantID != "image_w240_webp" {
		t.Errorf("VariantID = %q, want %q", variant.VariantID, "image_w240_webp")
	}
	if variant.StorageKey != "generated/test_01020304.webp" {
		t.Errorf("StorageKey = %q, want %q", variant.StorageKey, "generated/test_01020304.webp")
	}
	if variant.MimeType != "image/webp" {
		t.Errorf("MimeType = %q, want %q", variant.MimeType, "image/webp")
	}
	if variant.SizeBytes != 1024 {
		t.Errorf("SizeBytes = %d, want %d", variant.SizeBytes, 1024)
	}
	if variant.StorageBackendID != "test-backend" {
		t.Errorf("StorageBackendID = %q, want %q", variant.StorageBackendID, "test-backend")
	}
	if variant.Status != registry_dto.VariantStatusReady {
		t.Errorf("Status = %v, want %v", variant.Status, registry_dto.VariantStatusReady)
	}
}

func TestGenerateVariant_ReturnsError_WhenGetVariantDataFails(t *testing.T) {
	t.Parallel()

	sourceDataErr := errors.New("source data unavailable")

	mockRegistry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "test-artefact",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", StorageKey: "source.png"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return nil, sourceDataErr
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	if err == nil {
		t.Fatal("Expected error when GetVariantData fails")
	}
	if !strings.Contains(err.Error(), "failed to get source data") {
		t.Errorf("Expected error about source data failure, got: %v", err)
	}
}
