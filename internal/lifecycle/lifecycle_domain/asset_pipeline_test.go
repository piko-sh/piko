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

package lifecycle_domain

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type upsertCall struct {
	artefactID string
	sourcePath string
	profiles   []registry_dto.NamedProfile
}

func newPipelineRegistryMock(upsertError error) (*registry_domain.MockRegistryService, *[]upsertCall) {
	var calls []upsertCall
	mock := &registry_domain.MockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID, sourcePath string,
			_ io.Reader,
			_ string,
			profiles []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			calls = append(calls, upsertCall{
				artefactID: artefactID,
				sourcePath: sourcePath,
				profiles:   profiles,
			})
			if upsertError != nil {
				return nil, upsertError
			}
			return &registry_dto.ArtefactMeta{}, nil
		},
	}
	return mock, &calls
}
func Test_NewAssetPipelineOrchestrator(t *testing.T) {
	t.Parallel()

	t.Run("creates orchestrator with dependencies", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)
		require.NotNil(t, orchestrator)
	})
}
func Test_AssetPipelineOrchestrator_ProcessBuildResult(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil result", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		err := orchestrator.ProcessBuildResult(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("returns nil for empty asset manifest", func(t *testing.T) {
		t.Parallel()

		orchestrator := NewAssetPipelineOrchestrator(nil, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)
	})

	t.Run("processes image assets with profiles", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "assets/hero.jpg",
					AssetType:  "img",
					TransformationParams: map[string][]string{
						"sizes":  {"400px"},
						"format": {"webp"},
					},
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)
		require.Len(t, *calls, 1)
		assert.Equal(t, "assets/hero.jpg", (*calls)[0].artefactID)
		assert.NotEmpty(t, (*calls)[0].profiles)
	})

	t.Run("processes CSS assets with default profiles", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "styles/main.css",
					AssetType:  "css",
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)

		require.Len(t, *calls, 1)
		assert.Equal(t, "styles/main.css", (*calls)[0].artefactID)
	})

	t.Run("processes JavaScript assets", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "scripts/app.js",
					AssetType:  "js",
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)

		require.Len(t, *calls, 1)
		assert.Equal(t, "scripts/app.js", (*calls)[0].artefactID)
	})

	t.Run("processes SVG assets", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "icons/logo.svg",
					AssetType:  "svg",
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)

		require.Len(t, *calls, 1)
		assert.Equal(t, "icons/logo.svg", (*calls)[0].artefactID)
	})

	t.Run("processes multiple assets", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "assets/img1.jpg",
					AssetType:  "img",
					TransformationParams: map[string][]string{
						"sizes":  {"400px"},
						"format": {"webp"},
					},
				},
				{
					SourcePath: "styles/main.css",
					AssetType:  "css",
				},
				{
					SourcePath: "scripts/app.js",
					AssetType:  "js",
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)
		assert.Len(t, *calls, 3)
	})

	t.Run("skips assets with no profiles", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(nil)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "readme.md",
					AssetType:  "unknown",
				},
			},
		}

		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)
		assert.Empty(t, *calls)
	})

	t.Run("continues processing on upsert error", func(t *testing.T) {
		t.Parallel()

		mockRegistry, calls := newPipelineRegistryMock(assert.AnError)

		orchestrator := NewAssetPipelineOrchestrator(mockRegistry, nil)

		result := &annotator_dto.ProjectAnnotationResult{
			FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
				{
					SourcePath: "styles/main.css",
					AssetType:  "css",
				},
				{
					SourcePath: "scripts/app.js",
					AssetType:  "js",
				},
			},
		}
		err := orchestrator.ProcessBuildResult(context.Background(), result)
		assert.NoError(t, err)
		assert.Len(t, *calls, 2)
	})
}
func TestParseDensity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		density     string
		expected    float64
		expectError bool
	}{
		{
			name:     "1x density",
			density:  "1x",
			expected: 1.0,
		},
		{
			name:     "2x density",
			density:  "2x",
			expected: 2.0,
		},
		{
			name:     "3x density",
			density:  "3x",
			expected: 3.0,
		},
		{
			name:     "prefix x notation",
			density:  "x2",
			expected: 2.0,
		},
		{
			name:     "no x notation",
			density:  "2",
			expected: 2.0,
		},
		{
			name:     "decimal density",
			density:  "1.5x",
			expected: 1.5,
		},
		{
			name:     "uppercase X",
			density:  "2X",
			expected: 2.0,
		},
		{
			name:     "with whitespace",
			density:  " 2x ",
			expected: 2.0,
		},
		{
			name:        "invalid density",
			density:     "abc",
			expectError: true,
		},
		{
			name:        "empty string",
			density:     "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseDensity(tc.density)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expected, result, 0.001)
			}
		})
	}
}
func TestExtractSizeValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		size     string
		expected string
	}{
		{
			name:     "simple pixel value",
			size:     "400px",
			expected: "400px",
		},
		{
			name:     "simple vw value",
			size:     "100vw",
			expected: "100vw",
		},
		{
			name:     "breakpoint prefixed value",
			size:     "sm:50vw",
			expected: "50vw",
		},
		{
			name:     "breakpoint prefixed px",
			size:     "md:400px",
			expected: "400px",
		},
		{
			name:     "with whitespace",
			size:     "lg: 800px ",
			expected: "800px",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractSizeValue(tc.size)
			assert.Equal(t, tc.expected, result)
		})
	}
}
func TestParseSizesToWidths(t *testing.T) {
	t.Parallel()

	t.Run("pixel sizes", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"400px", "800px", "1200px"}
		widths := parseSizesToWidths(sizes)

		assert.Contains(t, widths, 400)
		assert.Contains(t, widths, 800)
		assert.Contains(t, widths, 1200)
	})

	t.Run("viewport width sizes", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"100vw"}
		widths := parseSizesToWidths(sizes)

		assert.Contains(t, widths, 320)
		assert.Contains(t, widths, 640)
		assert.Contains(t, widths, 768)
		assert.Contains(t, widths, 1024)
		assert.Contains(t, widths, 1280)
		assert.Contains(t, widths, 1536)
		assert.Contains(t, widths, 1920)
	})

	t.Run("50vw viewport width", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"50vw"}
		widths := parseSizesToWidths(sizes)

		assert.Contains(t, widths, 160)
		assert.Contains(t, widths, 320)
		assert.Contains(t, widths, 384)
		assert.Contains(t, widths, 512)
	})

	t.Run("mixed px and vw", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"400px", "50vw"}
		widths := parseSizesToWidths(sizes)
		assert.Contains(t, widths, 400)

		assert.Contains(t, widths, 160)
	})

	t.Run("breakpoint prefixed sizes", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"sm:50vw", "md:400px"}
		widths := parseSizesToWidths(sizes)
		assert.Contains(t, widths, 400)
		assert.Contains(t, widths, 160)
	})

	t.Run("empty sizes returns empty map", func(t *testing.T) {
		t.Parallel()

		widths := parseSizesToWidths(nil)
		assert.Empty(t, widths)

		widths2 := parseSizesToWidths([]string{})
		assert.Empty(t, widths2)
	})

	t.Run("invalid px value is ignored", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"abcpx", "400px"}
		widths := parseSizesToWidths(sizes)

		assert.Contains(t, widths, 400)
		assert.Len(t, widths, 1)
	})

	t.Run("zero or negative values are ignored", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"0px", "-100px", "400px"}
		widths := parseSizesToWidths(sizes)

		assert.Contains(t, widths, 400)
		assert.NotContains(t, widths, 0)
		assert.NotContains(t, widths, -100)
	})
}
func TestApplyDensitiesToWidths(t *testing.T) {
	t.Parallel()

	t.Run("applies density multipliers", func(t *testing.T) {
		t.Parallel()

		baseWidths := map[int]struct{}{400: {}, 800: {}}
		densities := []string{"1x", "2x"}

		result := applyDensitiesToWidths(baseWidths, densities)
		assert.Contains(t, result, 400)
		assert.Contains(t, result, 800)

		assert.Contains(t, result, 800)
		assert.Contains(t, result, 1600)
	})

	t.Run("ignores 1x density multiplier", func(t *testing.T) {
		t.Parallel()

		baseWidths := map[int]struct{}{400: {}}
		densities := []string{"1x"}

		result := applyDensitiesToWidths(baseWidths, densities)
		assert.Contains(t, result, 400)
		assert.Len(t, result, 1)
	})

	t.Run("handles decimal densities", func(t *testing.T) {
		t.Parallel()

		baseWidths := map[int]struct{}{100: {}}
		densities := []string{"1.5x"}

		result := applyDensitiesToWidths(baseWidths, densities)

		assert.Contains(t, result, 100)
		assert.Contains(t, result, 150)
	})

	t.Run("empty densities preserves base widths", func(t *testing.T) {
		t.Parallel()

		baseWidths := map[int]struct{}{400: {}, 800: {}}
		densities := []string{}

		result := applyDensitiesToWidths(baseWidths, densities)

		assert.Contains(t, result, 400)
		assert.Contains(t, result, 800)
		assert.Len(t, result, 2)
	})
}
func TestWidthSetToSortedSlice(t *testing.T) {
	t.Parallel()

	t.Run("returns sorted slice", func(t *testing.T) {
		t.Parallel()

		widthSet := map[int]struct{}{800: {}, 320: {}, 1200: {}, 640: {}}
		result := widthSetToSortedSlice(widthSet)

		require.Len(t, result, 4)
		assert.Equal(t, []int{320, 640, 800, 1200}, result)
	})

	t.Run("empty set returns nil", func(t *testing.T) {
		t.Parallel()

		result := widthSetToSortedSlice(map[int]struct{}{})
		assert.Nil(t, result)
	})
}

