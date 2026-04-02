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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestParseDensity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "x1 format returns 1.0",
			input:    "x1",
			expected: 1.0,
		},
		{
			name:     "x2 format returns 2.0",
			input:    "x2",
			expected: 2.0,
		},
		{
			name:     "2x format returns 2.0",
			input:    "2x",
			expected: 2.0,
		},
		{
			name:     "1x format returns 1.0",
			input:    "1x",
			expected: 1.0,
		},
		{
			name:     "decimal x1.5 format returns 1.5",
			input:    "x1.5",
			expected: 1.5,
		},
		{
			name:     "decimal 1.5x format returns 1.5",
			input:    "1.5x",
			expected: 1.5,
		},
		{
			name:     "X2 uppercase format returns 2.0",
			input:    "X2",
			expected: 2.0,
		},
		{
			name:     "2X uppercase suffix returns 2.0",
			input:    "2X",
			expected: 2.0,
		},
		{
			name:     "empty string returns default 1.0",
			input:    "",
			expected: 1.0,
		},
		{
			name:     "whitespace only returns default 1.0",
			input:    "   ",
			expected: 1.0,
		},
		{
			name:     "invalid string returns default 1.0",
			input:    "invalid",
			expected: 1.0,
		},
		{
			name:     "negative value returns default 1.0",
			input:    "x-1",
			expected: 1.0,
		},
		{
			name:     "zero value returns default 1.0",
			input:    "x0",
			expected: 1.0,
		},
		{
			name:     "x3 format returns 3.0",
			input:    "x3",
			expected: 3.0,
		},
		{
			name:     "leading whitespace is trimmed",
			input:    "  x2",
			expected: 2.0,
		},
		{
			name:     "trailing whitespace is trimmed",
			input:    "x2  ",
			expected: 2.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseDensity(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPixelWidth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "valid pixel width returns value",
			input:    "100px",
			expected: 100,
		},
		{
			name:     "large pixel width returns value",
			input:    "1920px",
			expected: 1920,
		},
		{
			name:     "single pixel returns value",
			input:    "1px",
			expected: 1,
		},
		{
			name:     "missing px suffix returns 0",
			input:    "100",
			expected: 0,
		},
		{
			name:     "vw suffix returns 0",
			input:    "100vw",
			expected: 0,
		},
		{
			name:     "percentage suffix returns 0",
			input:    "50%",
			expected: 0,
		},
		{
			name:     "zero pixels returns 0",
			input:    "0px",
			expected: 0,
		},
		{
			name:     "negative pixels returns 0",
			input:    "-100px",
			expected: 0,
		},
		{
			name:     "invalid number returns 0",
			input:    "abcpx",
			expected: 0,
		},
		{
			name:     "empty string returns 0",
			input:    "",
			expected: 0,
		},
		{
			name:     "decimal pixels returns 0 (Atoi fails)",
			input:    "100.5px",
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractPixelWidth(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetViewportWidth(t *testing.T) {
	t.Parallel()

	screens := map[string]int{
		"sm":  640,
		"md":  768,
		"lg":  1024,
		"xl":  1280,
		"2xl": 1536,
	}

	testCases := []struct {
		name       string
		screens    map[string]int
		breakpoint string
		expected   int
	}{
		{
			name:       "empty breakpoint returns default viewport width",
			screens:    screens,
			breakpoint: "",
			expected:   defaultViewportWidth,
		},
		{
			name:       "sm breakpoint returns 640",
			screens:    screens,
			breakpoint: "sm",
			expected:   640,
		},
		{
			name:       "md breakpoint returns 768",
			screens:    screens,
			breakpoint: "md",
			expected:   768,
		},
		{
			name:       "lg breakpoint returns 1024",
			screens:    screens,
			breakpoint: "lg",
			expected:   1024,
		},
		{
			name:       "xl breakpoint returns 1280",
			screens:    screens,
			breakpoint: "xl",
			expected:   1280,
		},
		{
			name:       "2xl breakpoint returns 1536",
			screens:    screens,
			breakpoint: "2xl",
			expected:   1536,
		},
		{
			name:       "unknown breakpoint returns default",
			screens:    screens,
			breakpoint: "unknown",
			expected:   defaultViewportWidth,
		},
		{
			name:       "nil screens with breakpoint returns default",
			screens:    nil,
			breakpoint: "sm",
			expected:   defaultViewportWidth,
		},
		{
			name:       "nil screens with empty breakpoint returns default",
			screens:    nil,
			breakpoint: "",
			expected:   defaultViewportWidth,
		},
		{
			name:       "empty screens map returns default",
			screens:    map[string]int{},
			breakpoint: "sm",
			expected:   defaultViewportWidth,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getViewportWidth(tc.screens, tc.breakpoint)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractViewportWidth(t *testing.T) {
	t.Parallel()

	screens := map[string]int{
		"sm": 640,
		"md": 768,
		"lg": 1024,
	}

	testCases := []struct {
		name       string
		sizeValue  string
		screens    map[string]int
		breakpoint string
		expected   int
	}{
		{
			name:       "100vw with no breakpoint uses default viewport",
			sizeValue:  "100vw",
			screens:    screens,
			breakpoint: "",
			expected:   defaultViewportWidth,
		},
		{
			name:       "50vw with no breakpoint returns half default viewport",
			sizeValue:  "50vw",
			screens:    screens,
			breakpoint: "",
			expected:   defaultViewportWidth / 2,
		},
		{
			name:       "100vw with sm breakpoint returns 640",
			sizeValue:  "100vw",
			screens:    screens,
			breakpoint: "sm",
			expected:   640,
		},
		{
			name:       "50vw with md breakpoint returns 384",
			sizeValue:  "50vw",
			screens:    screens,
			breakpoint: "md",
			expected:   384,
		},
		{
			name:       "25vw with lg breakpoint returns 256",
			sizeValue:  "25vw",
			screens:    screens,
			breakpoint: "lg",
			expected:   256,
		},
		{
			name:       "missing vw suffix returns 0",
			sizeValue:  "100",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "px suffix returns 0",
			sizeValue:  "100px",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "invalid number returns 0",
			sizeValue:  "abcvw",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "empty string returns 0",
			sizeValue:  "",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "decimal viewport percentage works",
			sizeValue:  "33.33vw",
			screens:    screens,
			breakpoint: "lg",
			expected:   341,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractViewportWidth(tc.sizeValue, tc.screens, tc.breakpoint)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractPercentageWidth(t *testing.T) {
	t.Parallel()

	screens := map[string]int{
		"sm": 640,
		"md": 768,
		"lg": 1024,
	}

	testCases := []struct {
		name       string
		sizeValue  string
		screens    map[string]int
		breakpoint string
		expected   int
	}{
		{
			name:       "100% with no breakpoint uses default viewport",
			sizeValue:  "100%",
			screens:    screens,
			breakpoint: "",
			expected:   defaultViewportWidth,
		},
		{
			name:       "50% with no breakpoint returns half default viewport",
			sizeValue:  "50%",
			screens:    screens,
			breakpoint: "",
			expected:   defaultViewportWidth / 2,
		},
		{
			name:       "100% with sm breakpoint returns 640",
			sizeValue:  "100%",
			screens:    screens,
			breakpoint: "sm",
			expected:   640,
		},
		{
			name:       "50% with md breakpoint returns 384",
			sizeValue:  "50%",
			screens:    screens,
			breakpoint: "md",
			expected:   384,
		},
		{
			name:       "25% with lg breakpoint returns 256",
			sizeValue:  "25%",
			screens:    screens,
			breakpoint: "lg",
			expected:   256,
		},
		{
			name:       "missing % suffix returns 0",
			sizeValue:  "100",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "vw suffix returns 0",
			sizeValue:  "100vw",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "px suffix returns 0",
			sizeValue:  "100px",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "invalid number returns 0",
			sizeValue:  "abc%",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "empty string returns 0",
			sizeValue:  "",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "decimal percentage works",
			sizeValue:  "33.33%",
			screens:    screens,
			breakpoint: "lg",
			expected:   341,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractPercentageWidth(tc.sizeValue, tc.screens, tc.breakpoint)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractWidth(t *testing.T) {
	t.Parallel()

	screens := map[string]int{
		"sm": 640,
		"md": 768,
		"lg": 1024,
	}

	testCases := []struct {
		name       string
		sizeValue  string
		screens    map[string]int
		breakpoint string
		expected   int
	}{
		{
			name:       "pixel value is extracted",
			sizeValue:  "800px",
			screens:    screens,
			breakpoint: "",
			expected:   800,
		},
		{
			name:       "viewport width is extracted",
			sizeValue:  "50vw",
			screens:    screens,
			breakpoint: "lg",
			expected:   512,
		},
		{
			name:       "percentage is extracted",
			sizeValue:  "50%",
			screens:    screens,
			breakpoint: "sm",
			expected:   320,
		},
		{
			name:       "pixel takes precedence",
			sizeValue:  "100px",
			screens:    screens,
			breakpoint: "sm",
			expected:   100,
		},
		{
			name:       "whitespace is trimmed",
			sizeValue:  "  800px  ",
			screens:    screens,
			breakpoint: "",
			expected:   800,
		},
		{
			name:       "invalid value returns 0",
			sizeValue:  "invalid",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "empty string returns 0",
			sizeValue:  "",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
		{
			name:       "em unit returns 0 (unsupported)",
			sizeValue:  "100em",
			screens:    screens,
			breakpoint: "",
			expected:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractWidth(tc.sizeValue, tc.screens, tc.breakpoint)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseSizes(t *testing.T) {
	t.Parallel()

	screens := map[string]int{
		"sm": 640,
		"md": 768,
		"lg": 1024,
	}

	testCases := []struct {
		name        string
		sizesString string
		screens     map[string]int
		expected    []int
	}{
		{
			name:        "empty string returns nil",
			sizesString: "",
			screens:     screens,
			expected:    nil,
		},
		{
			name:        "single pixel value",
			sizesString: "800px",
			screens:     screens,
			expected:    []int{800},
		},
		{
			name:        "single viewport width",
			sizesString: "100vw",
			screens:     screens,
			expected:    []int{defaultViewportWidth},
		},
		{
			name:        "breakpoint with pixel value",
			sizesString: "sm:800px",
			screens:     screens,
			expected:    []int{800},
		},
		{
			name:        "breakpoint with viewport width",
			sizesString: "sm:50vw",
			screens:     screens,
			expected:    []int{320},
		},
		{
			name:        "multiple sizes sorted",
			sizesString: "1024px 640px 320px",
			screens:     screens,
			expected:    []int{320, 640, 1024},
		},
		{
			name:        "mixed breakpoint and default",
			sizesString: "100vw sm:50vw md:800px",
			screens:     screens,
			expected:    []int{320, 800, defaultViewportWidth},
		},
		{
			name:        "duplicates are removed",
			sizesString: "800px 800px",
			screens:     screens,
			expected:    []int{800},
		},
		{
			name:        "invalid values are skipped",
			sizesString: "800px invalid 640px",
			screens:     screens,
			expected:    []int{640, 800},
		},
		{
			name:        "all invalid returns empty slice",
			sizesString: "invalid notasize",
			screens:     screens,
			expected:    []int{},
		},
		{
			name:        "percentage values",
			sizesString: "50% sm:50%",
			screens:     screens,
			expected:    []int{320, defaultViewportWidth / 2},
		},
		{
			name:        "malformed breakpoint (single colon) skipped",
			sizesString: ": 800px",
			screens:     screens,
			expected:    []int{800},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseSizes(tc.sizesString, tc.screens)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCalculateResponsiveVariants(t *testing.T) {
	t.Parallel()

	screens := map[string]int{
		"sm": 640,
		"md": 768,
		"lg": 1024,
	}

	testCases := []struct {
		screens          map[string]int
		name             string
		sizesString      string
		densities        []string
		defaultDensities []string
		expectedWidths   []int
		baseWidth        int
		expectedCount    int
	}{
		{
			name:             "uses base width when no sizes",
			baseWidth:        800,
			sizesString:      "",
			densities:        []string{"x1"},
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    1,
			expectedWidths:   []int{800},
		},
		{
			name:             "multiple densities multiply width",
			baseWidth:        400,
			sizesString:      "",
			densities:        []string{"x1", "x2"},
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    2,
			expectedWidths:   []int{400, 800},
		},
		{
			name:             "uses default densities when none provided",
			baseWidth:        500,
			sizesString:      "",
			densities:        nil,
			defaultDensities: []string{"x1", "x2"},
			screens:          screens,
			expectedCount:    2,
			expectedWidths:   []int{500, 1000},
		},
		{
			name:             "falls back to x1 when no densities",
			baseWidth:        600,
			sizesString:      "",
			densities:        nil,
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    1,
			expectedWidths:   []int{600},
		},
		{
			name:             "uses default breakpoints when no base width or sizes",
			baseWidth:        0,
			sizesString:      "",
			densities:        []string{"x1"},
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    len(defaultResponsiveBreakpoints),
			expectedWidths:   defaultResponsiveBreakpoints,
		},
		{
			name:             "parses sizes attribute",
			baseWidth:        0,
			sizesString:      "800px 400px",
			densities:        []string{"x1"},
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    2,
			expectedWidths:   []int{400, 800},
		},
		{
			name:             "deduplicates identical widths",
			baseWidth:        400,
			sizesString:      "400px",
			densities:        []string{"x1"},
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    1,
			expectedWidths:   []int{400},
		},
		{
			name:             "combines sizes and densities",
			baseWidth:        0,
			sizesString:      "400px 800px",
			densities:        []string{"x1", "x2"},
			defaultDensities: nil,
			screens:          screens,
			expectedCount:    4,
			expectedWidths:   []int{400, 800, 800, 1600},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := calculateResponsiveVariants(
				tc.baseWidth,
				tc.sizesString,
				tc.densities,
				tc.defaultDensities,
				tc.screens,
			)
			assert.Len(t, result, tc.expectedCount)

			actualWidths := make([]int, len(result))
			for i, v := range result {
				actualWidths[i] = v.Width
			}
			assert.Equal(t, tc.expectedWidths, actualWidths)
		})
	}
}

func TestCalculateResponsiveVariants_Sorting(t *testing.T) {
	t.Parallel()

	screens := map[string]int{}

	result := calculateResponsiveVariants(
		0,
		"1000px 500px",
		[]string{"x2", "x1"},
		nil,
		screens,
	)

	require.Len(t, result, 4)

	assert.Equal(t, 500, result[0].Width)
	assert.Equal(t, "x1", result[0].Density)

	assert.Equal(t, 1000, result[1].Width)

	assert.Equal(t, "x1", result[1].Density)

	assert.Equal(t, 1000, result[2].Width)
	assert.Equal(t, "x2", result[2].Density)

	assert.Equal(t, 2000, result[3].Width)
	assert.Equal(t, "x2", result[3].Density)
}

func TestCreateVariantDependency(t *testing.T) {
	t.Parallel()

	t.Run("creates variant with correct fields", func(t *testing.T) {
		t.Parallel()

		baseDep := &annotator_dto.StaticAssetDependency{
			SourcePath: "/images/photo.jpg",
			AssetType:  "img",
			TransformationParams: map[string]string{
				"format":      "webp",
				"quality":     "80",
				"width":       "800",
				"_responsive": "true",
				"densities":   "x1 x2",
				"sizes":       "100vw",
			},
			OriginComponentPath: "/components/Hero.pk",
			Location: ast_domain.Location{
				Line:   10,
				Column: 5,
			},
		}

		variant := ResponsiveVariantSpec{
			Width:   1600,
			Density: "x2",
		}

		result := createVariantDependency(baseDep, variant)

		assert.Equal(t, baseDep.SourcePath, result.SourcePath)
		assert.Equal(t, baseDep.AssetType, result.AssetType)
		assert.Equal(t, baseDep.OriginComponentPath, result.OriginComponentPath)
		assert.Equal(t, baseDep.Location, result.Location)

		assert.Equal(t, "webp", result.TransformationParams["format"])
		assert.Equal(t, "80", result.TransformationParams["quality"])

		assert.Equal(t, "1600", result.TransformationParams["width"])
		assert.Equal(t, "x2", result.TransformationParams["_density"])

		_, hasResponsive := result.TransformationParams["_responsive"]
		_, hasDensities := result.TransformationParams["densities"]
		_, hasSizes := result.TransformationParams["sizes"]
		assert.False(t, hasResponsive, "_responsive should be removed")
		assert.False(t, hasDensities, "densities should be removed")
		assert.False(t, hasSizes, "sizes should be removed")
	})

	t.Run("does not modify original dependency", func(t *testing.T) {
		t.Parallel()

		baseDep := &annotator_dto.StaticAssetDependency{
			SourcePath: "/images/photo.jpg",
			AssetType:  "img",
			TransformationParams: map[string]string{
				"width":       "800",
				"_responsive": "true",
			},
		}

		variant := ResponsiveVariantSpec{
			Width:   1600,
			Density: "x2",
		}

		_ = createVariantDependency(baseDep, variant)

		assert.Equal(t, "800", baseDep.TransformationParams["width"])
		assert.Equal(t, "true", baseDep.TransformationParams["_responsive"])
		_, hasDensity := baseDep.TransformationParams["_density"]
		assert.False(t, hasDensity, "original should not have _density")
	})

	t.Run("handles empty transformation params", func(t *testing.T) {
		t.Parallel()

		baseDep := &annotator_dto.StaticAssetDependency{
			SourcePath:           "/images/photo.jpg",
			AssetType:            "img",
			TransformationParams: map[string]string{},
		}

		variant := ResponsiveVariantSpec{
			Width:   400,
			Density: "x1",
		}

		result := createVariantDependency(baseDep, variant)

		assert.Equal(t, "400", result.TransformationParams["width"])
		assert.Equal(t, "x1", result.TransformationParams["_density"])
	})
}

func TestResponsiveVariantSpec(t *testing.T) {
	t.Parallel()

	t.Run("struct fields are accessible", func(t *testing.T) {
		t.Parallel()

		spec := ResponsiveVariantSpec{
			Density: "x2",
			Width:   1920,
		}

		assert.Equal(t, "x2", spec.Density)
		assert.Equal(t, 1920, spec.Width)
	})

	t.Run("zero value is valid", func(t *testing.T) {
		t.Parallel()

		var spec ResponsiveVariantSpec

		assert.Equal(t, "", spec.Density)
		assert.Equal(t, 0, spec.Width)
	})
}
