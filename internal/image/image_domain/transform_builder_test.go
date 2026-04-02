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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/image/image_dto"
)

func newTestService(t *testing.T, mock *mockTransformer) Service {
	t.Helper()

	service, err := NewService(
		map[string]TransformerPort{"mock": mock},
		"mock",
		DefaultServiceConfig(),
	)
	require.NoError(t, err)

	return service
}

func TestTransformBuilder_DimensionSetters(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	tests := []struct {
		build      func(*TransformBuilder) *TransformBuilder
		name       string
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "Width sets target width",
			build:      func(b *TransformBuilder) *TransformBuilder { return b.Width(800) },
			wantWidth:  800,
			wantHeight: 0,
		},
		{
			name:       "Height sets target height",
			build:      func(b *TransformBuilder) *TransformBuilder { return b.Height(600) },
			wantWidth:  0,
			wantHeight: 600,
		},
		{
			name:       "Size sets both dimensions",
			build:      func(b *TransformBuilder) *TransformBuilder { return b.Size(800, 600) },
			wantWidth:  800,
			wantHeight: 600,
		},
		{
			name:       "MaxWidth sets width and clears height",
			build:      func(b *TransformBuilder) *TransformBuilder { return b.MaxWidth(1920) },
			wantWidth:  1920,
			wantHeight: 0,
		},
		{
			name:       "MaxHeight clears width and sets height",
			build:      func(b *TransformBuilder) *TransformBuilder { return b.MaxHeight(1080) },
			wantWidth:  0,
			wantHeight: 1080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := service.Transform(strings.NewReader("test"))
			result := tt.build(builder)
			spec := result.Spec()

			assert.Equal(t, tt.wantWidth, spec.Width)
			assert.Equal(t, tt.wantHeight, spec.Height)
		})
	}
}

func TestTransformBuilder_FormatAndQuality(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	tests := []struct {
		name        string
		build       func(*TransformBuilder) *TransformBuilder
		wantFormat  string
		wantQuality int
	}{
		{
			name:        "Format sets output format",
			build:       func(b *TransformBuilder) *TransformBuilder { return b.Format("webp") },
			wantFormat:  "webp",
			wantQuality: 80,
		},
		{
			name:        "Quality sets compression quality",
			build:       func(b *TransformBuilder) *TransformBuilder { return b.Quality(85) },
			wantFormat:  "webp",
			wantQuality: 85,
		},
		{
			name: "Chaining format and quality",
			build: func(b *TransformBuilder) *TransformBuilder {
				return b.Format("jpeg").Quality(90)
			},
			wantFormat:  "jpeg",
			wantQuality: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := service.Transform(strings.NewReader("test"))
			result := tt.build(builder)
			spec := result.Spec()

			assert.Equal(t, tt.wantFormat, spec.Format)
			assert.Equal(t, tt.wantQuality, spec.Quality)
		})
	}
}

func TestTransformBuilder_FitModes(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	tests := []struct {
		name    string
		build   func(*TransformBuilder) *TransformBuilder
		wantFit image_dto.FitMode
	}{
		{
			name:    "Fit sets arbitrary fit mode",
			build:   func(b *TransformBuilder) *TransformBuilder { return b.Fit(image_dto.FitCover) },
			wantFit: image_dto.FitCover,
		},
		{
			name:    "Cover shorthand",
			build:   func(b *TransformBuilder) *TransformBuilder { return b.Cover() },
			wantFit: "cover",
		},
		{
			name:    "Contain shorthand",
			build:   func(b *TransformBuilder) *TransformBuilder { return b.Contain() },
			wantFit: "contain",
		},
		{
			name:    "Fill shorthand",
			build:   func(b *TransformBuilder) *TransformBuilder { return b.Fill() },
			wantFit: "fill",
		},
		{
			name:    "Inside shorthand",
			build:   func(b *TransformBuilder) *TransformBuilder { return b.Inside() },
			wantFit: "inside",
		},
		{
			name:    "Outside shorthand",
			build:   func(b *TransformBuilder) *TransformBuilder { return b.Outside() },
			wantFit: "outside",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := service.Transform(strings.NewReader("test"))
			result := tt.build(builder)
			spec := result.Spec()

			assert.Equal(t, tt.wantFit, spec.Fit)
		})
	}
}

