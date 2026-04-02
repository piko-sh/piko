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

package capabilities_functions

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
)

type mockImageService struct {
	transformStreamFunction     func(ctx context.Context, input io.Reader, spec image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error)
	generatePlaceholderFunction func(ctx context.Context, input io.Reader, baseSpec image_dto.TransformationSpec) (string, error)
}

func (m *mockImageService) Transform(_ io.Reader) *image_domain.TransformBuilder {
	return nil
}

func (m *mockImageService) TransformStream(ctx context.Context, input io.Reader, spec image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error) {
	if m.transformStreamFunction != nil {
		return m.transformStreamFunction(ctx, input, spec)
	}
	return nil, errors.New("no mock configured")
}

func (m *mockImageService) GenerateResponsiveVariants(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) ([]image_dto.ResponsiveVariant, error) {
	return nil, errors.New("not implemented")
}

func (m *mockImageService) GeneratePlaceholder(ctx context.Context, input io.Reader, baseSpec image_dto.TransformationSpec) (string, error) {
	if m.generatePlaceholderFunction != nil {
		return m.generatePlaceholderFunction(ctx, input, baseSpec)
	}
	return "", errors.New("no mock configured")
}

func (m *mockImageService) GetDimensions(_ context.Context, _ io.Reader) (int, int, error) {
	return 0, 0, errors.New("not implemented")
}

