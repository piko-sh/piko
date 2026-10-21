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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func expandedImage() *annotator_dto.FinalAssetDependency {
	return &annotator_dto.FinalAssetDependency{
		SourcePath: "lib/hero.jpg",
		AssetType:  "img",
		TransformationParams: map[string][]string{
			"width":              {"600", "1200"},
			paramExpandedDensity: {"x1", "x2"},
			"format":             {"avif", "webp"},
			"quality":            {"70"},
		},
	}
}

func pageWithSrcset(src string, keys ...string) map[string]*annotator_dto.AnnotationResult {
	srcset := make([]ast_domain.ResponsiveVariantMetadata, 0, len(keys))
	for _, key := range keys {
		srcset = append(srcset, ast_domain.ResponsiveVariantMetadata{VariantKey: key})
	}
	img := &ast_domain.TemplateNode{
		NodeType:      ast_domain.NodeElement,
		TagName:       "piko:img",
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{Srcset: srcset},
	}
	img.SetAttribute("src", src)
	return map[string]*annotator_dto.AnnotationResult{
		"page": {AnnotatedAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{img}}},
	}
}

func TestProcessBuildResultBuildsExactlyTheSrcsetVariants(t *testing.T) {
	t.Parallel()

	registry, calls := newPipelineRegistryMock(nil)
	pipeline := NewAssetPipelineOrchestrator(registry, nil, nil, nil)
	require.NoError(t, pipeline.ProcessBuildResult(t.Context(), &annotator_dto.ProjectAnnotationResult{
		ComponentResults:   pageWithSrcset("lib/hero.jpg", "image_w600_webp", "image_w1200_webp"),
		FinalAssetManifest: []*annotator_dto.FinalAssetDependency{expandedImage()},
	}))

	require.Len(t, *calls, 1)
	names := make([]string, 0)
	for _, profile := range (*calls)[0].profiles {
		names = append(names, profile.Name)
		mimeType, _ := profile.Profile.ResultingTags.GetByName("mimeType")
		assert.Equal(t, "image/webp", mimeType, "every variant records its MIME type")
		quality, _ := profile.Profile.Params.GetByName("quality")
		assert.Equal(t, "70", quality, "the image's own settings apply")
		_, leaked := profile.Profile.Params.GetByName(paramExpandedDensity)
		assert.False(t, leaked, "the annotator's marker is not a transform parameter")
	}
	assert.ElementsMatch(t, []string{"image_w600_webp", "image_w1200_webp"}, names,
		"exactly the variants the srcset names are built; no format the page never requests")
}

func TestCalculateRequiredWidthsAcceptsSpaceSeparatedDensities(t *testing.T) {
	t.Parallel()

	widths := (&AssetPipelineOrchestrator{}).calculateRequiredWidths([]string{"400px"}, []string{"x1 x2"})
	assert.Equal(t, []int{400, 800}, widths)
}

func TestParseVariantKey(t *testing.T) {
	t.Parallel()

	width, format, ok := parseVariantKey("image_w1200_webp")
	require.True(t, ok)
	assert.Equal(t, 1200, width)
	assert.Equal(t, "webp", format)

	for _, key := range []string{"placeholder", "image_w_webp", "image_wabc_webp", "image_w600_", "video_720p"} {
		_, _, ok := parseVariantKey(key)
		assert.False(t, ok, "%q is not an image variant key", key)
	}
}

func TestVerifySrcsetProfilesFailsOnAReferencedVariantNotProduced(t *testing.T) {
	t.Parallel()

	components := pageWithSrcset("lib/hero.jpg", "image_w600_webp", "image_w1200_webp")
	require.NoError(t, verifySrcsetProfiles(components, map[string]map[string]struct{}{
		"lib/hero.jpg": {"image_w600_webp": {}, "image_w1200_webp": {}},
	}), "a srcset whose variants are all produced passes")

	err := verifySrcsetProfiles(components, map[string]map[string]struct{}{
		"lib/hero.jpg": {"image_w600_webp": {}},
	})
	require.ErrorIs(t, err, ErrMissingSrcsetProfiles)
	assert.Contains(t, err.Error(), "lib/hero.jpg?v=image_w1200_webp")
}

func TestImageVariantsOnDemandLeavesSrcsetVariantsToTheServer(t *testing.T) {
	t.Parallel()

	registry, calls := newPipelineRegistryMock(nil)
	pipeline := NewAssetPipelineOrchestrator(registry, nil, nil, nil, WithImageVariantsOnDemand())
	err := pipeline.ProcessBuildResult(t.Context(), &annotator_dto.ProjectAnnotationResult{
		ComponentResults:   pageWithSrcset("lib/hero.jpg", "image_w600_webp"),
		FinalAssetManifest: []*annotator_dto.FinalAssetDependency{expandedImage()},
	})
	require.NoError(t, err, "variants generated on first request are not missing")
	assert.Empty(t, *calls, "a build that cannot transform images builds no srcset variants")
}
