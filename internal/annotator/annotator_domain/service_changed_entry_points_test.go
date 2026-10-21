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
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestResolveChangedEntryPoints(t *testing.T) {
	t.Parallel()

	graph := &annotator_dto.ComponentGraph{
		Components: map[string]*annotator_dto.ParsedComponent{
			"pages_index": {}, "pages_about": {}, "pages_blog": {},
		},
		PathToHashedName: map[string]string{
			"/project/pages/index.pk": "pages_index",
			"/project/pages/about.pk": "pages_about",
			"/project/pages/blog.pk":  "pages_blog",
		},
	}
	service := &AnnotatorService{resolver: newMockResolver("my-module", "/project")}

	testCases := []struct {
		existing    map[string]bool
		want        map[string]bool
		name        string
		entryPoints []annotator_dto.EntryPoint
	}{
		{
			name:        "relative and absolute entry points are matched to components",
			entryPoints: []annotator_dto.EntryPoint{{Path: "pages/about.pk"}, {Path: "/project/pages/index.pk"}},
			want:        map[string]bool{"pages_about": true, "pages_index": true},
		},
		{
			name: "a virtual collection page is matched through its template",
			entryPoints: []annotator_dto.EntryPoint{{
				Path:              "my-module/pages/blog/{slug}.pk",
				VirtualPageSource: &annotator_dto.VirtualPageSource{TemplatePath: "/project/pages/blog.pk"},
			}},
			want: map[string]bool{"pages_blog": true},
		},
		{
			name:        "no match annotates every component",
			entryPoints: []annotator_dto.EntryPoint{{Path: "pages/missing.pk"}},
			want:        nil,
		},
		{
			name:        "explicit changed components win",
			entryPoints: []annotator_dto.EntryPoint{{Path: "pages/about.pk"}},
			existing:    map[string]bool{"pages_index": true},
			want:        map[string]bool{"pages_index": true},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			options := &annotationOptions{changedComponents: testCase.existing}
			WithChangedEntryPoints(testCase.entryPoints)(options)

			service.resolveChangedEntryPoints(context.Background(), options, graph)

			assert.Equal(t, testCase.want, options.changedComponents)
		})
	}
}
