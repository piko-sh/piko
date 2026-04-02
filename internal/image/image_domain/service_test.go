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

package image_domain

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/image/image_dto"
)

type mockTransformer struct {
	errToReturn    error
	mimeType       string
	outputData     []byte
	transformCalls int
	mu             sync.RWMutex
}

func newMockTransformer() *mockTransformer {
	return &mockTransformer{
		mimeType: "image/mock",
	}
}

func (m *mockTransformer) Transform(_ context.Context, input io.Reader, output io.Writer, _ image_dto.TransformationSpec) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transformCalls++

	inBytes, err := io.ReadAll(input)
	if err != nil {
		return "", err
	}

	if m.errToReturn != nil {
		return "", m.errToReturn
	}

	var dataToWrite []byte
	if m.outputData != nil {
		dataToWrite = m.outputData
	} else {
		dataToWrite = inBytes
	}

	if _, err := output.Write(dataToWrite); err != nil {
		return "", err
	}

	return m.mimeType, nil
}

func (m *mockTransformer) GetSupportedFormats() []string {
	return []string{"jpeg", "jpg", "png", "webp", "avif", "gif"}
}

func (m *mockTransformer) GetSupportedModifiers() []string {
	return []string{
		"greyscale", "blur", "sharpen", "rotate", "flip",
		"brightness", "contrast", "saturation",
		"hue", "tint", "gravity", "focus", "radius",
	}
}

func (m *mockTransformer) GetDimensions(_ context.Context, _ io.Reader) (int, int, error) {
	return 800, 600, nil
}

func (m *mockTransformer) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errToReturn = err
}

func (m *mockTransformer) setTransformResult(data []byte, mimeType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputData = data
	m.mimeType = mimeType
}

func (m *mockTransformer) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errToReturn = nil
	m.outputData = nil
	m.mimeType = "image/mock"
	m.transformCalls = 0
}

func (m *mockTransformer) getTransformCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transformCalls
}

func TestNewService(t *testing.T) {
	mockTransformer := newMockTransformer()

	tests := []struct {
		transformers    map[string]TransformerPort
		name            string
		defaultProvider string
		errContains     string
		config          ServiceConfig
		wantErr         bool
	}{
		{
			name: "valid imageService creation",
			transformers: map[string]TransformerPort{
				"mock": mockTransformer,
			},
			defaultProvider: "mock",
			config:          DefaultServiceConfig(),
			wantErr:         false,
		},
		{
			name:            "no transformers provided",
			transformers:    map[string]TransformerPort{},
			defaultProvider: "mock",
			config:          DefaultServiceConfig(),
			wantErr:         true,
			errContains:     "at least one image transformer must be provided",
		},
		{
			name: "empty default provider",
			transformers: map[string]TransformerPort{
				"mock": mockTransformer,
			},
			defaultProvider: "",
			config:          DefaultServiceConfig(),
			wantErr:         true,
			errContains:     "default image provider cannot be empty",
		},
		{
			name: "default provider not registered",
			transformers: map[string]TransformerPort{
				"mock": mockTransformer,
			},
			defaultProvider: "nonexistent",
			config:          DefaultServiceConfig(),
			wantErr:         true,
			errContains:     "default image provider 'nonexistent' is not registered",
		},
		{
			name: "config with zero values uses defaults",
			transformers: map[string]TransformerPort{
				"mock": mockTransformer,
			},
			defaultProvider: "mock",
			config: ServiceConfig{
				MaxImageWidth:    0,
				MaxImageHeight:   0,
				MaxImagePixels:   0,
				MaxFileSizeBytes: 0,
				TransformTimeout: 0,
				AllowedFormats:   nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageService, err := NewService(tt.transformers, tt.defaultProvider, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewService() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewService() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewService() unexpected error = %v", err)
				}
				if imageService == nil {
					t.Errorf("NewService() returned nil imageService")
				}

				s, ok := imageService.(*service)
				if !ok {
					t.Fatal("NewService() should return *service")
				}
				if s.defaultProvider != tt.defaultProvider {
					t.Errorf("NewService() defaultProvider = %q, want %q", s.defaultProvider, tt.defaultProvider)
				}

				if len(s.providerCapabilities) != len(tt.transformers) {
					t.Errorf("NewService() providerCapabilities count = %d, want %d", len(s.providerCapabilities), len(tt.transformers))
				}

				defaults := DefaultServiceConfig()
				if s.config.MaxImageWidth <= 0 {
					t.Errorf("NewService() MaxImageWidth not set to default")
				}
				if s.config.MaxImageHeight <= 0 {
					t.Errorf("NewService() MaxImageHeight not set to default")
				}
				if s.config.MaxImagePixels <= 0 {
					t.Errorf("NewService() MaxImagePixels not set to default")
				}
				if s.config.MaxFileSizeBytes <= 0 {
					t.Errorf("NewService() MaxFileSizeBytes not set to default")
				}
				if s.config.TransformTimeout <= 0 {
					t.Errorf("NewService() TransformTimeout not set to default")
				}
				if len(s.config.AllowedFormats) == 0 {
					t.Errorf("NewService() AllowedFormats not set to default, got %v, want %v", s.config.AllowedFormats, defaults.AllowedFormats)
				}
			}
		})
	}
}

