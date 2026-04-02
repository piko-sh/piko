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
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/image/image_dto"
)

type stubTransformer struct{}

func (m *stubTransformer) Transform(_ context.Context, _ io.Reader, _ io.Writer, _ image_dto.TransformationSpec) (string, error) {
	return "image/webp", nil
}

func (m *stubTransformer) GetSupportedFormats() []string {
	return []string{"jpeg", "png", "webp"}
}

func (m *stubTransformer) GetSupportedModifiers() []string {
	return []string{"blur", "greyscale"}
}

func (m *stubTransformer) GetDimensions(_ context.Context, _ io.Reader) (int, int, error) {
	return 800, 600, nil
}

func TestVariantBuilder_BasicUsage(t *testing.T) {
	spec := Variant().
		Size(200, 200).
		Format("webp").
		Quality(80).
		Cover().
		Build()

	if spec.Width != 200 {
		t.Errorf("expected width 200, got %d", spec.Width)
	}
	if spec.Height != 200 {
		t.Errorf("expected height 200, got %d", spec.Height)
	}
	if spec.Format != "webp" {
		t.Errorf("expected format webp, got %s", spec.Format)
	}
	if spec.Quality != 80 {
		t.Errorf("expected quality 80, got %d", spec.Quality)
	}
	if spec.Fit != image_dto.FitCover {
		t.Errorf("expected fit cover, got %s", spec.Fit)
	}
}

func TestVariantBuilder_MaxWidth(t *testing.T) {
	spec := Variant().MaxWidth(1920).Build()

	if spec.Width != 1920 {
		t.Errorf("expected width 1920, got %d", spec.Width)
	}
	if spec.Height != 0 {
		t.Errorf("expected height 0 (preserve aspect), got %d", spec.Height)
	}
}

func TestVariantBuilder_MaxHeight(t *testing.T) {
	spec := Variant().MaxHeight(1080).Build()

	if spec.Width != 0 {
		t.Errorf("expected width 0 (preserve aspect), got %d", spec.Width)
	}
	if spec.Height != 1080 {
		t.Errorf("expected height 1080, got %d", spec.Height)
	}
}

func TestVariantBuilder_FitModes(t *testing.T) {
	testCases := []struct {
		name     string
		builder  func() *VariantBuilder
		expected image_dto.FitMode
	}{
		{
			name:     "Cover",
			builder:  func() *VariantBuilder { return Variant().Cover() },
			expected: "cover",
		},
		{
			name:     "Contain",
			builder:  func() *VariantBuilder { return Variant().Contain() },
			expected: "contain",
		},
		{
			name:     "Fill",
			builder:  func() *VariantBuilder { return Variant().Fill() },
			expected: "fill",
		},
		{
			name:     "Inside",
			builder:  func() *VariantBuilder { return Variant().Inside() },
			expected: "inside",
		},
		{
			name:     "Outside",
			builder:  func() *VariantBuilder { return Variant().Outside() },
			expected: "outside",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec := tc.builder().Build()
			if spec.Fit != tc.expected {
				t.Errorf("expected fit %s, got %s", tc.expected, spec.Fit)
			}
		})
	}
}

func TestVariantBuilder_Modifiers(t *testing.T) {
	spec := Variant().
		Blur(5.0).
		Greyscale().
		WithModifier("custom", "value").
		Build()

	if spec.Modifiers["blur"] != "5.0" {
		t.Errorf("expected blur 5.0, got %s", spec.Modifiers["blur"])
	}
	if spec.Modifiers["greyscale"] != "true" {
		t.Errorf("expected greyscale true, got %s", spec.Modifiers["greyscale"])
	}
	if spec.Modifiers["custom"] != "value" {
		t.Errorf("expected custom value, got %s", spec.Modifiers["custom"])
	}
}

func TestVariantBuilder_WithoutEnlargement(t *testing.T) {
	spec := Variant().WithoutEnlargement().Build()

	if !spec.WithoutEnlargement {
		t.Error("expected WithoutEnlargement to be true")
	}
}

func TestVariantBuilder_Background(t *testing.T) {
	spec := Variant().Background("#FF0000").Build()

	if spec.Background != "#FF0000" {
		t.Errorf("expected background #FF0000, got %s", spec.Background)
	}
}

