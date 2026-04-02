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

package image_provider_imaging

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/media"
)

func newTestImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	return img
}

func newGradientImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{
				R: uint8((x * 255) / max(w-1, 1)),
				G: uint8((y * 255) / max(h-1, 1)),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

func TestParseAspectRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "16:9", input: "16:9", want: 16.0 / 9.0, wantErr: false},
		{name: "4:3", input: "4:3", want: 4.0 / 3.0, wantErr: false},
		{name: "1:1", input: "1:1", want: 1.0, wantErr: false},
		{name: "missing colon", input: "16", want: 0, wantErr: true},
		{name: "too many parts", input: "16:9:4", want: 0, wantErr: true},
		{name: "non-positive width", input: "0:9", want: 0, wantErr: true},
		{name: "non-positive height", input: "16:0", want: 0, wantErr: true},
		{name: "invalid width", input: "abc:9", want: 0, wantErr: true},
		{name: "invalid height", input: "16:xyz", want: 0, wantErr: true},
		{name: "negative width", input: "-1:9", want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAspectRatio(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}

func TestCalculateTargetDimensions(t *testing.T) {
	t.Parallel()

	t.Run("empty ratio returns original", func(t *testing.T) {
		t.Parallel()

		w, h := calculateTargetDimensions(1920, 1080, "")
		assert.Equal(t, 1920, w)
		assert.Equal(t, 1080, h)
	})

	t.Run("width only with ratio calculates height", func(t *testing.T) {
		t.Parallel()

		w, h := calculateTargetDimensions(1920, 0, "16:9")
		assert.Equal(t, 1920, w)
		assert.Equal(t, 1080, h)
	})

	t.Run("height only with ratio calculates width", func(t *testing.T) {
		t.Parallel()

		w, h := calculateTargetDimensions(0, 1080, "16:9")
		assert.Equal(t, 1920, w)
		assert.Equal(t, 1080, h)
	})

	t.Run("both set returns original", func(t *testing.T) {
		t.Parallel()

		w, h := calculateTargetDimensions(800, 600, "16:9")
		assert.Equal(t, 800, w)
		assert.Equal(t, 600, h)
	})

	t.Run("invalid ratio returns original", func(t *testing.T) {
		t.Parallel()

		w, h := calculateTargetDimensions(800, 600, "invalid")
		assert.Equal(t, 800, w)
		assert.Equal(t, 600, h)
	})
}

func TestApplyWithoutEnlargement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  int
		current int
		want    int
	}{
		{name: "target smaller than current", target: 500, current: 1000, want: 500},
		{name: "target larger than current", target: 2000, current: 1000, want: 1000},
		{name: "target equals current", target: 1000, current: 1000, want: 1000},
		{name: "target is zero", target: 0, current: 1000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := applyWithoutEnlargement(tt.target, tt.current)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyGreyscale(t *testing.T) {
	t.Parallel()

	t.Run("true applies greyscale", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyGreyscale(&img, map[string]string{"greyscale": "true"})
		assert.NotEqual(t, original, img)
	})

	t.Run("empty string applies greyscale", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyGreyscale(&img, map[string]string{"greyscale": ""})
		assert.NotEqual(t, original, img)
	})

	t.Run("false does not apply", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyGreyscale(&img, map[string]string{"greyscale": "false"})
		assert.Equal(t, original, img)
	})

	t.Run("missing key does not apply", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyGreyscale(&img, map[string]string{})
		assert.Equal(t, original, img)
	})
}

func TestApplyBlur(t *testing.T) {
	t.Parallel()

	t.Run("valid sigma applies blur", func(t *testing.T) {
		t.Parallel()

		img := newGradientImage(20, 20)
		original := img
		applyBlur(&img, map[string]string{"blur": "2.5"})
		assert.NotEqual(t, original, img)
	})

	t.Run("zero sigma no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyBlur(&img, map[string]string{"blur": "0"})
		assert.Equal(t, original, img)
	})

	t.Run("invalid string no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyBlur(&img, map[string]string{"blur": "abc"})
		assert.Equal(t, original, img)
	})

	t.Run("missing key no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyBlur(&img, map[string]string{})
		assert.Equal(t, original, img)
	})
}

func TestApplySharpen(t *testing.T) {
	t.Parallel()

	t.Run("valid sigma applies sharpen", func(t *testing.T) {
		t.Parallel()

		img := newGradientImage(20, 20)
		original := img
		applySharpen(&img, map[string]string{"sharpen": "1.5"})
		assert.NotEqual(t, original, img)
	})

	t.Run("missing key no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applySharpen(&img, map[string]string{})
		assert.Equal(t, original, img)
	})

	t.Run("invalid string no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applySharpen(&img, map[string]string{"sharpen": "abc"})
		assert.Equal(t, original, img)
	})
}