func TestSelectTransformer(t *testing.T) {
	mockTransformer1 := newMockTransformer()
	mockTransformer2 := newMockTransformer()

	imageService, err := NewService(
		map[string]TransformerPort{
			"mock1": mockTransformer1,
			"mock2": mockTransformer2,
		},
		"mock1",
		DefaultServiceConfig(),
	)
	if err != nil {
		t.Fatalf("Failed to create imageService: %v", err)
	}

	s, ok := imageService.(*service)
	if !ok {
		t.Fatal("NewService() should return *service")
	}

	tests := []struct {
		name         string
		wantProvider string
		errContains  string
		spec         image_dto.TransformationSpec
		wantErr      bool
	}{
		{
			name: "use default provider when not specified",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
			},
			wantProvider: "mock1",
			wantErr:      false,
		},
		{
			name: "use specified provider",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "jpeg",
				Provider: "mock2",
			},
			wantProvider: "mock2",
			wantErr:      false,
		},
		{
			name: "provider not found",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "jpeg",
				Provider: "nonexistent",
			},
			wantErr:     true,
			errContains: "image provider 'nonexistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, name, err := s.selectTransformer(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("selectTransformer() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("selectTransformer() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("selectTransformer() unexpected error = %v", err)
				}
				if tr == nil {
					t.Errorf("selectTransformer() returned nil transformer")
				}
				if name != tt.wantProvider {
					t.Errorf("selectTransformer() name = %q, want %q", name, tt.wantProvider)
				}
			}
		})
	}
}

func TestTransformStream(t *testing.T) {
	tests := []struct {
		mockSetup   func(*mockTransformer)
		name        string
		errContains string
		inputData   []byte
		spec        image_dto.TransformationSpec
		wantErr     bool
	}{
		{
			name: "successful transformation",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "cover",
			},
			inputData: []byte("fake image data"),
			mockSetup: func(m *mockTransformer) {
				m.reset()
				m.setTransformResult([]byte("transformed image data"), "image/jpeg")
			},
			wantErr: false,
		},
		{
			name: "invalid spec negative width",
			spec: image_dto.TransformationSpec{
				Width:   -100,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
			},
			inputData: []byte("fake image data"),
			mockSetup: func(m *mockTransformer) {
				m.reset()
			},
			wantErr:     true,
			errContains: "invalid transformation spec",
		},
		{
			name: "transformer returns error",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "cover",
			},
			inputData: []byte("fake image data"),
			mockSetup: func(m *mockTransformer) {
				m.reset()
				m.setError(errors.New("transformation failed"))
			},
			wantErr:     true,
			errContains: "transformation failed",
		},
		{
			name: "uppercase format is normalised",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "JPEG",
				Fit:     "cover",
			},
			inputData: []byte("fake image data"),
			mockSetup: func(m *mockTransformer) {
				m.reset()
				m.setTransformResult([]byte("transformed image data"), "image/jpeg")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransformer := newMockTransformer()
			if tt.mockSetup != nil {
				tt.mockSetup(mockTransformer)
			}

			imageService, err := NewService(
				map[string]TransformerPort{
					"mock": mockTransformer,
				},
				"mock",
				DefaultServiceConfig(),
			)
			if err != nil {
				t.Fatalf("Failed to create imageService: %v", err)
			}

			ctx := context.Background()

			input := bytes.NewReader(tt.inputData)

			result, err := imageService.TransformStream(ctx, input, tt.spec)

			if tt.wantErr {

				if err != nil {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("TransformStream() error = %v, want error containing %q", err, tt.errContains)
					}
				} else if result != nil && result.Body != nil {

					_, readErr := io.ReadAll(result.Body)
					if readErr == nil {
						t.Errorf("TransformStream() expected error but got nil")
					} else if !strings.Contains(readErr.Error(), tt.errContains) {
						t.Errorf("TransformStream() error = %v, want error containing %q", readErr, tt.errContains)
					}
				} else {
					t.Errorf("TransformStream() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("TransformStream() unexpected error = %v", err)
				}
				if result == nil {
					t.Fatal("TransformStream() returned nil result")
				}
				if result.Body == nil {
					t.Errorf("TransformStream() result.Body is nil")
				}
				if result.MIMEType == "" {
					t.Errorf("TransformStream() result.MIMEType is empty")
				}

				outputData, err := io.ReadAll(result.Body)
				if err != nil {
					t.Errorf("Failed to read result body: %v", err)
				}
				if len(outputData) == 0 {
					t.Errorf("TransformStream() returned empty output data")
				}

				calls := mockTransformer.getTransformCalls()
				if calls == 0 {
					t.Errorf("TransformStream() did not call transformer")
				}
			}
		})
	}
}

