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
	"slices"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestAggregateProjectAssets_EmptyInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []*annotator_dto.AnnotationResult
		expected int
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: 0,
		},
		{
			name:     "empty slice",
			input:    []*annotator_dto.AnnotationResult{},
			expected: 0,
		},
		{
			name: "slice with nil results",
			input: []*annotator_dto.AnnotationResult{
				nil,
				nil,
			},
			expected: 0,
		},
		{
			name: "slice with results having nil dependencies",
			input: []*annotator_dto.AnnotationResult{
				{AssetDependencies: nil},
				{AssetDependencies: nil},
			},
			expected: 0,
		},
		{
			name: "slice with results having empty dependencies",
			input: []*annotator_dto.AnnotationResult{
				{AssetDependencies: []*annotator_dto.StaticAssetDependency{}},
				{AssetDependencies: []*annotator_dto.StaticAssetDependency{}},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := AggregateProjectAssets(tt.input)

			if len(result) != tt.expected {
				t.Errorf("Expected %d assets, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestAggregateProjectAssets_SingleAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validateResult func(*testing.T, []*annotator_dto.FinalAssetDependency)
		name           string
		input          []*annotator_dto.AnnotationResult
		expectedCount  int
	}{
		{
			name: "single page with single asset no params",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "img/hero.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				if result[0].SourcePath != "img/hero.jpg" {
					t.Errorf("Expected source path 'img/hero.jpg', got '%s'", result[0].SourcePath)
				}
				if result[0].AssetType != "image" {
					t.Errorf("Expected asset type 'image', got '%s'", result[0].AssetType)
				}
				if len(result[0].TransformationParams) != 0 {
					t.Errorf("Expected no transformation params, got %d", len(result[0].TransformationParams))
				}
			},
		},
		{
			name: "single page with single asset with single param",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "800",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				if len(result[0].TransformationParams) != 1 {
					t.Errorf("Expected 1 transformation param, got %d", len(result[0].TransformationParams))
				}
				if widths, ok := result[0].TransformationParams["width"]; !ok {
					t.Error("Expected 'width' param to exist")
				} else if len(widths) != 1 || widths[0] != "800" {
					t.Errorf("Expected width value ['800'], got %v", widths)
				}
			},
		},
		{
			name: "single page with single asset with multiple params",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width":  "800",
								"height": "600",
								"format": "webp",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				if len(result[0].TransformationParams) != 3 {
					t.Errorf("Expected 3 transformation params, got %d", len(result[0].TransformationParams))
				}
				if widths, ok := result[0].TransformationParams["width"]; !ok || len(widths) != 1 || widths[0] != "800" {
					t.Errorf("Expected width value ['800'], got %v", widths)
				}
				if heights, ok := result[0].TransformationParams["height"]; !ok || len(heights) != 1 || heights[0] != "600" {
					t.Errorf("Expected height value ['600'], got %v", heights)
				}
				if formats, ok := result[0].TransformationParams["format"]; !ok || len(formats) != 1 || formats[0] != "webp" {
					t.Errorf("Expected format value ['webp'], got %v", formats)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := AggregateProjectAssets(tt.input)

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d assets, got %d", tt.expectedCount, len(result))
				return
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestAggregateProjectAssets_Deduplication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validateResult func(*testing.T, []*annotator_dto.FinalAssetDependency)
		name           string
		input          []*annotator_dto.AnnotationResult
		expectedCount  int
	}{
		{
			name: "same asset on two pages no params",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "img/logo.png",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "img/logo.png",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				if result[0].SourcePath != "img/logo.png" {
					t.Errorf("Expected source path 'img/logo.png', got '%s'", result[0].SourcePath)
				}
			},
		},
		{
			name: "same asset on three pages with same params",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/logo.png",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "100",
							},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/logo.png",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "100",
							},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/logo.png",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "100",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				if widths, ok := result[0].TransformationParams["width"]; !ok {
					t.Error("Expected 'width' param to exist")
				} else if len(widths) != 1 || widths[0] != "100" {
					t.Errorf("Expected width value ['100'], got %v", widths)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := AggregateProjectAssets(tt.input)

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d assets, got %d", tt.expectedCount, len(result))
				return
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestAggregateProjectAssets_ParameterMerging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validateResult func(*testing.T, []*annotator_dto.FinalAssetDependency)
		name           string
		input          []*annotator_dto.AnnotationResult
		expectedCount  int
	}{
		{
			name: "same asset with different param values on different pages",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/product.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "300",
							},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/product.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "600",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				widths := result[0].TransformationParams["width"]
				if len(widths) != 2 {
					t.Errorf("Expected 2 width values, got %d", len(widths))
					return
				}
				if widths[0] != "300" || widths[1] != "600" {
					t.Errorf("Expected sorted widths ['300', '600'], got %v", widths)
				}
			},
		},
		{
			name: "same asset with comma-separated values",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"widths": "300,600,900",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				widths := result[0].TransformationParams["widths"]
				if len(widths) != 3 {
					t.Errorf("Expected 3 width values, got %d", len(widths))
					return
				}
				if widths[0] != "300" || widths[1] != "600" || widths[2] != "900" {
					t.Errorf("Expected sorted widths ['300', '600', '900'], got %v", widths)
				}
			},
		},
		{
			name: "same asset with comma-separated values on multiple pages",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"widths": "300,600",
							},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"widths": "600,900",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				widths := result[0].TransformationParams["widths"]
				if len(widths) != 3 {
					t.Errorf("Expected 3 unique width values, got %d", len(widths))
					return
				}
				if widths[0] != "300" || widths[1] != "600" || widths[2] != "900" {
					t.Errorf("Expected sorted deduplicated widths ['300', '600', '900'], got %v", widths)
				}
			},
		},
		{
			name: "same asset with whitespace in comma-separated values",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"widths": " 300 , 600 , 900 ",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				widths := result[0].TransformationParams["widths"]
				if len(widths) != 3 {
					t.Errorf("Expected 3 width values, got %d", len(widths))
					return
				}
				if widths[0] != "300" || widths[1] != "600" || widths[2] != "900" {
					t.Errorf("Expected trimmed widths ['300', '600', '900'], got %v", widths)
				}
			},
		},
		{
			name: "same asset with empty values in comma-separated list",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"widths": "300,,600,,900",
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				widths := result[0].TransformationParams["widths"]
				if len(widths) != 3 {
					t.Errorf("Expected 3 width values (empty filtered), got %d", len(widths))
					return
				}
				if widths[0] != "300" || widths[1] != "600" || widths[2] != "900" {
					t.Errorf("Expected filtered widths ['300', '600', '900'], got %v", widths)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := AggregateProjectAssets(tt.input)

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d assets, got %d", tt.expectedCount, len(result))
				return
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestAggregateProjectAssets_MultipleAssets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		validateResult func(*testing.T, []*annotator_dto.FinalAssetDependency)
		name           string
		input          []*annotator_dto.AnnotationResult
		expectedCount  int
	}{
		{
			name: "single page with multiple different assets",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "img/hero.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "img/logo.png",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "styles/main.css",
							AssetType:            "stylesheet",
							TransformationParams: map[string]string{},
						},
					},
				},
			},
			expectedCount: 3,
		},
		{
			name: "multiple pages with different assets",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "800",
							},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/logo.png",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "100",
							},
						},
					},
				},
			},
			expectedCount: 2,
		},
		{
			name: "multiple pages with mix of shared and unique assets",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/logo.png",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "100",
							},
						},
						{
							SourcePath: "img/hero.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "800",
							},
						},
					},
				},
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath: "img/logo.png",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "100",
							},
						},
						{
							SourcePath: "img/product.jpg",
							AssetType:  "image",
							TransformationParams: map[string]string{
								"width": "600",
							},
						},
					},
				},
			},
			expectedCount: 3,
			validateResult: func(t *testing.T, result []*annotator_dto.FinalAssetDependency) {
				foundLogo := false
				foundHero := false
				foundProduct := false
				for _, asset := range result {
					switch asset.SourcePath {
					case "img/logo.png":
						foundLogo = true
					case "img/hero.jpg":
						foundHero = true
					case "img/product.jpg":
						foundProduct = true
					}
				}
				if !foundLogo || !foundHero || !foundProduct {
					t.Errorf("Expected all three assets, found: logo=%v hero=%v product=%v", foundLogo, foundHero, foundProduct)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := AggregateProjectAssets(tt.input)

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d assets, got %d", tt.expectedCount, len(result))
				return
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestAggregateProjectAssets_Sorting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         []*annotator_dto.AnnotationResult
		expectedOrder []string
	}{
		{
			name: "assets sorted by source path",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "img/zebra.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "img/apple.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "img/monkey.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
					},
				},
			},
			expectedOrder: []string{"img/apple.jpg", "img/monkey.jpg", "img/zebra.jpg"},
		},
		{
			name: "same path different types sorted by type",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "assets/main",
							AssetType:            "stylesheet",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "assets/main",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
					},
				},
			},
			expectedOrder: []string{"assets/main", "assets/main"},
		},
		{
			name: "complex sorting scenario",
			input: []*annotator_dto.AnnotationResult{
				{
					AssetDependencies: []*annotator_dto.StaticAssetDependency{
						{
							SourcePath:           "z/file.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "a/file.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
						{
							SourcePath:           "m/file.jpg",
							AssetType:            "image",
							TransformationParams: map[string]string{},
						},
					},
				},
			},
			expectedOrder: []string{"a/file.jpg", "m/file.jpg", "z/file.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := AggregateProjectAssets(tt.input)

			if len(result) != len(tt.expectedOrder) {
				t.Errorf("Expected %d assets, got %d", len(tt.expectedOrder), len(result))
				return
			}

			for i, expectedPath := range tt.expectedOrder {
				if result[i].SourcePath != expectedPath {
					t.Errorf("Expected asset at index %d to have path '%s', got '%s'", i, expectedPath, result[i].SourcePath)
				}
			}
		})
	}
}

