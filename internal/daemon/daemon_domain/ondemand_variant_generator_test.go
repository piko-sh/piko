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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.NotNil(t, generator, "NewOnDemandVariantGenerator returned nil")

	impl := mustAsGeneratorImpl(t, generator)
	assert.NotNil(t, impl.clock, "Expected clock to be set to RealClock when nil provided")
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

	assert.Same(t, mockClock, impl.clock, "Expected generator to use provided mock clock")
}

func TestDefaultOnDemandGeneratorConfig_ReturnsValidDefaults(t *testing.T) {
	t.Parallel()

	config := DefaultOnDemandGeneratorConfig()

	assert.Nil(t, config.Clock, "Expected Clock to be nil by default")
	assert.Equal(t, "local_disk_cache", config.StorageBackendID,
		"Expected StorageBackendID = 'local_disk_cache'")
	assert.Equal(t, 4096, config.MaxWidth, "Expected MaxWidth = 4096")
	assert.Equal(t, 1, config.MinWidth, "Expected MinWidth = 1")
	assert.Equal(t, 80, config.DefaultQuality, "Expected DefaultQuality = 80")
	assert.NotEmpty(t, config.AllowedFormats, "Expected AllowedFormats to contain at least one format")
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
				assert.Nilf(t, result, "Expected nil for profileName %q", tc.profileName)
				return
			}

			require.NotNilf(t, result, "Expected non-nil result for profileName %q", tc.profileName)

			assert.Equal(t, tc.expectedWidth, result.Width, "Width")
			assert.Equal(t, tc.expectedFmt, result.Format, "Format")
			assert.Equal(t, 80, result.Quality, "Quality want 80 (default)")
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

	require.Error(t, err, "Expected error for invalid profile name")
	assert.ErrorContains(t, err, "invalid or disallowed profile name",
		"Expected error message about invalid profile")
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

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil result")
	assert.Equal(t, existingVariant.VariantID, result.VariantID, "Expected to return existing variant")
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

	require.Error(t, err, "Expected error when source variant is missing")
	assert.ErrorContains(t, err, "source variant not found", "Expected error about missing source variant")
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

	require.Error(t, err, "Expected error when transform fails")
	assert.ErrorContains(t, err, "image transform failed", "Expected error about transform failure")
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
			{VariantID: "source", StorageKey: "source.png", ContentHash: "0000000000000000000000000000000000000000000000000000000000000001"},
		},
	}

	result, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	require.NoError(t, err)
	require.NotNil(t, result, "Expected non-nil result")
	assert.Equal(t, "image_w240_webp", result.VariantID, "VariantID")
	assert.Equal(t, "image/webp", result.MimeType, "MimeType")
	assert.NotZero(t, result.SizeBytes, "Expected SizeBytes > 0")
}

type closeTrackingReadCloser struct {
	reader io.Reader
	closed bool
}

func (c *closeTrackingReadCloser) Read(buffer []byte) (int, error) {
	if c.closed {
		return 0, errors.New("read after close")
	}
	return c.reader.Read(buffer)
}

func (c *closeTrackingReadCloser) Close() error {
	c.closed = true
	return nil
}

func TestGenerateVariant_KeepsSourceOpenUntilOutputDrained(t *testing.T) {
	t.Parallel()

	sourceBytes := []byte("original source image bytes")
	source := &closeTrackingReadCloser{reader: bytes.NewReader(sourceBytes), closed: false}

	var stored []byte

	mockRegistry := &registry_domain.MockRegistryService{
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return source, nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return &registry_domain.MockBlobStore{
				PutFunc: func(_ context.Context, _ string, data io.Reader) error {
					var buffer bytes.Buffer
					_, copyErr := io.Copy(&buffer, data)
					stored = buffer.Bytes()
					return copyErr
				},
			}, nil
		},
		AddVariantFunc: func(_ context.Context, _ string, variant *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:             "test-artefact",
				ActualVariants: []registry_dto.Variant{*variant},
			}, nil
		},
	}

	mockCapability := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, input io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return io.MultiReader(input), nil
		},
	}

	generator := createTestGeneratorWithDeps(t, mockRegistry, mockCapability)
	artefact := &registry_dto.ArtefactMeta{
		ID:         "test-artefact",
		SourcePath: "images/test.png",
		ActualVariants: []registry_dto.Variant{
			{VariantID: "source", StorageKey: "source.png", ContentHash: "0000000000000000000000000000000000000000000000000000000000000001"},
		},
	}

	result, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, sourceBytes, stored, "output must read the full source before the source is closed")
	require.True(t, source.closed, "source stream must be closed once the output has been drained")
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

	require.Error(t, err, "Expected error when blob store fails")
	assert.ErrorContains(t, err, "failed to get blob store", "Expected error about blob store failure")
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
			assert.Equalf(t, tc.expected, result, "isAllowedFormat(%q)", tc.format)
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

	require.NotNil(t, result, "Expected to find source variant")
	assert.Equal(t, "source", result.VariantID, "VariantID")
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

	assert.Nil(t, result, "Expected nil when no source variant")
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
			assert.Equalf(t, tc.expected, result, "getExtensionForFormat(%q)", tc.format)
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
			assert.Equalf(t, tc.expected, result, "getMimeTypeForFormat(%q)", tc.format)
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

	assert.True(t, strings.HasPrefix(key, "tmp/"), "Expected temp key to start with 'tmp/'")
	assert.Contains(t, key, "artefact-id", "Expected temp key to contain artefact ID")
	assert.Contains(t, key, "image_w240_webp", "Expected temp key to contain profile name")
}