func TestVariantBuilder_AspectRatio(t *testing.T) {
	spec := Variant().AspectRatio("16:9").Build()

	if spec.AspectRatio != "16:9" {
		t.Errorf("expected aspect ratio 16:9, got %s", spec.AspectRatio)
	}
}

func TestVariantBuilder_FromSpec(t *testing.T) {
	original := image_dto.TransformationSpec{
		Width:   100,
		Height:  100,
		Format:  "jpeg",
		Quality: 75,
	}

	spec := Variant().FromSpec(original).Quality(90).Build()

	if spec.Width != 100 {
		t.Errorf("expected width 100, got %d", spec.Width)
	}
	if spec.Quality != 90 {
		t.Errorf("expected quality 90, got %d", spec.Quality)
	}
}

func TestImageConfigBuilder_BasicUsage(t *testing.T) {
	transformer := &stubTransformer{}

	config, err := Image().
		Provider("mock", transformer).
		MaxFileSizeMB(50).
		MaxDimensions(4096, 4096).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(config.Providers))
	}
	if config.DefaultProvider != "mock" {
		t.Errorf("expected default provider mock, got %s", config.DefaultProvider)
	}
	if config.ServiceConfig.MaxFileSizeBytes != 50*1024*1024 {
		t.Errorf("expected max file size 50MB, got %d", config.ServiceConfig.MaxFileSizeBytes)
	}
	if config.ServiceConfig.MaxImageWidth != 4096 {
		t.Errorf("expected max width 4096, got %d", config.ServiceConfig.MaxImageWidth)
	}
}

func TestImageConfigBuilder_MultipleProviders(t *testing.T) {
	transformer1 := &stubTransformer{}
	transformer2 := &stubTransformer{}

	config, err := Image().
		Provider("mock1", transformer1).
		Provider("mock2", transformer2).
		DefaultProvider("mock2").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(config.Providers))
	}
	if config.DefaultProvider != "mock2" {
		t.Errorf("expected default provider mock2, got %s", config.DefaultProvider)
	}
}

func TestImageConfigBuilder_WithVariant(t *testing.T) {
	transformer := &stubTransformer{}

	config, err := Image().
		Provider("mock", transformer).
		WithVariant("thumb", Variant().Size(100, 100).Cover().Build()).
		WithVariantBuilder("avatar", Variant().Size(64, 64).Format("webp")).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.PredefinedVariants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(config.PredefinedVariants))
	}

	thumb, ok := config.GetVariant("thumb")
	if !ok {
		t.Error("expected thumb variant to exist")
	}
	if thumb.Width != 100 {
		t.Errorf("expected thumb width 100, got %d", thumb.Width)
	}

	avatar, ok := config.GetVariant("avatar")
	if !ok {
		t.Error("expected avatar variant to exist")
	}
	if avatar.Format != "webp" {
		t.Errorf("expected avatar format webp, got %s", avatar.Format)
	}
}

func TestImageConfigBuilder_TransformTimeout(t *testing.T) {
	transformer := &stubTransformer{}

	config, err := Image().
		Provider("mock", transformer).
		TransformTimeout(60 * time.Second).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ServiceConfig.TransformTimeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", config.ServiceConfig.TransformTimeout)
	}
}

func TestImageConfigBuilder_AllowedFormats(t *testing.T) {
	transformer := &stubTransformer{}

	config, err := Image().
		Provider("mock", transformer).
		AllowedFormats("webp", "jpeg").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.ServiceConfig.AllowedFormats) != 2 {
		t.Errorf("expected 2 allowed formats, got %d", len(config.ServiceConfig.AllowedFormats))
	}
}

func TestImageConfigBuilder_FromDefaults(t *testing.T) {
	transformer := &stubTransformer{}

	config, err := Image().
		FromDefaults().
		Provider("mock", transformer).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.PredefinedVariants) == 0 {
		t.Error("expected default predefined variants")
	}

	if _, ok := config.PredefinedVariants["thumb_100"]; !ok {
		t.Error("expected thumb_100 default variant")
	}
	if _, ok := config.PredefinedVariants["thumb_200"]; !ok {
		t.Error("expected thumb_200 default variant")
	}
}