func TestBuildSidecarKey(t *testing.T) {
	mockTransformer := newMockTransformer()
	imageService, err := NewService(
		map[string]TransformerPort{
			"mock": mockTransformer,
		},
		"mock",
		DefaultServiceConfig(),
	)
	if err != nil {
		t.Fatalf("Failed to create imageService: %v", err)
	}

	s, ok := imageService.(*service)
	if !ok {
		t.Fatal("NewService() should return *service")
	}

	tests := []struct {
		name        string
		originalKey string
		wantContain []string
		spec        image_dto.TransformationSpec
	}{
		{
			name:        "basic transformation",
			originalKey: "path/to/image.jpg",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "webp",
			},
			wantContain: []string{"path/to/image", ".transform_", ".webp"},
		},
		{
			name:        "different original extension",
			originalKey: "folder/photo.png",
			spec: image_dto.TransformationSpec{
				Width:   1024,
				Height:  768,
				Quality: 90,
				Format:  "jpeg",
			},
			wantContain: []string{"folder/photo", ".transform_", ".jpeg"},
		},
		{
			name:        "same spec produces same hash",
			originalKey: "test.jpg",
			spec: image_dto.TransformationSpec{
				Width:   500,
				Height:  500,
				Quality: 80,
				Format:  "png",
			},
			wantContain: []string{"test", ".transform_", ".png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.buildSidecarKey(tt.originalKey, tt.spec)

			for _, part := range tt.wantContain {
				if !strings.Contains(result, part) {
					t.Errorf("buildSidecarKey() = %q, want to contain %q", result, part)
				}
			}

			result2 := s.buildSidecarKey(tt.originalKey, tt.spec)
			if result != result2 {
				t.Errorf("buildSidecarKey() not consistent: first=%q, second=%q", result, result2)
			}
		})
	}

	t.Run("different specs produce different keys", func(t *testing.T) {
		originalKey := "test.jpg"
		spec1 := image_dto.TransformationSpec{
			Width:   800,
			Height:  600,
			Quality: 85,
			Format:  "jpeg",
		}
		spec2 := image_dto.TransformationSpec{
			Width:   1024,
			Height:  768,
			Quality: 85,
			Format:  "jpeg",
		}

		key1 := s.buildSidecarKey(originalKey, spec1)
		key2 := s.buildSidecarKey(originalKey, spec2)

		if key1 == key2 {
			t.Errorf("buildSidecarKey() produced same key for different specs: %q", key1)
		}
	})
}