func TestApplyRotation(t *testing.T) {
	t.Parallel()

	t.Run("90 degrees", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(20, 10)
		applyRotation(&img, map[string]string{"rotate": "90"})
		bounds := img.Bounds()
		assert.Equal(t, 10, bounds.Dx())
		assert.Equal(t, 20, bounds.Dy())
	})

	t.Run("180 degrees", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(20, 10)
		applyRotation(&img, map[string]string{"rotate": "180"})
		bounds := img.Bounds()
		assert.Equal(t, 20, bounds.Dx())
		assert.Equal(t, 10, bounds.Dy())
	})

	t.Run("270 degrees", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(20, 10)
		applyRotation(&img, map[string]string{"rotate": "270"})
		bounds := img.Bounds()
		assert.Equal(t, 10, bounds.Dx())
		assert.Equal(t, 20, bounds.Dy())
	})

	t.Run("unsupported angle no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(20, 10)
		original := img
		applyRotation(&img, map[string]string{"rotate": "45"})
		assert.Equal(t, original, img)
	})

	t.Run("invalid string no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(20, 10)
		original := img
		applyRotation(&img, map[string]string{"rotate": "abc"})
		assert.Equal(t, original, img)
	})

	t.Run("missing key no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(20, 10)
		original := img
		applyRotation(&img, map[string]string{})
		assert.Equal(t, original, img)
	})
}

func TestApplyFlip(t *testing.T) {
	t.Parallel()

	t.Run("horizontal", func(t *testing.T) {
		t.Parallel()

		img := newGradientImage(10, 10)
		original := img
		applyFlip(&img, map[string]string{"flip": "horizontal"})
		assert.NotEqual(t, original, img)
	})

	t.Run("h shorthand", func(t *testing.T) {
		t.Parallel()

		img := newGradientImage(10, 10)
		original := img
		applyFlip(&img, map[string]string{"flip": "h"})
		assert.NotEqual(t, original, img)
	})

	t.Run("vertical", func(t *testing.T) {
		t.Parallel()

		img := newGradientImage(10, 10)
		original := img
		applyFlip(&img, map[string]string{"flip": "vertical"})
		assert.NotEqual(t, original, img)
	})

	t.Run("v shorthand", func(t *testing.T) {
		t.Parallel()

		img := newGradientImage(10, 10)
		original := img
		applyFlip(&img, map[string]string{"flip": "v"})
		assert.NotEqual(t, original, img)
	})

	t.Run("unknown value no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyFlip(&img, map[string]string{"flip": "diagonal"})
		assert.Equal(t, original, img)
	})

	t.Run("missing key no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyFlip(&img, map[string]string{})
		assert.Equal(t, original, img)
	})
}

func TestApplyColourAdjustments(t *testing.T) {
	t.Parallel()

	t.Run("brightness adjusts image", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyColourAdjustments(&img, map[string]string{"brightness": "20"})
		assert.NotEqual(t, original, img)
	})

	t.Run("contrast adjusts image", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyColourAdjustments(&img, map[string]string{"contrast": "15"})
		assert.NotEqual(t, original, img)
	})

	t.Run("saturation adjusts image", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyColourAdjustments(&img, map[string]string{"saturation": "30"})
		assert.NotEqual(t, original, img)
	})

	t.Run("invalid values skip", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyColourAdjustments(&img, map[string]string{
			"brightness": "abc",
			"contrast":   "xyz",
			"saturation": "definition",
		})
		assert.Equal(t, original, img)
	})

	t.Run("missing keys no-op", func(t *testing.T) {
		t.Parallel()

		img := newTestImage(10, 10)
		original := img
		applyColourAdjustments(&img, map[string]string{})
		assert.Equal(t, original, img)
	})
}

func TestGetSupportedFormats(t *testing.T) {
	t.Parallel()

	p := &Provider{}
	formats := p.GetSupportedFormats()
	assert.Equal(t, []string{"jpeg", "jpg", "png", "webp", "gif"}, formats)
}

func TestGetSupportedModifiers(t *testing.T) {
	t.Parallel()

	p := &Provider{}
	modifiers := p.GetSupportedModifiers()
	assert.Len(t, modifiers, 8)
	assert.Contains(t, modifiers, "greyscale")
	assert.Contains(t, modifiers, "blur")
	assert.Contains(t, modifiers, "sharpen")
	assert.Contains(t, modifiers, "rotate")
	assert.Contains(t, modifiers, "flip")
	assert.Contains(t, modifiers, "brightness")
	assert.Contains(t, modifiers, "contrast")
	assert.Contains(t, modifiers, "saturation")
}

func TestNewProvider(t *testing.T) {
	t.Parallel()

	p := NewProvider(Config{})
	require.NotNil(t, p)
	assert.NotNil(t, p.semaphore)
	assert.NotZero(t, p.config.MaxFileSizeBytes)
}

func TestNewProvider_CustomConfig(t *testing.T) {
	t.Parallel()

	p := NewProvider(Config{
		ImageServiceConfig: media.ImageServiceConfig{
			MaxFileSizeBytes: 1024,
		},
	})
	require.NotNil(t, p)
	assert.Equal(t, int64(1024), p.config.MaxFileSizeBytes)
}