func TestImageConfigBuilder_ValidationErrors(t *testing.T) {
	testCases := []struct {
		builder func() (*ImageConfig, error)
		name    string
		wantErr bool
	}{
		{
			name: "no providers",
			builder: func() (*ImageConfig, error) {
				return Image().Build()
			},
			wantErr: true,
		},
		{
			name: "empty provider name",
			builder: func() (*ImageConfig, error) {
				return Image().Provider("", &stubTransformer{}).Build()
			},
			wantErr: true,
		},
		{
			name: "nil transformer",
			builder: func() (*ImageConfig, error) {
				return Image().Provider("test", nil).Build()
			},
			wantErr: true,
		},
		{
			name: "default provider not registered",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					DefaultProvider("nonexistent").
					Build()
			},
			wantErr: true,
		},
		{
			name: "empty variant name",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					WithVariant("", Variant().Build()).
					Build()
			},
			wantErr: true,
		},
		{
			name: "nil variant builder",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					WithVariantBuilder("test", nil).
					Build()
			},
			wantErr: true,
		},
		{
			name: "invalid format in allowed formats",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					AllowedFormats("invalid_format").
					Build()
			},
			wantErr: true,
		},
		{
			name: "variant with negative width",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					WithVariant("test", image_dto.TransformationSpec{Width: -100}).
					Build()
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.builder()
			if tc.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestImageConfigBuilder_MustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but got none")
		}
	}()

	Image().MustBuild()
}

func TestDefaultPredefinedVariants(t *testing.T) {
	variants := DefaultPredefinedVariants()

	expectedVariants := []string{"thumb_100", "thumb_200", "thumb_400", "preview_800", "lqip"}

	for _, name := range expectedVariants {
		if _, ok := variants[name]; !ok {
			t.Errorf("expected variant %s to be in defaults", name)
		}
	}

	lqip := variants["lqip"]
	if lqip.Quality > 30 {
		t.Errorf("expected LQIP quality to be low, got %d", lqip.Quality)
	}
	if lqip.Width > 50 {
		t.Errorf("expected LQIP width to be small, got %d", lqip.Width)
	}
}

func TestImageConfigBuilder_MaxPixels(t *testing.T) {
	t.Parallel()

	config, err := Image().
		Provider("mock", &stubTransformer{}).
		MaxPixels(50_000_000).
		Build()

	require.NoError(t, err)
	assert.Equal(t, int64(50_000_000), config.ServiceConfig.MaxImagePixels)
}

func TestImageConfigBuilder_MaxFileSizeBytes(t *testing.T) {
	t.Parallel()

	config, err := Image().
		Provider("mock", &stubTransformer{}).
		MaxFileSizeBytes(100_000).
		Build()

	require.NoError(t, err)
	assert.Equal(t, int64(100_000), config.ServiceConfig.MaxFileSizeBytes)
}

func TestImageConfigBuilder_DefaultQuality(t *testing.T) {
	t.Parallel()

	t.Run("valid quality builds successfully", func(t *testing.T) {
		t.Parallel()

		_, err := Image().
			Provider("mock", &stubTransformer{}).
			DefaultQuality(85).
			Build()

		assert.NoError(t, err)
	})

	t.Run("zero quality returns error", func(t *testing.T) {
		t.Parallel()

		_, err := Image().
			Provider("mock", &stubTransformer{}).
			DefaultQuality(0).
			Build()

		assert.Error(t, err)
	})

	t.Run("over 100 quality returns error", func(t *testing.T) {
		t.Parallel()

		_, err := Image().
			Provider("mock", &stubTransformer{}).
			DefaultQuality(101).
			Build()

		assert.Error(t, err)
	})
}

func TestImageConfigBuilder_WithFallbackIcon(t *testing.T) {
	t.Parallel()

	t.Run("adds icon path to config", func(t *testing.T) {
		t.Parallel()

		builder := Image().
			Provider("mock", &stubTransformer{}).
			WithFallbackIcon("application/pdf", "/icons/pdf.svg")

		assert.Equal(t, "/icons/pdf.svg", builder.config.FallbackIconPaths["application/pdf"])
	})

	t.Run("initialises nil map on first call", func(t *testing.T) {
		t.Parallel()

		builder := Image().
			Provider("mock", &stubTransformer{})
		builder.config.FallbackIconPaths = nil

		builder.WithFallbackIcon("application/pdf", "/icons/pdf.svg")
		assert.NotNil(t, builder.config.FallbackIconPaths)
		assert.Equal(t, "/icons/pdf.svg", builder.config.FallbackIconPaths["application/pdf"])
	})
}