func TestParseIsPlaceholder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params   capabilities_domain.CapabilityParams
		name     string
		expected bool
	}{
		{
			name:     "should return false when no placeholder key",
			params:   capabilities_domain.CapabilityParams{},
			expected: false,
		},
		{
			name:     "should return false for nil params",
			params:   nil,
			expected: false,
		},
		{
			name:     "should return true for true",
			params:   capabilities_domain.CapabilityParams{"placeholder": "true"},
			expected: true,
		},
		{
			name:     "should return true for TRUE (case insensitive)",
			params:   capabilities_domain.CapabilityParams{"placeholder": "TRUE"},
			expected: true,
		},
		{
			name:     "should return true for True (mixed case)",
			params:   capabilities_domain.CapabilityParams{"placeholder": "True"},
			expected: true,
		},
		{
			name:     "should return true for yes",
			params:   capabilities_domain.CapabilityParams{"placeholder": "yes"},
			expected: true,
		},
		{
			name:     "should return true for YES (case insensitive)",
			params:   capabilities_domain.CapabilityParams{"placeholder": "YES"},
			expected: true,
		},
		{
			name:     "should return true for 1",
			params:   capabilities_domain.CapabilityParams{"placeholder": "1"},
			expected: true,
		},
		{
			name:     "should return false for false",
			params:   capabilities_domain.CapabilityParams{"placeholder": "false"},
			expected: false,
		},
		{
			name:     "should return false for no",
			params:   capabilities_domain.CapabilityParams{"placeholder": "no"},
			expected: false,
		},
		{
			name:     "should return false for 0",
			params:   capabilities_domain.CapabilityParams{"placeholder": "0"},
			expected: false,
		},
		{
			name:     "should return false for empty string",
			params:   capabilities_domain.CapabilityParams{"placeholder": ""},
			expected: false,
		},
		{
			name:     "should return false for random string",
			params:   capabilities_domain.CapabilityParams{"placeholder": "maybe"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseIsPlaceholder(tc.params)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParsePlaceholderWidth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params      capabilities_domain.CapabilityParams
		name        string
		expectedVal int
		expectedOk  bool
	}{
		{
			name:        "should return false when key not present",
			params:      capabilities_domain.CapabilityParams{},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for non-numeric value",
			params:      capabilities_domain.CapabilityParams{"placeholder-width": "abc"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for zero width",
			params:      capabilities_domain.CapabilityParams{"placeholder-width": "0"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for negative width",
			params:      capabilities_domain.CapabilityParams{"placeholder-width": "-5"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should parse valid width",
			params:      capabilities_domain.CapabilityParams{"placeholder-width": "30"},
			expectedVal: 30,
			expectedOk:  true,
		},
		{
			name:        "should parse width of 1",
			params:      capabilities_domain.CapabilityParams{"placeholder-width": "1"},
			expectedVal: 1,
			expectedOk:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := parsePlaceholderWidth(tc.params)
			assert.Equal(t, tc.expectedVal, value)
			assert.Equal(t, tc.expectedOk, ok)
		})
	}
}

func TestParsePlaceholderHeight(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params      capabilities_domain.CapabilityParams
		name        string
		expectedVal int
		expectedOk  bool
	}{
		{
			name:        "should return false when key not present",
			params:      capabilities_domain.CapabilityParams{},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for non-numeric value",
			params:      capabilities_domain.CapabilityParams{"placeholder-height": "abc"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for negative height",
			params:      capabilities_domain.CapabilityParams{"placeholder-height": "-1"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should parse zero height (valid for auto-calculate)",
			params:      capabilities_domain.CapabilityParams{"placeholder-height": "0"},
			expectedVal: 0,
			expectedOk:  true,
		},
		{
			name:        "should parse valid height",
			params:      capabilities_domain.CapabilityParams{"placeholder-height": "20"},
			expectedVal: 20,
			expectedOk:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := parsePlaceholderHeight(tc.params)
			assert.Equal(t, tc.expectedVal, value)
			assert.Equal(t, tc.expectedOk, ok)
		})
	}
}

func TestParsePlaceholderQuality(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params      capabilities_domain.CapabilityParams
		name        string
		expectedVal int
		expectedOk  bool
	}{
		{
			name:        "should return false when key not present",
			params:      capabilities_domain.CapabilityParams{},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for non-numeric value",
			params:      capabilities_domain.CapabilityParams{"placeholder-quality": "high"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for quality below minimum",
			params:      capabilities_domain.CapabilityParams{"placeholder-quality": "0"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for quality above maximum",
			params:      capabilities_domain.CapabilityParams{"placeholder-quality": "101"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should parse minimum quality",
			params:      capabilities_domain.CapabilityParams{"placeholder-quality": "1"},
			expectedVal: 1,
			expectedOk:  true,
		},
		{
			name:        "should parse maximum quality",
			params:      capabilities_domain.CapabilityParams{"placeholder-quality": "100"},
			expectedVal: 100,
			expectedOk:  true,
		},
		{
			name:        "should parse middle quality",
			params:      capabilities_domain.CapabilityParams{"placeholder-quality": "50"},
			expectedVal: 50,
			expectedOk:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := parsePlaceholderQuality(tc.params)
			assert.Equal(t, tc.expectedVal, value)
			assert.Equal(t, tc.expectedOk, ok)
		})
	}
}

func TestParsePlaceholderBlur(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params      capabilities_domain.CapabilityParams
		name        string
		expectedVal float64
		expectedOk  bool
	}{
		{
			name:        "should return false when key not present",
			params:      capabilities_domain.CapabilityParams{},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for non-numeric value",
			params:      capabilities_domain.CapabilityParams{"placeholder-blur": "heavy"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should return false for negative blur",
			params:      capabilities_domain.CapabilityParams{"placeholder-blur": "-1.0"},
			expectedVal: 0,
			expectedOk:  false,
		},
		{
			name:        "should parse zero blur (valid)",
			params:      capabilities_domain.CapabilityParams{"placeholder-blur": "0"},
			expectedVal: 0.0,
			expectedOk:  true,
		},
		{
			name:        "should parse float blur value",
			params:      capabilities_domain.CapabilityParams{"placeholder-blur": "5.5"},
			expectedVal: 5.5,
			expectedOk:  true,
		},
		{
			name:        "should parse integer blur value",
			params:      capabilities_domain.CapabilityParams{"placeholder-blur": "10"},
			expectedVal: 10.0,
			expectedOk:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, ok := parsePlaceholderBlur(tc.params)
			assert.InDelta(t, tc.expectedVal, value, 0.001)
			assert.Equal(t, tc.expectedOk, ok)
		})
	}
}

func TestBuildPlaceholderSpec(t *testing.T) {
	t.Parallel()

	t.Run("should use defaults when no params provided", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec := buildPlaceholderSpec(params)
		require.NotNil(t, spec)
		assert.True(t, spec.Enabled)
		assert.Equal(t, defaultPlaceholderWidth, spec.Width)
		assert.Equal(t, defaultPlaceholderHeight, spec.Height)
		assert.Equal(t, defaultPlaceholderQuality, spec.Quality)
		assert.InDelta(t, defaultPlaceholderBlur, spec.BlurSigma, 0.001)
	})

	t.Run("should override width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"placeholder-width": "30"}
		spec := buildPlaceholderSpec(params)
		assert.Equal(t, 30, spec.Width)
	})

	t.Run("should override height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"placeholder-height": "15"}
		spec := buildPlaceholderSpec(params)
		assert.Equal(t, 15, spec.Height)
	})

	t.Run("should override quality", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"placeholder-quality": "25"}
		spec := buildPlaceholderSpec(params)
		assert.Equal(t, 25, spec.Quality)
	})

	t.Run("should override blur", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"placeholder-blur": "3.5"}
		spec := buildPlaceholderSpec(params)
		assert.InDelta(t, 3.5, spec.BlurSigma, 0.001)
	})

	t.Run("should override all values", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"placeholder-width":   "40",
			"placeholder-height":  "25",
			"placeholder-quality": "15",
			"placeholder-blur":    "7.0",
		}
		spec := buildPlaceholderSpec(params)
		assert.Equal(t, 40, spec.Width)
		assert.Equal(t, 25, spec.Height)
		assert.Equal(t, 15, spec.Quality)
		assert.InDelta(t, 7.0, spec.BlurSigma, 0.001)
	})

	t.Run("should ignore invalid overrides and keep defaults", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"placeholder-width":   "invalid",
			"placeholder-quality": "-5",
		}
		spec := buildPlaceholderSpec(params)
		assert.Equal(t, defaultPlaceholderWidth, spec.Width)
		assert.Equal(t, defaultPlaceholderQuality, spec.Quality)
	})
}

