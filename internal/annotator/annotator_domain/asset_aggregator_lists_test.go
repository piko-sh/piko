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
)

func TestAggregateProjectAssetsSplitsOnlyListParams(t *testing.T) {
	t.Parallel()

	manifest := AggregateProjectAssets([]*annotator_dto.AnnotationResult{{
		AssetDependencies: []*annotator_dto.StaticAssetDependency{{
			SourcePath: "img/hero.jpg",
			AssetType:  "img",
			TransformationParams: map[string]string{
				"background": "rgb(0, 0, 0)",
				"densities":  "x1 x2,x3",
				"format":     "webp, avif",
				"sizes":      "(max-width: 600px) 100vw, 50vw",
			},
		}},
	}})
	require.Len(t, manifest, 1)
	params := manifest[0].TransformationParams

	assert.Equal(t, []string{"rgb(0, 0, 0)"}, params["background"], "a single value keeps its commas")
	assert.Equal(t, []string{"x1", "x2", "x3"}, params["densities"], "densities split on spaces and commas")
	assert.Equal(t, []string{"avif", "webp"}, params["format"])
	assert.Equal(t, []string{"(max-width: 600px) 100vw", "50vw"}, params["sizes"])
}

func TestGenerateVariantKeyUsesTheFirstFormat(t *testing.T) {
	t.Parallel()

	variant := &annotator_dto.StaticAssetDependency{TransformationParams: map[string]string{
		"width": "600", "format": "avif, webp",
	}}
	assert.Equal(t, "image_w600_avif", generateVariantKey(variant))
}
