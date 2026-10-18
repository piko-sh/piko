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

package seo_adapters

import (
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

const (
	translateDeterminismRuns = 32
)

func TestTranslate_ComponentsInHashOrder(t *testing.T) {
	t.Parallel()

	newPage := func(sourcePath string) *annotator_dto.VirtualComponent {
		return &annotator_dto.VirtualComponent{
			IsPage:   true,
			IsPublic: true,
			Source:   &annotator_dto.ParsedComponent{SourcePath: sourcePath},
		}
	}
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_h": newPage("pages/h.pk"),
				"page_c": newPage("pages/c.pk"),
				"page_f": newPage("pages/f.pk"),
				"page_a": newPage("pages/a.pk"),
				"page_g": newPage("pages/g.pk"),
				"page_b": newPage("pages/b.pk"),
				"page_e": newPage("pages/e.pk"),
				"page_d": newPage("pages/d.pk"),
			},
		},
	}
	want := []string{"page_a", "page_b", "page_c", "page_d", "page_e", "page_f", "page_g", "page_h"}
	translator := NewProjectViewTranslator(nil)

	for range translateDeterminismRuns {
		view := translator.Translate(result)

		hashes := make([]string, 0, len(view.Components))
		for _, component := range view.Components {
			hashes = append(hashes, component.HashedName)
		}
		require.Equal(t, want, hashes, "components must be listed in hashed-name order on every run")
	}
}