func TestTransformBuilder_VisualSettings(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	t.Run("WithoutEnlargement", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).WithoutEnlargement().Spec()
		assert.True(t, spec.WithoutEnlargement)
	})

	t.Run("Background", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).Background("#FF0000").Spec()
		assert.Equal(t, "#FF0000", spec.Background)
	})

	t.Run("AspectRatio", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).AspectRatio("16:9").Spec()
		assert.Equal(t, "16:9", spec.AspectRatio)
	})

	t.Run("Provider", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).Provider("vips").Spec()
		assert.Equal(t, "vips", spec.Provider)
	})
}

func TestTransformBuilder_Modifiers(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	t.Run("WithModifier initialises map and sets value", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).
			WithModifier("custom", "val").
			Spec()

		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "val", spec.Modifiers["custom"])
	})

	t.Run("Blur sets blur modifier", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).
			Blur(5.0).
			Spec()

		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "5.0", spec.Modifiers["blur"])
	})

	t.Run("Greyscale sets greyscale modifier", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).
			Greyscale().
			Spec()

		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "true", spec.Modifiers["greyscale"])
	})

	t.Run("chaining multiple modifiers", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).
			Blur(3.0).
			Greyscale().
			WithModifier("sharpen", "2").
			Spec()

		require.NotNil(t, spec.Modifiers)
		assert.Equal(t, "3.0", spec.Modifiers["blur"])
		assert.Equal(t, "true", spec.Modifiers["greyscale"])
		assert.Equal(t, "2", spec.Modifiers["sharpen"])
	})
}

func TestTransformBuilder_VariantOperations(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	thumbSpec := image_dto.TransformationSpec{
		Width:   200,
		Height:  200,
		Format:  "webp",
		Quality: 80,
		Fit:     "cover",
	}

	variants := map[string]image_dto.TransformationSpec{
		"thumb": thumbSpec,
	}

	t.Run("UseVariant applies predefined variant", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).
			WithPredefinedVariants(variants).
			UseVariant("thumb").
			Spec()

		assert.Equal(t, 200, spec.Width)
		assert.Equal(t, 200, spec.Height)
		assert.Equal(t, image_dto.FitCover, spec.Fit)
	})

	t.Run("UseVariant with missing name is no-op", func(t *testing.T) {
		t.Parallel()

		builder := service.Transform(strings.NewReader("test")).
			WithPredefinedVariants(variants).
			Width(800)

		specBefore := builder.Spec()
		builder.UseVariant("nonexistent")
		specAfter := builder.Spec()

		assert.Equal(t, specBefore.Width, specAfter.Width)
	})

	t.Run("UseVariant with nil variants is no-op", func(t *testing.T) {
		t.Parallel()

		builder := service.Transform(strings.NewReader("test")).Width(800)
		specBefore := builder.Spec()
		builder.UseVariant("thumb")
		specAfter := builder.Spec()

		assert.Equal(t, specBefore.Width, specAfter.Width)
	})

	t.Run("WithPredefinedVariants sets variants map", func(t *testing.T) {
		t.Parallel()

		builder := service.Transform(strings.NewReader("test")).
			WithPredefinedVariants(variants)

		assert.NotNil(t, builder.predefinedVariants)
		assert.Len(t, builder.predefinedVariants, 1)
	})

	t.Run("FromSpec copies existing spec", func(t *testing.T) {
		t.Parallel()

		existingSpec := image_dto.TransformationSpec{
			Width:   1024,
			Height:  768,
			Format:  "jpeg",
			Quality: 90,
		}

		spec := service.Transform(strings.NewReader("test")).
			FromSpec(existingSpec).
			Spec()

		assert.Equal(t, 1024, spec.Width)
		assert.Equal(t, 768, spec.Height)
		assert.Equal(t, "jpeg", spec.Format)
		assert.Equal(t, 90, spec.Quality)
		assert.NotNil(t, spec.Modifiers, "FromSpec should initialise nil Modifiers map")
	})

	t.Run("Spec returns current spec without executing", func(t *testing.T) {
		t.Parallel()

		spec := service.Transform(strings.NewReader("test")).
			Width(640).
			Height(480).
			Format("png").
			Spec()

		assert.Equal(t, 640, spec.Width)
		assert.Equal(t, 480, spec.Height)
		assert.Equal(t, "png", spec.Format)
	})
}

