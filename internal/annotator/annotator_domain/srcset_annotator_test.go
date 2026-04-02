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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestGenerateVariantKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		variant  *annotator_dto.StaticAssetDependency
		expected string
	}{
		{
			name: "width and format specified",
			variant: &annotator_dto.StaticAssetDependency{
				TransformationParams: map[string]string{
					"width":  "300px",
					"format": "avif",
				},
			},
			expected: "image_w300_avif",
		},
		{
			name: "width without px suffix",
			variant: &annotator_dto.StaticAssetDependency{
				TransformationParams: map[string]string{
					"width":  "600",
					"format": "webp",
				},
			},
			expected: "image_w600_webp",
		},
		{
			name: "format defaults to webp when empty",
			variant: &annotator_dto.StaticAssetDependency{
				TransformationParams: map[string]string{
					"width": "200px",
				},
			},
			expected: "image_w200_webp",
		},
		{
			name: "empty width",
			variant: &annotator_dto.StaticAssetDependency{
				TransformationParams: map[string]string{
					"format": "png",
				},
			},
			expected: "image_w_png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := generateVariantKey(tc.variant)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateAssetURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		variant  *annotator_dto.StaticAssetDependency
		config   AnnotatorPathsConfig
		expected string
	}{
		{
			name: "custom artefact serve path",
			variant: &annotator_dto.StaticAssetDependency{
				SourcePath: "images/hero.jpg",
				TransformationParams: map[string]string{
					"width":  "300px",
					"format": "webp",
				},
			},
			config: AnnotatorPathsConfig{
				ArtefactServePath: "/custom/assets",
			},
			expected: "/custom/assets/images/hero.jpg?v=image_w300_webp",
		},
		{
			name: "default artefact serve path when empty",
			variant: &annotator_dto.StaticAssetDependency{
				SourcePath: "img/logo.png",
				TransformationParams: map[string]string{
					"width":  "100",
					"format": "avif",
				},
			},
			config:   AnnotatorPathsConfig{},
			expected: "/_piko/assets/img/logo.png?v=image_w100_avif",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := generateAssetURL(tc.variant, tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildSrcsetMetadata(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		variants []*annotator_dto.StaticAssetDependency
		config   AnnotatorPathsConfig
		expected []ast_domain.ResponsiveVariantMetadata
	}{
		{
			name:     "empty variants",
			variants: nil,
			config:   AnnotatorPathsConfig{},
			expected: []ast_domain.ResponsiveVariantMetadata{},
		},
		{
			name: "single variant with px suffix",
			variants: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "hero.jpg",
					TransformationParams: map[string]string{
						"width":    "400px",
						"height":   "300px",
						"_density": "2x",
						"format":   "webp",
					},
				},
			},
			config: AnnotatorPathsConfig{},
			expected: []ast_domain.ResponsiveVariantMetadata{
				{
					Width:      400,
					Height:     300,
					Density:    "2x",
					VariantKey: "image_w400_webp",
					URL:        "/_piko/assets/hero.jpg?v=image_w400_webp",
				},
			},
		},
		{
			name: "sorts by width then density",
			variants: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "hero.jpg",
					TransformationParams: map[string]string{
						"width":    "800px",
						"_density": "2x",
						"format":   "webp",
					},
				},
				{
					SourcePath: "hero.jpg",
					TransformationParams: map[string]string{
						"width":    "400px",
						"_density": "1x",
						"format":   "webp",
					},
				},
				{
					SourcePath: "hero.jpg",
					TransformationParams: map[string]string{
						"width":    "400px",
						"_density": "2x",
						"format":   "webp",
					},
				},
			},
			config: AnnotatorPathsConfig{},
			expected: []ast_domain.ResponsiveVariantMetadata{
				{
					Width:      400,
					Height:     0,
					Density:    "1x",
					VariantKey: "image_w400_webp",
					URL:        "/_piko/assets/hero.jpg?v=image_w400_webp",
				},
				{
					Width:      400,
					Height:     0,
					Density:    "2x",
					VariantKey: "image_w400_webp",
					URL:        "/_piko/assets/hero.jpg?v=image_w400_webp",
				},
				{
					Width:      800,
					Height:     0,
					Density:    "2x",
					VariantKey: "image_w800_webp",
					URL:        "/_piko/assets/hero.jpg?v=image_w800_webp",
				},
			},
		},
		{
			name: "density defaults to x1 when empty",
			variants: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "icon.png",
					TransformationParams: map[string]string{
						"width":  "50",
						"format": "webp",
					},
				},
			},
			config: AnnotatorPathsConfig{},
			expected: []ast_domain.ResponsiveVariantMetadata{
				{
					Width:      50,
					Height:     0,
					Density:    "x1",
					VariantKey: "image_w50_webp",
					URL:        "/_piko/assets/icon.png?v=image_w50_webp",
				},
			},
		},
		{
			name: "non-numeric width treated as zero",
			variants: []*annotator_dto.StaticAssetDependency{
				{
					SourcePath: "banner.jpg",
					TransformationParams: map[string]string{
						"width":    "auto",
						"height":   "invalid",
						"_density": "1x",
						"format":   "webp",
					},
				},
			},
			config: AnnotatorPathsConfig{},
			expected: []ast_domain.ResponsiveVariantMetadata{
				{
					Width:      0,
					Height:     0,
					Density:    "1x",
					VariantKey: "image_wauto_webp",
					URL:        "/_piko/assets/banner.jpg?v=image_wauto_webp",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := buildSrcsetMetadata(tc.variants, tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestProcessPikoImgNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node             *ast_domain.TemplateNode
		variantsBySource map[string][]*annotator_dto.StaticAssetDependency
		config           AnnotatorPathsConfig
		name             string
		wantContinue     bool
		wantSrcset       bool
	}{
		{
			name: "non-element node returns true without annotation",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				TagName:  "piko:img",
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{},
			config:           AnnotatorPathsConfig{},
			wantContinue:     true,
			wantSrcset:       false,
		},
		{
			name: "non-piko-img element returns true without annotation",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{},
			config:           AnnotatorPathsConfig{},
			wantContinue:     true,
			wantSrcset:       false,
		},
		{
			name: "piko:img without src attribute returns true",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:img",
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{},
			config:           AnnotatorPathsConfig{},
			wantContinue:     true,
			wantSrcset:       false,
		},
		{
			name: "piko:img with empty src returns true",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: ""},
				},
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{},
			config:           AnnotatorPathsConfig{},
			wantContinue:     true,
			wantSrcset:       false,
		},
		{
			name: "piko:img with src but no variants returns true",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "hero.jpg"},
				},
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{},
			config:           AnnotatorPathsConfig{},
			wantContinue:     true,
			wantSrcset:       false,
		},
		{
			name: "piko:img with matching variants annotates srcset",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "hero.jpg"},
				},
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{
				"hero.jpg": {
					{
						SourcePath: "hero.jpg",
						TransformationParams: map[string]string{
							"width":    "400px",
							"_density": "1x",
							"format":   "webp",
						},
					},
				},
			},
			config:       AnnotatorPathsConfig{},
			wantContinue: true,
			wantSrcset:   true,
		},
		{
			name: "piko:img initialises GoAnnotations when nil",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "src", Value: "logo.png"},
				},
				GoAnnotations: nil,
			},
			variantsBySource: map[string][]*annotator_dto.StaticAssetDependency{
				"logo.png": {
					{
						SourcePath: "logo.png",
						TransformationParams: map[string]string{
							"width":    "200",
							"_density": "2x",
							"format":   "webp",
						},
					},
				},
			},
			config:       AnnotatorPathsConfig{},
			wantContinue: true,
			wantSrcset:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := processPikoImgNode(context.Background(), tc.node, tc.variantsBySource, tc.config)
			assert.Equal(t, tc.wantContinue, result)

			if tc.wantSrcset {
				require.NotNil(t, tc.node.GoAnnotations, "GoAnnotations should be initialised")
				assert.NotEmpty(t, tc.node.GoAnnotations.Srcset, "Srcset should be populated")
			} else {
				if tc.node.GoAnnotations != nil {
					assert.Empty(t, tc.node.GoAnnotations.Srcset, "Srcset should not be set")
				}
			}
		})
	}
}