func TestSpecToMIMEType(t *testing.T) {
	tests := []struct {
		name     string
		wantMIME string
		spec     image_dto.TransformationSpec
	}{
		{
			name: "jpeg format",
			spec: image_dto.TransformationSpec{
				Format: "jpeg",
			},
			wantMIME: "image/jpeg",
		},
		{
			name: "jpg format",
			spec: image_dto.TransformationSpec{
				Format: "jpg",
			},
			wantMIME: "image/jpeg",
		},
		{
			name: "png format",
			spec: image_dto.TransformationSpec{
				Format: "png",
			},
			wantMIME: "image/png",
		},
		{
			name: "webp format",
			spec: image_dto.TransformationSpec{
				Format: "webp",
			},
			wantMIME: "image/webp",
		},
		{
			name: "avif format",
			spec: image_dto.TransformationSpec{
				Format: "avif",
			},
			wantMIME: "image/avif",
		},
		{
			name: "gif format",
			spec: image_dto.TransformationSpec{
				Format: "gif",
			},
			wantMIME: "image/gif",
		},
		{
			name: "uppercase format",
			spec: image_dto.TransformationSpec{
				Format: "JPEG",
			},
			wantMIME: "image/jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMIME := specToMIMEType(tt.spec)
			if gotMIME != tt.wantMIME {
				t.Errorf("specToMIMEType() = %q, want %q", gotMIME, tt.wantMIME)
			}
		})
	}
}

func TestGetFallbackIcon(t *testing.T) {
	mockTransformer := newMockTransformer()

	config := DefaultServiceConfig()
	config.FallbackIconPaths = map[string]string{}

	imageService, err := NewService(
		map[string]TransformerPort{
			"mock": mockTransformer,
		},
		"mock",
		config,
	)
	if err != nil {
		t.Fatalf("Failed to create imageService: %v", err)
	}

	s, ok := imageService.(*service)
	if !ok {
		t.Fatal("NewService() should return *service")
	}

	tests := []struct {
		name        string
		contentType string
		expectNil   bool
	}{
		{
			name:        "application/pdf returns nil when no icons configured",
			contentType: "application/pdf",
			expectNil:   true,
		},
		{
			name:        "application/zip returns nil when no icons configured",
			contentType: "application/zip",
			expectNil:   true,
		},
		{
			name:        "default returns nil when no icons configured",
			contentType: "unknown/type",
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.getFallbackIcon(tt.contentType)
			if tt.expectNil && result != nil {
				t.Errorf("getFallbackIcon() = %v, want nil", result)
			}
		})
	}
}

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()

	if config.MaxImageWidth <= 0 {
		t.Errorf("DefaultServiceConfig() MaxImageWidth = %d, want > 0", config.MaxImageWidth)
	}
	if config.MaxImageHeight <= 0 {
		t.Errorf("DefaultServiceConfig() MaxImageHeight = %d, want > 0", config.MaxImageHeight)
	}
	if config.MaxImagePixels <= 0 {
		t.Errorf("DefaultServiceConfig() MaxImagePixels = %d, want > 0", config.MaxImagePixels)
	}
	if config.MaxFileSizeBytes <= 0 {
		t.Errorf("DefaultServiceConfig() MaxFileSizeBytes = %d, want > 0", config.MaxFileSizeBytes)
	}
	if config.TransformTimeout <= 0 {
		t.Errorf("DefaultServiceConfig() TransformTimeout = %v, want > 0", config.TransformTimeout)
	}
	if len(config.AllowedFormats) == 0 {
		t.Errorf("DefaultServiceConfig() AllowedFormats is empty, want some formats")
	}
	if config.FallbackIconPaths == nil {
		t.Errorf("DefaultServiceConfig() FallbackIconPaths is nil, want empty map")
	}
}

func TestParseDensity(t *testing.T) {
	tests := []struct {
		name    string
		density string
		want    float64
	}{
		{
			name:    "x1 density",
			density: "x1",
			want:    1.0,
		},
		{
			name:    "x2 density",
			density: "x2",
			want:    2.0,
		},
		{
			name:    "x3 density",
			density: "x3",
			want:    3.0,
		},
		{
			name:    "1x density",
			density: "1x",
			want:    1.0,
		},
		{
			name:    "2x density",
			density: "2x",
			want:    2.0,
		},
		{
			name:    "just number",
			density: "2",
			want:    2.0,
		},
		{
			name:    "decimal density",
			density: "1.5",
			want:    1.5,
		},
		{
			name:    "uppercase X",
			density: "X2",
			want:    2.0,
		},
		{
			name:    "invalid density returns default",
			density: "invalid",
			want:    1.0,
		},
		{
			name:    "zero density returns default",
			density: "0",
			want:    1.0,
		},
		{
			name:    "negative density returns default",
			density: "-1",
			want:    1.0,
		},
		{
			name:    "empty string returns default",
			density: "",
			want:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDensity(tt.density)
			if got != tt.want {
				t.Errorf("parseDensity(%q) = %v, want %v", tt.density, got, tt.want)
			}
		})
	}
}