func TestTransformBuilder_Do(t *testing.T) {
	t.Parallel()

	t.Run("successful transform returns result", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("transformed"), "image/webp")
		service := newTestService(t, mock)

		ctx := context.Background()
		result, err := service.Transform(strings.NewReader("input")).
			Width(800).
			Height(600).
			Format("webp").
			Quality(85).
			Cover().
			Do(ctx)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.MIMEType)

		body, readErr := io.ReadAll(result.Body)
		require.NoError(t, readErr)
		assert.Equal(t, []byte("transformed"), body)
	})

	t.Run("nil input returns error", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		service := newTestService(t, mock)

		ctx := context.Background()
		builder := service.Transform(nil)
		result, err := builder.Do(ctx)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no input provided")
	})

	t.Run("transformer error propagated through pipe", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setError(errors.New("transform failed"))
		service := newTestService(t, mock)

		ctx := context.Background()
		result, err := service.Transform(strings.NewReader("input")).
			Width(800).
			Format("jpeg").
			Quality(85).
			Cover().
			Do(ctx)

		if err != nil {
			assert.Contains(t, err.Error(), "transform failed")
			return
		}

		require.NotNil(t, result)
		_, readErr := io.ReadAll(result.Body)
		require.Error(t, readErr)
		assert.Contains(t, readErr.Error(), "transform failed")
	})
}

func TestTransformBuilder_DoToWriter(t *testing.T) {
	t.Parallel()

	t.Run("successful transform writes to buffer", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setTransformResult([]byte("output data"), "image/png")
		service := newTestService(t, mock)

		ctx := context.Background()
		var buffer bytes.Buffer
		err := service.Transform(strings.NewReader("input")).
			Width(640).
			Format("png").
			Quality(90).
			DoToWriter(ctx, &buffer)

		require.NoError(t, err)
		assert.Equal(t, []byte("output data"), buffer.Bytes())
	})

	t.Run("nil input returns error", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		service := newTestService(t, mock)

		ctx := context.Background()
		var buffer bytes.Buffer
		err := service.Transform(nil).DoToWriter(ctx, &buffer)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no input provided")
	})

	t.Run("transformer error returns error", func(t *testing.T) {
		t.Parallel()

		mock := newMockTransformer()
		mock.setError(errors.New("write failed"))
		service := newTestService(t, mock)

		ctx := context.Background()
		var buffer bytes.Buffer
		err := service.Transform(strings.NewReader("input")).
			Width(800).
			Format("jpeg").
			Quality(85).
			Cover().
			DoToWriter(ctx, &buffer)

		require.Error(t, err)
	})
}

func TestTransformBuilder_Chaining(t *testing.T) {
	t.Parallel()

	mock := newMockTransformer()
	service := newTestService(t, mock)

	spec := service.Transform(strings.NewReader("test")).
		Width(800).
		Height(600).
		Format("webp").
		Quality(85).
		Cover().
		WithoutEnlargement().
		Background("#FFFFFF").
		AspectRatio("16:9").
		Provider("mock").
		Blur(2.5).
		Spec()

	assert.Equal(t, 800, spec.Width)
	assert.Equal(t, 600, spec.Height)
	assert.Equal(t, "webp", spec.Format)
	assert.Equal(t, 85, spec.Quality)
	assert.Equal(t, image_dto.FitCover, spec.Fit)
	assert.True(t, spec.WithoutEnlargement)
	assert.Equal(t, "#FFFFFF", spec.Background)
	assert.Equal(t, "16:9", spec.AspectRatio)
	assert.Equal(t, "mock", spec.Provider)
	assert.Equal(t, "2.5", spec.Modifiers["blur"])
}