func TestGenerateFinalStorageKey_IncludesHash(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))
	hash := []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x11, 0x22}

	key := generator.generateFinalStorageKey("images/photo.png", hash, "webp")

	assert.True(t, strings.HasPrefix(key, "generated/"), "Expected final key to start with 'generated/'")
	assert.Contains(t, key, "123456789abcdef0", "Expected final key to contain hash")
	assert.True(t, strings.HasSuffix(key, ".webp"), "Expected final key to end with .webp")
}

func TestGetOrCreateVariantMutex_CreatesNew(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	mu := generator.getOrCreateVariantMutex("new-key")

	assert.NotNil(t, mu, "Expected non-nil mutex")
}

func TestGetOrCreateVariantMutex_ReturnsExisting(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	mu1 := generator.getOrCreateVariantMutex("same-key")
	mu2 := generator.getOrCreateVariantMutex("same-key")

	assert.Same(t, mu1, mu2, "Expected same mutex for same key")
}

func TestCleanupVariantMutex_RemovesMutex(t *testing.T) {
	t.Parallel()

	generator := mustAsGeneratorImpl(t, createTestGenerator(t))

	_ = generator.getOrCreateVariantMutex("test-key")

	generator.cleanupVariantMutex("test-key")

	generator.inProgressMutex.Lock()
	_, exists := generator.inProgress["test-key"]
	generator.inProgressMutex.Unlock()

	assert.False(t, exists, "Expected mutex to be removed after cleanup")
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

	if err != nil {
		require.ErrorIs(t, err, io.EOF, "Unexpected error")
	}
	assert.Equal(t, len(data), n, "Read bytes count")
	assert.Equal(t, int64(len(data)), byteCount, "byteCount")
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

	assert.ErrorIs(t, err, io.EOF, "Expected io.EOF")
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
	require.Truef(t, ok, "expected *onDemandVariantGeneratorImpl, got %T", generator)
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

	require.Error(t, err, "Expected error when write to blob store fails")
	assert.ErrorContains(t, err, "failed to write blob", "Expected error about write failure")
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

	require.Error(t, err, "Expected error when output is zero bytes")
	assert.ErrorContains(t, err, "zero bytes", "Expected error about zero bytes")
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

	require.Error(t, err, "Expected error when rename blob fails")
	assert.ErrorContains(t, err, "failed to rename blob", "Expected error about rename failure")
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
			{VariantID: "source", StorageKey: "source.png", ContentHash: "0000000000000000000000000000000000000000000000000000000000000001"},
		},
	}

	_, err := generator.GenerateVariant(context.Background(), artefact, "image_w240_webp")

	require.Error(t, err, "Expected error when add variant fails")
	assert.ErrorContains(t, err, "failed to add variant", "Expected error about add variant failure")
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
	variant, err := generator.buildVariantRecord("image_w240_webp", "generated/test_01020304.webp", profile, hash, 1024, "source-content-hash")
	require.NoError(t, err, "buildVariantRecord must fingerprint a variant with a source content hash")

	assert.Equal(t, "image_w240_webp", variant.VariantID, "VariantID")
	assert.Equal(t, "generated/test_01020304.webp", variant.StorageKey, "StorageKey")
	assert.Equal(t, "image/webp", variant.MimeType, "MimeType")
	assert.Equal(t, int64(1024), variant.SizeBytes, "SizeBytes")
	assert.Equal(t, "test-backend", variant.StorageBackendID, "StorageBackendID")
	assert.Equal(t, registry_dto.VariantStatusReady, variant.Status, "Status")
	assert.Equal(t, registry_dto.ProducerRuntime, variant.Producer, "Producer want ProducerRuntime")
	assert.Equal(t, registry_dto.KindDerived, variant.Kind, "Kind want KindDerived")
	assert.Equal(t, "source-content-hash", variant.Transform.ParentContentHash, "Transform.ParentContentHash")
	assert.Equal(t, "image-transform", variant.Transform.CapabilityName, "Transform.CapabilityName")
	assert.NotEmpty(t, variant.InputFingerprint,
		"InputFingerprint must be set for a derived variant with a known source hash")
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

	require.Error(t, err, "Expected error when GetVariantData fails")
	assert.ErrorContains(t, err, "failed to get source data", "Expected error about source data failure")
}