func TestAnnotateSrcsetForWebImages(t *testing.T) {
	t.Parallel()

	t.Run("nil template AST returns early", func(t *testing.T) {
		t.Parallel()

		AnnotateSrcsetForWebImages(context.Background(), nil, []*annotator_dto.StaticAssetDependency{
			{SourcePath: "test.jpg"},
		}, AnnotatorPathsConfig{})
	})

	t.Run("empty dependencies returns early", func(t *testing.T) {
		t.Parallel()
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:img",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "hero.jpg"},
					},
				},
			},
		}
		AnnotateSrcsetForWebImages(context.Background(), ast, nil, AnnotatorPathsConfig{})
		assert.Nil(t, ast.RootNodes[0].GoAnnotations)
	})

	t.Run("annotates matching piko:img nodes", func(t *testing.T) {
		t.Parallel()
		imgNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "hero.jpg"},
			},
		}
		divNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}
		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{divNode, imgNode},
		}

		deps := []*annotator_dto.StaticAssetDependency{
			{
				SourcePath: "hero.jpg",
				AssetType:  "img",
				TransformationParams: map[string]string{
					"width":    "800px",
					"_density": "1x",
					"format":   "webp",
				},
			},
			{
				SourcePath: "hero.jpg",
				AssetType:  "img",
				TransformationParams: map[string]string{
					"width":    "400px",
					"_density": "2x",
					"format":   "webp",
				},
			},
			{

				SourcePath: "styles.css",
				AssetType:  "css",
				TransformationParams: map[string]string{
					"_density": "1x",
				},
			},
			{

				SourcePath: "hero.jpg",
				AssetType:  "img",
				TransformationParams: map[string]string{
					"width": "800px",
				},
			},
		}

		AnnotateSrcsetForWebImages(context.Background(), templateAST, deps, AnnotatorPathsConfig{})

		assert.Nil(t, divNode.GoAnnotations)
		require.NotNil(t, imgNode.GoAnnotations)
		assert.Len(t, imgNode.GoAnnotations.Srcset, 2)

		assert.Equal(t, 400, imgNode.GoAnnotations.Srcset[0].Width)
		assert.Equal(t, 800, imgNode.GoAnnotations.Srcset[1].Width)
	})

	t.Run("skips dependencies without density", func(t *testing.T) {
		t.Parallel()
		imgNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "photo.jpg"},
			},
		}
		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{imgNode},
		}

		deps := []*annotator_dto.StaticAssetDependency{
			{
				SourcePath:           "photo.jpg",
				AssetType:            "img",
				TransformationParams: map[string]string{"width": "400px"},
			},
		}

		AnnotateSrcsetForWebImages(context.Background(), templateAST, deps, AnnotatorPathsConfig{})
		assert.Nil(t, imgNode.GoAnnotations)
	})
}