func TestImageConfigBuilder_FromConfig(t *testing.T) {
	t.Parallel()

	customConfig := ServiceConfig{
		MaxImageWidth:    2048,
		MaxImageHeight:   2048,
		MaxImagePixels:   10_000_000,
		MaxFileSizeBytes: 25 * 1024 * 1024,
		TransformTimeout: 15 * time.Second,
		AllowedFormats:   []string{"jpeg", "png"},
	}

	config, err := Image().
		Provider("mock", &stubTransformer{}).
		FromConfig(customConfig).
		Build()

	require.NoError(t, err)
	assert.Equal(t, 2048, config.ServiceConfig.MaxImageWidth)
	assert.Equal(t, 2048, config.ServiceConfig.MaxImageHeight)
	assert.Equal(t, int64(10_000_000), config.ServiceConfig.MaxImagePixels)
}

func TestImageConfig_GetVariant_NilMap(t *testing.T) {
	t.Parallel()

	config := &ImageConfig{
		PredefinedVariants: nil,
	}

	_, ok := config.GetVariant("thumb")
	assert.False(t, ok)
}

func TestImageConfigBuilder_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		builder func() (*ImageConfig, error)
		name    string
	}{
		{
			name: "negative MaxImageWidth",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					MaxDimensions(-1, 100).
					Build()
			},
		},
		{
			name: "negative MaxImageHeight",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					MaxDimensions(100, -1).
					Build()
			},
		},
		{
			name: "negative MaxImagePixels",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					MaxPixels(-1).
					Build()
			},
		},
		{
			name: "negative MaxFileSizeBytes",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					MaxFileSizeBytes(-1).
					Build()
			},
		},
		{
			name: "negative TransformTimeout",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					TransformTimeout(-1 * time.Second).
					Build()
			},
		},
		{
			name: "variant with negative height",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					WithVariant("bad", image_dto.TransformationSpec{Height: -1}).
					Build()
			},
		},
		{
			name: "variant with invalid quality",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					WithVariant("bad", image_dto.TransformationSpec{Quality: 101}).
					Build()
			},
		},
		{
			name: "variant with invalid format",
			builder: func() (*ImageConfig, error) {
				return Image().
					Provider("mock", &stubTransformer{}).
					WithVariant("bad", image_dto.TransformationSpec{Format: "bmp"}).
					Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.builder()
			assert.Error(t, err)
		})
	}
}

func TestVariantBuilder_Width(t *testing.T) {
	t.Parallel()

	spec := Variant().Width(640).Build()
	assert.Equal(t, 640, spec.Width)
}

func TestVariantBuilder_Height(t *testing.T) {
	t.Parallel()

	spec := Variant().Height(480).Build()
	assert.Equal(t, 480, spec.Height)
}

func TestVariantBuilder_Fit(t *testing.T) {
	t.Parallel()

	spec := Variant().Fit("cover").Build()
	assert.Equal(t, image_dto.FitCover, spec.Fit)
}

func TestVariantBuilder_Provider(t *testing.T) {
	t.Parallel()

	spec := Variant().Provider("vips").Build()
	assert.Equal(t, "vips", spec.Provider)
}

func TestVariantBuilder_NilModifiersInit(t *testing.T) {
	t.Parallel()

	t.Run("WithModifier on fresh builder initialises map", func(t *testing.T) {
		t.Parallel()

		spec := Variant().WithModifier("key", "val").Build()
		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "val", spec.Modifiers["key"])
	})

	t.Run("Blur on fresh builder initialises map", func(t *testing.T) {
		t.Parallel()

		spec := Variant().Blur(3.0).Build()
		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "3.0", spec.Modifiers["blur"])
	})

	t.Run("Greyscale on fresh builder initialises map", func(t *testing.T) {
		t.Parallel()

		spec := Variant().Greyscale().Build()
		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "true", spec.Modifiers["greyscale"])
	})
}