func TestParseSizes(t *testing.T) {
	tests := []struct {
		screens map[string]int
		name    string
		sizes   string
		wantLen int
	}{
		{
			name:    "empty sizes returns nil",
			sizes:   "",
			screens: nil,
			wantLen: 0,
		},
		{
			name:    "with sizes string returns default widths",
			sizes:   "100vw sm:50vw md:400px",
			screens: nil,
			wantLen: 6,
		},
		{
			name:  "custom screens still returns default widths",
			sizes: "100vw",
			screens: map[string]int{
				"sm": 640,
				"md": 1024,
			},
			wantLen: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSizes(tt.sizes, tt.screens)
			if len(got) != tt.wantLen {
				t.Errorf("parseSizes() returned %d widths, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestNewServiceWithDefaultTransformer(t *testing.T) {
	t.Parallel()

	imageService := NewServiceWithDefaultTransformer("custom")
	require.NotNil(t, imageService)

	s, ok := imageService.(*service)
	require.True(t, ok)
	assert.Equal(t, "custom", s.defaultProvider)
	assert.NotNil(t, s.transformers)
	assert.Empty(t, s.transformers)
}

func TestService_GenerateResponsiveVariants(t *testing.T) {
	t.Parallel()

	t.Run("nil Responsive spec returns error", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:      800,
			Format:     "webp",
			Quality:    80,
			Fit:        "cover",
			Responsive: nil,
		}

		_, err := service.GenerateResponsiveVariants(ctx, strings.NewReader("img"), spec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "responsive spec is nil")
	})

	t.Run("valid spec generates variants", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("variant-data"), "image/webp")
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Responsive: &image_dto.ResponsiveSpec{
				Sizes:     "100vw",
				Densities: []string{"x1"},
			},
		}

		variants, err := service.GenerateResponsiveVariants(ctx, strings.NewReader("img"), spec)
		require.NoError(t, err)
		assert.NotEmpty(t, variants)
	})

	t.Run("default densities when none specified", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("data"), "image/webp")
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Responsive: &image_dto.ResponsiveSpec{
				Sizes:     "100vw",
				Densities: nil,
			},
		}

		variants, err := service.GenerateResponsiveVariants(ctx, strings.NewReader("img"), spec)
		require.NoError(t, err)
		assert.NotEmpty(t, variants)
	})

	t.Run("transformer error propagated", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setError(errors.New("transform failed"))
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Responsive: &image_dto.ResponsiveSpec{
				Sizes:     "100vw",
				Densities: []string{"x1"},
			},
		}

		_, err := service.GenerateResponsiveVariants(ctx, strings.NewReader("img"), spec)
		require.Error(t, err)
	})

	t.Run("empty sizes falls back to base width", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("data"), "image/webp")
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   400,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Responsive: &image_dto.ResponsiveSpec{
				Sizes:     "",
				Densities: []string{"x1"},
			},
		}

		variants, err := service.GenerateResponsiveVariants(ctx, strings.NewReader("img"), spec)
		require.NoError(t, err)
		assert.Len(t, variants, 1)
		assert.Equal(t, 400, variants[0].Width)
	})
}

func TestService_GeneratePlaceholder(t *testing.T) {
	t.Parallel()

	t.Run("nil Placeholder spec returns error", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:       800,
			Format:      "webp",
			Quality:     80,
			Fit:         "cover",
			Placeholder: nil,
		}

		_, err := service.GeneratePlaceholder(ctx, strings.NewReader("img"), spec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "placeholder spec is not enabled")
	})

	t.Run("disabled Placeholder returns error", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Placeholder: &image_dto.PlaceholderSpec{
				Enabled: false,
			},
		}

		_, err := service.GeneratePlaceholder(ctx, strings.NewReader("img"), spec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "placeholder spec is not enabled")
	})

	t.Run("valid spec returns data URL", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("tiny-image"), "image/webp")
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Placeholder: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     20,
				Quality:   10,
				BlurSigma: 5.0,
			},
		}

		dataURL, err := service.GeneratePlaceholder(ctx, strings.NewReader("img"), spec)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(dataURL, "data:"))
		assert.Contains(t, dataURL, ";base64,")
	})

	t.Run("zero placeholder fields use defaults", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("tiny"), "image/webp")
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Placeholder: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     0,
				Quality:   0,
				BlurSigma: 0,
			},
		}

		dataURL, err := service.GeneratePlaceholder(ctx, strings.NewReader("img"), spec)
		require.NoError(t, err)
		assert.NotEmpty(t, dataURL)
	})

	t.Run("transformer error propagated", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setError(errors.New("placeholder transform failed"))
		service := createService(t, mock)

		ctx := context.Background()

		spec := image_dto.TransformationSpec{
			Width:   800,
			Format:  "webp",
			Quality: 80,
			Fit:     "cover",
			Placeholder: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     20,
				Quality:   10,
				BlurSigma: 5.0,
			},
		}

		_, err := service.GeneratePlaceholder(ctx, strings.NewReader("img"), spec)
		require.Error(t, err)
	})
}