func Test_AssetPipelineOrchestrator_generateImageProfiles(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	t.Run("generates profiles for sizes and formats", func(t *testing.T) {
		t.Parallel()

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath: "hero.jpg",
			TransformationParams: map[string][]string{
				"sizes":   {"400px", "800px"},
				"format":  {"webp", "jpeg"},
				"quality": {"80"},
			},
		}

		profiles := orchestrator.generateImageProfiles(asset)
		assert.Len(t, profiles, 4)
		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.Contains(t, profileNames, "image_w400_webp")
		assert.Contains(t, profileNames, "image_w400_jpeg")
		assert.Contains(t, profileNames, "image_w800_webp")
		assert.Contains(t, profileNames, "image_w800_jpeg")
	})

	t.Run("includes placeholder profile when enabled", func(t *testing.T) {
		t.Parallel()

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath: "hero.jpg",
			TransformationParams: map[string][]string{
				"sizes":       {"400px"},
				"format":      {"webp"},
				"placeholder": {"true"},
			},
		}

		profiles := orchestrator.generateImageProfiles(asset)
		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.Contains(t, profileNames, "placeholder")
	})

	t.Run("no placeholder when not enabled", func(t *testing.T) {
		t.Parallel()

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath: "hero.jpg",
			TransformationParams: map[string][]string{
				"sizes":       {"400px"},
				"format":      {"webp"},
				"placeholder": {"false"},
			},
		}

		profiles := orchestrator.generateImageProfiles(asset)

		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.NotContains(t, profileNames, "placeholder")
	})

	t.Run("returns empty for missing sizes", func(t *testing.T) {
		t.Parallel()

		asset := &annotator_dto.FinalAssetDependency{
			SourcePath: "hero.jpg",
			TransformationParams: map[string][]string{
				"format": {"webp"},
			},
		}

		profiles := orchestrator.generateImageProfiles(asset)
		assert.Empty(t, profiles)
	})
}
func Test_AssetPipelineOrchestrator_parseQuality(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	testCases := []struct {
		name     string
		values   []string
		expected int
	}{
		{
			name:     "valid quality",
			values:   []string{"80"},
			expected: 80,
		},
		{
			name:     "quality 1 is valid",
			values:   []string{"1"},
			expected: 1,
		},
		{
			name:     "quality 100 is valid",
			values:   []string{"100"},
			expected: 100,
		},
		{
			name:     "quality above 100 uses default",
			values:   []string{"150"},
			expected: 80,
		},
		{
			name:     "quality below 1 uses default",
			values:   []string{"0"},
			expected: 80,
		},
		{
			name:     "invalid quality uses default",
			values:   []string{"abc"},
			expected: 80,
		},
		{
			name:     "empty values uses default",
			values:   []string{},
			expected: 80,
		},
		{
			name:     "nil values uses default",
			values:   nil,
			expected: 80,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := orchestrator.parseQuality(tc.values)
			assert.Equal(t, tc.expected, result)
		})
	}
}
func Test_AssetPipelineOrchestrator_parseFormats(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	t.Run("uses provided formats", func(t *testing.T) {
		t.Parallel()

		formats := orchestrator.parseFormats([]string{"webp", "avif"}, "image.jpg")
		assert.Equal(t, []string{"webp", "avif"}, formats)
	})

	t.Run("defaults to modern formats plus original", func(t *testing.T) {
		t.Parallel()

		formats := orchestrator.parseFormats(nil, "image.jpg")
		assert.Equal(t, []string{"webp", "avif", "jpg"}, formats)
	})

	t.Run("handles png source", func(t *testing.T) {
		t.Parallel()

		formats := orchestrator.parseFormats(nil, "logo.png")
		assert.Equal(t, []string{"webp", "avif", "png"}, formats)
	})
}
func Test_AssetPipelineOrchestrator_shouldGeneratePlaceholder(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	testCases := []struct {
		params   map[string][]string
		name     string
		expected bool
	}{
		{
			name:     "true enables placeholder",
			params:   map[string][]string{"placeholder": {"true"}},
			expected: true,
		},
		{
			name:     "yes enables placeholder",
			params:   map[string][]string{"placeholder": {"yes"}},
			expected: true,
		},
		{
			name:     "1 enables placeholder",
			params:   map[string][]string{"placeholder": {"1"}},
			expected: true,
		},
		{
			name:     "enabled enables placeholder",
			params:   map[string][]string{"placeholder": {"enabled"}},
			expected: true,
		},
		{
			name:     "false disables placeholder",
			params:   map[string][]string{"placeholder": {"false"}},
			expected: false,
		},
		{
			name:     "no disables placeholder",
			params:   map[string][]string{"placeholder": {"no"}},
			expected: false,
		},
		{
			name:     "missing key disables placeholder",
			params:   map[string][]string{},
			expected: false,
		},
		{
			name:     "empty value disables placeholder",
			params:   map[string][]string{"placeholder": {}},
			expected: false,
		},
		{
			name:     "case insensitive TRUE",
			params:   map[string][]string{"placeholder": {"TRUE"}},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := orchestrator.shouldGeneratePlaceholder(tc.params)
			assert.Equal(t, tc.expected, result)
		})
	}
}
func Test_AssetPipelineOrchestrator_extractStandardParams(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	t.Run("extracts standard params", func(t *testing.T) {
		t.Parallel()

		params := map[string][]string{
			"fit":                {"cover"},
			"aspectratio":        {"16:9"},
			"withoutenlargement": {"true"},
			"background":         {"#FFFFFF"},
			"provider":           {"vips"},
			"height":             {"600"},
			"crop":               {"center"},

			"custom":      {"value"},
			"placeholder": {"true"},
		}

		result := orchestrator.extractStandardParams(params)

		assert.Equal(t, "cover", result["fit"])
		assert.Equal(t, "16:9", result["aspectratio"])
		assert.Equal(t, "true", result["withoutenlargement"])
		assert.Equal(t, "#FFFFFF", result["background"])
		assert.Equal(t, "vips", result["provider"])
		assert.Equal(t, "600", result["height"])
		assert.Equal(t, "center", result["crop"])
		_, hasCustom := result["custom"]
		assert.False(t, hasCustom)
	})

	t.Run("handles empty params", func(t *testing.T) {
		t.Parallel()

		result := orchestrator.extractStandardParams(map[string][]string{})
		assert.Empty(t, result)
	})
}
func Test_AssetPipelineOrchestrator_getPassthroughModifiers(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	t.Run("extracts only modifier params", func(t *testing.T) {
		t.Parallel()

		params := map[string][]string{

			"sizes":       {"100vw"},
			"format":      {"webp"},
			"quality":     {"80"},
			"fit":         {"cover"},
			"placeholder": {"true"},

			"greyscale":  {"true"},
			"blur":       {"5.0"},
			"sharpen":    {"2.0"},
			"rotate":     {"90"},
			"flip":       {"horizontal"},
			"brightness": {"10"},
			"contrast":   {"20"},
			"saturation": {"-10"},
			"hue":        {"180"},
			"tint":       {"#FF0000"},
			"gravity":    {"attention"},
			"radius":     {"10"},
		}

		result := orchestrator.getPassthroughModifiers(params)
		assert.Equal(t, "true", result["greyscale"])
		assert.Equal(t, "5.0", result["blur"])
		assert.Equal(t, "2.0", result["sharpen"])
		assert.Equal(t, "90", result["rotate"])
		assert.Equal(t, "horizontal", result["flip"])
		assert.Equal(t, "10", result["brightness"])
		assert.Equal(t, "20", result["contrast"])
		assert.Equal(t, "-10", result["saturation"])
		assert.Equal(t, "180", result["hue"])
		assert.Equal(t, "#FF0000", result["tint"])
		assert.Equal(t, "attention", result["gravity"])
		assert.Equal(t, "10", result["radius"])
		_, hasSizes := result["sizes"]
		assert.False(t, hasSizes)
		_, hasFormat := result["format"]
		assert.False(t, hasFormat)
		_, hasFit := result["fit"]
		assert.False(t, hasFit)
		_, hasPlaceholder := result["placeholder"]
		assert.False(t, hasPlaceholder)
	})

	t.Run("handles empty params", func(t *testing.T) {
		t.Parallel()

		result := orchestrator.getPassthroughModifiers(map[string][]string{})
		assert.Empty(t, result)
	})
}
func Test_AssetPipelineOrchestrator_calculateRequiredWidths(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	t.Run("combines sizes and densities", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"400px", "800px"}
		densities := []string{"1x", "2x"}

		result := orchestrator.calculateRequiredWidths(sizes, densities)

		assert.Contains(t, result, 400)
		assert.Contains(t, result, 800)
		assert.Contains(t, result, 1600)
	})

	t.Run("returns nil for empty sizes", func(t *testing.T) {
		t.Parallel()

		result := orchestrator.calculateRequiredWidths(nil, []string{"2x"})
		assert.Nil(t, result)
	})

	t.Run("results are sorted", func(t *testing.T) {
		t.Parallel()

		sizes := []string{"800px", "400px", "1200px"}
		result := orchestrator.calculateRequiredWidths(sizes, nil)

		require.Len(t, result, 3)
		assert.Equal(t, 400, result[0])
		assert.Equal(t, 800, result[1])
		assert.Equal(t, 1200, result[2])
	})
}
func Test_AssetPipelineOrchestrator_generatePlaceholderProfile(t *testing.T) {
	t.Parallel()

	orchestrator := &AssetPipelineOrchestrator{}

	t.Run("creates placeholder with defaults", func(t *testing.T) {
		t.Parallel()

		params := map[string][]string{
			"placeholder": {"true"},
		}

		profile := orchestrator.generatePlaceholderProfile(params, 80, nil, nil)

		require.NotNil(t, profile)
		assert.Equal(t, "placeholder", profile.Name)
		assert.Equal(t, registry_dto.PriorityNeed, profile.Profile.Priority)
		width, _ := profile.Profile.Params.GetByName("placeholder-width")
		assert.Equal(t, "20", width)

		quality, _ := profile.Profile.Params.GetByName("placeholder-quality")
		assert.Equal(t, "10", quality)

		blur, _ := profile.Profile.Params.GetByName("placeholder-blur")
		assert.Equal(t, "5.0", blur)

		format, _ := profile.Profile.Params.GetByName("format")
		assert.Equal(t, "webp", format)
	})

	t.Run("uses custom placeholder params", func(t *testing.T) {
		t.Parallel()

		params := map[string][]string{
			"placeholder":         {"true"},
			"placeholder-width":   {"30"},
			"placeholder-height":  {"20"},
			"placeholder-quality": {"15"},
			"placeholder-blur":    {"3.0"},
			"format":              {"jpeg,webp"},
		}

		profile := orchestrator.generatePlaceholderProfile(params, 80, nil, nil)

		require.NotNil(t, profile)

		width, _ := profile.Profile.Params.GetByName("placeholder-width")
		assert.Equal(t, "30", width)

		height, _ := profile.Profile.Params.GetByName("placeholder-height")
		assert.Equal(t, "20", height)

		quality, _ := profile.Profile.Params.GetByName("placeholder-quality")
		assert.Equal(t, "15", quality)

		blur, _ := profile.Profile.Params.GetByName("placeholder-blur")
		assert.Equal(t, "3.0", blur)

		format, _ := profile.Profile.Params.GetByName("format")
		assert.Equal(t, "jpeg", format)
	})

	t.Run("includes standard params in placeholder", func(t *testing.T) {
		t.Parallel()

		params := map[string][]string{
			"placeholder": {"true"},
		}
		standardParams := map[string]string{
			"fit":         "cover",
			"aspectratio": "16:9",
		}

		profile := orchestrator.generatePlaceholderProfile(params, 80, standardParams, nil)

		require.NotNil(t, profile)

		fit, _ := profile.Profile.Params.GetByName("fit")
		assert.Equal(t, "cover", fit)

		aspectratio, _ := profile.Profile.Params.GetByName("aspectratio")
		assert.Equal(t, "16:9", aspectratio)
	})
}
func TestBuildVariantParams(t *testing.T) {
	t.Parallel()

	t.Run("sets width format and quality", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality:        80,
			standardParams: nil,
			modifiers:      nil,
		}

		params := buildVariantParams(400, "webp", config)

		width, ok := params.GetByName("width")
		assert.True(t, ok)
		assert.Equal(t, "400", width)

		format, ok := params.GetByName("format")
		assert.True(t, ok)
		assert.Equal(t, "webp", format)

		quality, ok := params.GetByName("quality")
		assert.True(t, ok)
		assert.Equal(t, "80", quality)
	})

	t.Run("includes standard params", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality: 80,
			standardParams: map[string]string{
				"fit":         "cover",
				"aspectratio": "16:9",
			},
			modifiers: nil,
		}

		params := buildVariantParams(800, "jpeg", config)

		fit, ok := params.GetByName("fit")
		assert.True(t, ok)
		assert.Equal(t, "cover", fit)

		ar, ok := params.GetByName("aspectratio")
		assert.True(t, ok)
		assert.Equal(t, "16:9", ar)
	})

	t.Run("includes modifiers", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality:        80,
			standardParams: nil,
			modifiers: map[string]string{
				"greyscale": "true",
				"blur":      "5.0",
			},
		}

		params := buildVariantParams(640, "avif", config)

		greyscale, ok := params.GetByName("greyscale")
		assert.True(t, ok)
		assert.Equal(t, "true", greyscale)

		blur, ok := params.GetByName("blur")
		assert.True(t, ok)
		assert.Equal(t, "5.0", blur)
	})

	t.Run("combines all param types", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality: 90,
			standardParams: map[string]string{
				"fit": "contain",
			},
			modifiers: map[string]string{
				"sharpen": "2.0",
			},
		}

		params := buildVariantParams(1200, "webp", config)

		width, _ := params.GetByName("width")
		assert.Equal(t, "1200", width)

		format, _ := params.GetByName("format")
		assert.Equal(t, "webp", format)

		quality, _ := params.GetByName("quality")
		assert.Equal(t, "90", quality)

		fit, _ := params.GetByName("fit")
		assert.Equal(t, "contain", fit)

		sharpen, _ := params.GetByName("sharpen")
		assert.Equal(t, "2.0", sharpen)
	})

	t.Run("handles empty config", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality:        0,
			standardParams: nil,
			modifiers:      nil,
		}

		params := buildVariantParams(320, "png", config)

		width, _ := params.GetByName("width")
		assert.Equal(t, "320", width)

		format, _ := params.GetByName("format")
		assert.Equal(t, "png", format)

		quality, _ := params.GetByName("quality")
		assert.Equal(t, "0", quality)
	})
}
func TestAddViewportWidths(t *testing.T) {
	t.Parallel()

	t.Run("adds widths for 100vw", func(t *testing.T) {
		t.Parallel()

		widths := make(map[int]struct{})
		addViewportWidths("100", widths)
		assert.Contains(t, widths, 320)
		assert.Contains(t, widths, 640)
		assert.Contains(t, widths, 768)
		assert.Contains(t, widths, 1024)
		assert.Contains(t, widths, 1280)
		assert.Contains(t, widths, 1536)
		assert.Contains(t, widths, 1920)
	})

	t.Run("adds widths for 50vw", func(t *testing.T) {
		t.Parallel()

		widths := make(map[int]struct{})
		addViewportWidths("50", widths)

		assert.Contains(t, widths, 160)
		assert.Contains(t, widths, 320)
		assert.Contains(t, widths, 384)
		assert.Contains(t, widths, 512)
		assert.Contains(t, widths, 640)
		assert.Contains(t, widths, 768)
		assert.Contains(t, widths, 960)
	})

	t.Run("skips zero or negative results", func(t *testing.T) {
		t.Parallel()

		widths := make(map[int]struct{})
		addViewportWidths("0", widths)

		assert.Empty(t, widths)
	})

	t.Run("handles invalid vw value", func(t *testing.T) {
		t.Parallel()

		widths := make(map[int]struct{})
		addViewportWidths("invalid", widths)

		assert.Empty(t, widths)
	})
}
func TestCreateVariantProfile(t *testing.T) {
	t.Parallel()

	t.Run("creates profile with correct name", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality:        80,
			standardParams: nil,
			modifiers:      nil,
		}

		profile := createVariantProfile(400, "webp", config)

		assert.Equal(t, "image_w400_webp", profile.Name)
	})

	t.Run("sets correct capability and priority", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality: 80,
		}

		profile := createVariantProfile(800, "jpeg", config)

		assert.Equal(t, "image-transform", profile.Profile.CapabilityName)
		assert.Equal(t, registry_dto.PriorityWant, profile.Profile.Priority)
	})

	t.Run("sets resulting tags", func(t *testing.T) {
		t.Parallel()

		config := imageProfileConfig{
			quality: 80,
		}

		profile := createVariantProfile(640, "avif", config)

		tagType, ok := profile.Profile.ResultingTags.GetByName("type")
		assert.True(t, ok)
		assert.Equal(t, "image-variant", tagType)
	})
}