func TestAggregateProjectAssets_ComplexScenario(t *testing.T) {
	t.Parallel()

	input := []*annotator_dto.AnnotationResult{
		{
			AssetDependencies: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "img/logo.png",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"width": "100",
					},
				},
				{
					SourcePath: "img/hero.jpg",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"widths": "800,1200",
						"format": "webp",
					},
				},
			},
		},
		{
			AssetDependencies: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "img/logo.png",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"width": "100",
					},
				},
				{
					SourcePath: "img/product.jpg",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"widths": "400,600,800",
					},
				},
			},
		},
		{
			AssetDependencies: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "img/logo.png",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"width": "100",
					},
				},
				{
					SourcePath: "img/hero.jpg",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"widths": "1200,1600",
						"format": "webp",
					},
				},
				{
					SourcePath: "img/gallery1.jpg",
					AssetType:  "image",
					TransformationParams: map[string]string{
						"widths": "300,600",
					},
				},
			},
		},
		nil,
	}

	result := AggregateProjectAssets(input)

	if len(result) != 4 {
		t.Errorf("Expected 4 unique assets, got %d", len(result))
		return
	}

	logoIndex := findAssetIndex(result, "img/logo.png")
	if logoIndex == -1 {
		t.Fatal("Expected to find logo.png in results")
	}
	logoWidths := result[logoIndex].TransformationParams["width"]
	if len(logoWidths) != 1 || logoWidths[0] != "100" {
		t.Errorf("Expected logo width ['100'], got %v", logoWidths)
	}

	heroIndex := findAssetIndex(result, "img/hero.jpg")
	if heroIndex == -1 {
		t.Fatal("Expected to find hero.jpg in results")
	}
	heroWidths := result[heroIndex].TransformationParams["widths"]
	if len(heroWidths) != 3 {
		t.Errorf("Expected 3 merged width values for hero, got %d", len(heroWidths))
	}
	expectedHeroWidths := []string{"1200", "1600", "800"}
	for i, expected := range expectedHeroWidths {
		if !slices.Contains(heroWidths, expected) {
			t.Errorf("Expected hero width '%s' at index %d to be in result", expected, i)
		}
	}

	productIndex := findAssetIndex(result, "img/product.jpg")
	if productIndex == -1 {
		t.Fatal("Expected to find product.jpg in results")
	}
	productWidths := result[productIndex].TransformationParams["widths"]
	if len(productWidths) != 3 {
		t.Errorf("Expected 3 width values for product, got %d", len(productWidths))
	}

	galleryIndex := findAssetIndex(result, "img/gallery1.jpg")
	if galleryIndex == -1 {
		t.Fatal("Expected to find gallery1.jpg in results")
	}

	if result[0].SourcePath > result[1].SourcePath ||
		result[1].SourcePath > result[2].SourcePath ||
		result[2].SourcePath > result[3].SourcePath {
		t.Error("Results are not sorted by source path")
	}
}

func findAssetIndex(assets []*annotator_dto.FinalAssetDependency, sourcePath string) int {
	for i, asset := range assets {
		if asset.SourcePath == sourcePath {
			return i
		}
	}
	return -1
}