func TestEncodePlaceholderAsDataURL(t *testing.T) {
	t.Parallel()

	t.Run("encodes bytes to data URL", func(t *testing.T) {
		t.Parallel()

		result := encodePlaceholderAsDataURL("image/png", []byte("hello"))
		assert.True(t, strings.HasPrefix(result, "data:image/png;base64,"))
		assert.NotEqual(t, "data:image/png;base64,", result)
	})

	t.Run("empty data produces valid data URL", func(t *testing.T) {
		t.Parallel()

		result := encodePlaceholderAsDataURL("image/png", []byte{})
		assert.Equal(t, "data:image/png;base64,", result)
	})
}

func TestService_GetDimensions(t *testing.T) {
	t.Parallel()

	t.Run("valid provider returns dimensions", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		service := createService(t, mock)

		ctx := context.Background()

		w, h, err := service.GetDimensions(ctx, strings.NewReader("img"))

		require.NoError(t, err)
		assert.Equal(t, 800, w)
		assert.Equal(t, 600, h)
	})

	t.Run("no provider returns error", func(t *testing.T) {
		t.Parallel()

		service := NewServiceWithDefaultTransformer("missing")

		ctx := context.Background()

		_, _, err := service.GetDimensions(ctx, strings.NewReader("img"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no image provider configured")
	})
}

func TestService_Name(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	imageService := createService(t, mock)

	s, ok := imageService.(*service)
	require.True(t, ok)
	assert.Equal(t, "ImageService", s.Name())
}

func TestService_Check(t *testing.T) {
	t.Parallel()

	t.Run("with transformers returns healthy", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		imageService := createService(t, mock)

		s, ok := imageService.(*service)
		require.True(t, ok)
		status := s.Check(context.Background(), healthprobe_dto.CheckType(""))

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Contains(t, status.Message, "1 transformer")
		assert.Equal(t, "ImageService", status.Name)
	})

	t.Run("without transformers returns unhealthy", func(t *testing.T) {
		t.Parallel()

		imageService := NewServiceWithDefaultTransformer("none")
		s, ok := imageService.(*service)
		require.True(t, ok)
		status := s.Check(context.Background(), healthprobe_dto.CheckType(""))

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Contains(t, status.Message, "No image transformers")
	})
}

func TestSpecToMIMEType_DefaultCase(t *testing.T) {
	t.Parallel()

	spec := image_dto.TransformationSpec{Format: "tiff"}
	result := specToMIMEType(spec)
	assert.NotEmpty(t, result)
}

func TestGetFallbackIcon_WithIcons(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	imageService, err := NewService(
		map[string]TransformerPort{"mock": mock},
		"mock",
		DefaultServiceConfig(),
	)
	require.NoError(t, err)

	s, ok := imageService.(*service)
	require.True(t, ok)
	s.fallbackIcons = map[string][]byte{
		"application/pdf":          []byte("pdf-icon"),
		image_dto.ImageNameDefault: []byte("default-icon"),
	}

	t.Run("matching prefix returns specific icon", func(t *testing.T) {
		t.Parallel()

		result := s.getFallbackIcon("application/pdf")
		assert.Equal(t, []byte("pdf-icon"), result)
	})

	t.Run("no matching prefix returns default icon", func(t *testing.T) {
		t.Parallel()

		result := s.getFallbackIcon("video/mp4")
		assert.Equal(t, []byte("default-icon"), result)
	})
}

func createService(t *testing.T, mock *mockTransformer) Service {
	t.Helper()

	service, err := NewService(
		map[string]TransformerPort{"mock": mock},
		"mock",
		DefaultServiceConfig(),
	)
	require.NoError(t, err)

	return service
}