func TestParseTransformParams(t *testing.T) {
	t.Parallel()

	t.Run("should parse provider", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"provider": "vips"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "vips", spec.Provider)
	})

	t.Run("should parse width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "800"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 800, spec.Width)
	})

	t.Run("should ignore invalid width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "abc"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 0, spec.Width)
	})

	t.Run("should ignore negative width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "-10"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 0, spec.Width)
	})

	t.Run("should parse height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"height": "600"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 600, spec.Height)
	})

	t.Run("should ignore invalid height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"height": "abc"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 0, spec.Height)
	})

	t.Run("should parse quality", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"quality": "85"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 85, spec.Quality)
	})

	t.Run("should ignore quality below minimum", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"quality": "0"}
		spec := image_dto.DefaultTransformationSpec()
		originalQuality := spec.Quality
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, originalQuality, spec.Quality)
	})

	t.Run("should ignore quality above maximum", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"quality": "200"}
		spec := image_dto.DefaultTransformationSpec()
		originalQuality := spec.Quality
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, originalQuality, spec.Quality)
	})

	t.Run("should parse format", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"format": "jpeg"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "jpeg", spec.Format)
	})

	t.Run("should not set format when original", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"format": "original"}
		spec := image_dto.DefaultTransformationSpec()
		originalFormat := spec.Format
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, originalFormat, spec.Format)
	})

	t.Run("should parse fit", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"fit": "cover"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, image_dto.FitCover, spec.Fit)
	})

	t.Run("should treat crop as unknown modifier", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"crop": "true"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "true", spec.Modifiers["crop"])
	})

	t.Run("should parse aspectratio", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"aspectratio": "16:9"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "16:9", spec.AspectRatio)
	})

	t.Run("should parse aspect_ratio alias", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"aspect_ratio": "4:3"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "4:3", spec.AspectRatio)
	})

	t.Run("should parse withoutenlargement true", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"withoutenlargement": "true"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.True(t, spec.WithoutEnlargement)
	})

	t.Run("should parse without_enlargement alias", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"without_enlargement": "true"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.True(t, spec.WithoutEnlargement)
	})

	t.Run("should parse background", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"background": "#FFFFFF"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "#FFFFFF", spec.Background)
	})

	t.Run("should parse bg alias for background", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"bg": "#000000"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "#000000", spec.Background)
	})

	t.Run("should store unknown params as modifiers", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"greyscale": "true",
			"blur":      "5.0",
			"sharpen":   "2.0",
		}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "true", spec.Modifiers["greyscale"])
		assert.Equal(t, "5.0", spec.Modifiers["blur"])
		assert.Equal(t, "2.0", spec.Modifiers["sharpen"])
	})

	t.Run("should skip placeholder-related params from modifiers", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"placeholder":         "true",
			"placeholder-width":   "20",
			"placeholder-height":  "15",
			"placeholder-quality": "10",
			"placeholder-blur":    "5.0",
		}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Empty(t, spec.Modifiers)
	})

	t.Run("should handle case insensitive keys", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"Width": "500", "Height": "300"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 500, spec.Width)
		assert.Equal(t, 300, spec.Height)
	})

	t.Run("should parse zero width and height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "0", "height": "0"}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, 0, spec.Width)
		assert.Equal(t, 0, spec.Height)
	})

	t.Run("should handle all params together", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"provider":           "vips",
			"width":              "800",
			"height":             "600",
			"quality":            "85",
			"format":             "webp",
			"fit":                "cover",
			"aspectratio":        "16:9",
			"withoutenlargement": "true",
			"background":         "#FF0000",
			"greyscale":          "true",
		}
		spec := image_dto.DefaultTransformationSpec()
		spec.Modifiers = map[string]string{}
		parseTransformParams(params, &spec)
		assert.Equal(t, "vips", spec.Provider)
		assert.Equal(t, 800, spec.Width)
		assert.Equal(t, 600, spec.Height)
		assert.Equal(t, 85, spec.Quality)
		assert.Equal(t, "webp", spec.Format)
		assert.Equal(t, image_dto.FitCover, spec.Fit)
		assert.Equal(t, "16:9", spec.AspectRatio)
		assert.True(t, spec.WithoutEnlargement)
		assert.Equal(t, "#FF0000", spec.Background)
		assert.Equal(t, "true", spec.Modifiers["greyscale"])
	})
}

func TestBuildTransformSpec(t *testing.T) {
	t.Parallel()

	t.Run("should return default spec when no params and not placeholder", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec := buildTransformSpec(params, false)
		assert.Nil(t, spec.Placeholder)
		assert.NotNil(t, spec.Modifiers)
	})

	t.Run("should include placeholder spec when isPlaceholder is true", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec := buildTransformSpec(params, true)
		require.NotNil(t, spec.Placeholder)
		assert.True(t, spec.Placeholder.Enabled)
		assert.Equal(t, defaultPlaceholderWidth, spec.Placeholder.Width)
	})

	t.Run("should parse transform params", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"width":  "640",
			"height": "480",
		}
		spec := buildTransformSpec(params, false)
		assert.Equal(t, 640, spec.Width)
		assert.Equal(t, 480, spec.Height)
	})

	t.Run("should include placeholder with overrides", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"placeholder-width":   "40",
			"placeholder-quality": "20",
		}
		spec := buildTransformSpec(params, true)
		require.NotNil(t, spec.Placeholder)
		assert.Equal(t, 40, spec.Placeholder.Width)
		assert.Equal(t, 20, spec.Placeholder.Quality)
	})
}

func TestImageTransform(t *testing.T) {
	t.Parallel()

	t.Run("should perform standard image transform", func(t *testing.T) {
		t.Parallel()
		service := &mockImageService{
			transformStreamFunction: func(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error) {
				return &image_dto.TransformedImageResult{
					Body:     io.NopCloser(strings.NewReader("transformed")),
					MIMEType: "image/webp",
				}, nil
			},
		}

		capabilityFunction := ImageTransform(service)
		params := capabilities_domain.CapabilityParams{
			"width":  "800",
			"height": "600",
		}
		result, err := capabilityFunction(context.Background(), strings.NewReader("input image"), params)
		require.NoError(t, err)
		require.NotNil(t, result)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, "transformed", string(output))
	})

	t.Run("should perform placeholder transform", func(t *testing.T) {
		t.Parallel()
		service := &mockImageService{
			generatePlaceholderFunction: func(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) (string, error) {
				return "data:image/jpeg;base64,abc123", nil
			},
		}

		capabilityFunction := ImageTransform(service)
		params := capabilities_domain.CapabilityParams{
			"placeholder": "true",
		}
		result, err := capabilityFunction(context.Background(), strings.NewReader("input image"), params)
		require.NoError(t, err)
		require.NotNil(t, result)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, "data:image/jpeg;base64,abc123", string(output))
	})

	t.Run("should return error when transform stream fails", func(t *testing.T) {
		t.Parallel()
		service := &mockImageService{
			transformStreamFunction: func(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error) {
				return nil, errors.New("transform failed")
			},
		}

		capabilityFunction := ImageTransform(service)
		params := capabilities_domain.CapabilityParams{}
		_, err := capabilityFunction(context.Background(), strings.NewReader("input"), params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "image transform stream")
	})

	t.Run("should return error when placeholder generation fails", func(t *testing.T) {
		t.Parallel()
		service := &mockImageService{
			generatePlaceholderFunction: func(_ context.Context, _ io.Reader, _ image_dto.TransformationSpec) (string, error) {
				return "", errors.New("placeholder failed")
			},
		}

		capabilityFunction := ImageTransform(service)
		params := capabilities_domain.CapabilityParams{
			"placeholder": "true",
		}
		_, err := capabilityFunction(context.Background(), strings.NewReader("input"), params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "placeholder")
	})

	t.Run("should pass transform params to spec", func(t *testing.T) {
		t.Parallel()
		var capturedSpec image_dto.TransformationSpec
		service := &mockImageService{
			transformStreamFunction: func(_ context.Context, _ io.Reader, spec image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error) {
				capturedSpec = spec
				return &image_dto.TransformedImageResult{
					Body: io.NopCloser(strings.NewReader("ok")),
				}, nil
			},
		}

		capabilityFunction := ImageTransform(service)
		params := capabilities_domain.CapabilityParams{
			"width":   "640",
			"height":  "480",
			"quality": "85",
			"format":  "jpeg",
			"fit":     "cover",
		}
		result, err := capabilityFunction(context.Background(), strings.NewReader("input"), params)
		require.NoError(t, err)
		_, _ = io.ReadAll(result)

		assert.Equal(t, 640, capturedSpec.Width)
		assert.Equal(t, 480, capturedSpec.Height)
		assert.Equal(t, 85, capturedSpec.Quality)
		assert.Equal(t, image_dto.FitCover, capturedSpec.Fit)
	})

	t.Run("should return error for invalid transformation spec", func(t *testing.T) {
		t.Parallel()
		service := &mockImageService{}

		capabilityFunction := ImageTransform(service)
		params := capabilities_domain.CapabilityParams{
			"format": "bmp",
		}
		_, err := capabilityFunction(context.Background(), strings.NewReader("input"), params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transformation spec")
	})
}
