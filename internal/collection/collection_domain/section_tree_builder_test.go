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

package collection_domain

import (
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

func TestBuildSectionTree(t *testing.T) {
	testCases := []struct {
		checkStructure func(t *testing.T, result []collection_dto.SectionNode)
		name           string
		sections       []markdown_dto.SectionData
		opts           []SectionTreeOption
		wantTopLevel   int
		wantNil        bool
	}{
		{
			name:     "EmptySections",
			sections: nil,
			wantNil:  true,
		},
		{
			name: "SingleH2",
			sections: []markdown_dto.SectionData{
				{Title: "Introduction", Slug: "introduction", Level: 2},
			},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "Introduction" {
					t.Errorf("expected title 'Introduction', got %q", result[0].Title)
				}
				if result[0].Slug != "introduction" {
					t.Errorf("expected slug 'introduction', got %q", result[0].Slug)
				}
				if result[0].Level != 2 {
					t.Errorf("expected level 2, got %d", result[0].Level)
				}
				if len(result[0].Children) != 0 {
					t.Errorf("expected no children, got %d", len(result[0].Children))
				}
			},
		},
		{
			name: "TwoH2Siblings",
			sections: []markdown_dto.SectionData{
				{Title: "First", Slug: "first", Level: 2},
				{Title: "Second", Slug: "second", Level: 2},
			},
			wantTopLevel: 2,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "First" {
					t.Errorf("expected first title 'First', got %q", result[0].Title)
				}
				if result[1].Title != "Second" {
					t.Errorf("expected second title 'Second', got %q", result[1].Title)
				}
			},
		},
		{
			name: "H2WithH3Children",
			sections: []markdown_dto.SectionData{
				{Title: "Parent", Slug: "parent", Level: 2},
				{Title: "Child A", Slug: "child-a", Level: 3},
				{Title: "Child B", Slug: "child-b", Level: 3},
			},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "Parent" {
					t.Errorf("expected parent title 'Parent', got %q", result[0].Title)
				}
				if len(result[0].Children) != 2 {
					t.Fatalf("expected 2 children, got %d", len(result[0].Children))
				}
				if result[0].Children[0].Title != "Child A" {
					t.Errorf("expected first child 'Child A', got %q", result[0].Children[0].Title)
				}
				if result[0].Children[1].Title != "Child B" {
					t.Errorf("expected second child 'Child B', got %q", result[0].Children[1].Title)
				}
			},
		},
		{
			name: "ThreeLevelHierarchy",
			sections: []markdown_dto.SectionData{
				{Title: "H2", Slug: "h2", Level: 2},
				{Title: "H3", Slug: "h3", Level: 3},
				{Title: "H4", Slug: "h4", Level: 4},
			},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if len(result[0].Children) != 1 {
					t.Fatalf("expected 1 child of h2, got %d", len(result[0].Children))
				}
				h3 := result[0].Children[0]
				if h3.Title != "H3" {
					t.Errorf("expected h3 title, got %q", h3.Title)
				}
				if len(h3.Children) != 1 {
					t.Fatalf("expected 1 child of h3, got %d", len(h3.Children))
				}
				if h3.Children[0].Title != "H4" {
					t.Errorf("expected h4 title, got %q", h3.Children[0].Title)
				}
			},
		},
		{
			name: "AllFilteredOut",
			sections: []markdown_dto.SectionData{
				{Title: "H1", Slug: "h1", Level: 1},
				{Title: "H5", Slug: "h5", Level: 5},
			},
			wantNil: true,
		},
		{
			name: "WithMinLevel3",
			sections: []markdown_dto.SectionData{
				{Title: "H2", Slug: "h2", Level: 2},
				{Title: "H3", Slug: "h3", Level: 3},
				{Title: "H4", Slug: "h4", Level: 4},
			},
			opts:         []SectionTreeOption{WithMinLevel(3)},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "H3" {
					t.Errorf("expected root to be H3 (min=3), got %q", result[0].Title)
				}
				if len(result[0].Children) != 1 {
					t.Fatalf("expected 1 child, got %d", len(result[0].Children))
				}
				if result[0].Children[0].Title != "H4" {
					t.Errorf("expected child H4, got %q", result[0].Children[0].Title)
				}
			},
		},
		{
			name: "WithMaxLevel3",
			sections: []markdown_dto.SectionData{
				{Title: "H2", Slug: "h2", Level: 2},
				{Title: "H3", Slug: "h3", Level: 3},
				{Title: "H4", Slug: "h4", Level: 4},
			},
			opts:         []SectionTreeOption{WithMaxLevel(3)},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if len(result[0].Children) != 1 {
					t.Fatalf("expected 1 child (h3), got %d", len(result[0].Children))
				}
				if result[0].Children[0].Title != "H3" {
					t.Errorf("expected child H3, got %q", result[0].Children[0].Title)
				}
				if len(result[0].Children[0].Children) != 0 {
					t.Errorf("expected no grandchildren (h4 filtered), got %d", len(result[0].Children[0].Children))
				}
			},
		},
		{
			name: "WithBothMinAndMaxLevel",
			sections: []markdown_dto.SectionData{
				{Title: "H2", Slug: "h2", Level: 2},
				{Title: "H3", Slug: "h3", Level: 3},
				{Title: "H4", Slug: "h4", Level: 4},
			},
			opts:         []SectionTreeOption{WithMinLevel(3), WithMaxLevel(3)},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "H3" {
					t.Errorf("expected only H3, got %q", result[0].Title)
				}
				if len(result[0].Children) != 0 {
					t.Errorf("expected no children with max=3, got %d", len(result[0].Children))
				}
			},
		},
		{
			name: "OrphanDeeperSection",
			sections: []markdown_dto.SectionData{
				{Title: "H3 Orphan", Slug: "h3-orphan", Level: 3},
				{Title: "H2 After", Slug: "h2-after", Level: 2},
			},
			wantTopLevel: 2,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "H3 Orphan" {
					t.Errorf("expected first node 'H3 Orphan', got %q", result[0].Title)
				}
				if result[1].Title != "H2 After" {
					t.Errorf("expected second node 'H2 After', got %q", result[1].Title)
				}
			},
		},
		{
			name: "H2ThenH4DirectlySkipH3",
			sections: []markdown_dto.SectionData{
				{Title: "H2", Slug: "h2", Level: 2},
				{Title: "H4 Deep", Slug: "h4-deep", Level: 4},
			},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if len(result[0].Children) != 1 {
					t.Fatalf("expected 1 child (h4 attached directly), got %d", len(result[0].Children))
				}
				if result[0].Children[0].Title != "H4 Deep" {
					t.Errorf("expected child 'H4 Deep', got %q", result[0].Children[0].Title)
				}
			},
		},
		{
			name: "ComplexRealisticToC",
			sections: []markdown_dto.SectionData{
				{Title: "Getting Started", Slug: "getting-started", Level: 2},
				{Title: "Installation", Slug: "installation", Level: 3},
				{Title: "Configuration", Slug: "configuration", Level: 3},
				{Title: "Advanced Options", Slug: "advanced-options", Level: 4},
				{Title: "API Reference", Slug: "api-reference", Level: 2},
				{Title: "Endpoints", Slug: "endpoints", Level: 3},
			},
			wantTopLevel: 2,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				gs := result[0]
				if gs.Title != "Getting Started" {
					t.Errorf("expected 'Getting Started', got %q", gs.Title)
				}
				if len(gs.Children) != 2 {
					t.Fatalf("expected 2 children under Getting Started, got %d", len(gs.Children))
				}
				config := gs.Children[1]
				if config.Title != "Configuration" {
					t.Errorf("expected 'Configuration', got %q", config.Title)
				}
				if len(config.Children) != 1 {
					t.Fatalf("expected 1 child under Configuration, got %d", len(config.Children))
				}
				if config.Children[0].Title != "Advanced Options" {
					t.Errorf("expected 'Advanced Options', got %q", config.Children[0].Title)
				}

				api := result[1]
				if api.Title != "API Reference" {
					t.Errorf("expected 'API Reference', got %q", api.Title)
				}
				if len(api.Children) != 1 {
					t.Fatalf("expected 1 child under API Reference, got %d", len(api.Children))
				}
			},
		},
		{
			name: "H1FilteredByDefault",
			sections: []markdown_dto.SectionData{
				{Title: "Page Title", Slug: "page-title", Level: 1},
				{Title: "Content", Slug: "content", Level: 2},
			},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if result[0].Title != "Content" {
					t.Errorf("expected 'Content' (h1 filtered), got %q", result[0].Title)
				}
			},
		},
		{
			name: "H5FilteredByDefault",
			sections: []markdown_dto.SectionData{
				{Title: "H2", Slug: "h2", Level: 2},
				{Title: "H5 Deep", Slug: "h5-deep", Level: 5},
			},
			wantTopLevel: 1,
			checkStructure: func(t *testing.T, result []collection_dto.SectionNode) {
				t.Helper()
				if len(result[0].Children) != 0 {
					t.Errorf("expected no children (h5 filtered by default max=4), got %d", len(result[0].Children))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildSectionTree(tc.sections, tc.opts...)

			if tc.wantNil {
				if result != nil {
					t.Fatalf("expected nil result, got %d nodes", len(result))
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if len(result) != tc.wantTopLevel {
				t.Fatalf("expected %d top-level nodes, got %d", tc.wantTopLevel, len(result))
			}

			if tc.checkStructure != nil {
				tc.checkStructure(t, result)
			}
		})
	}
}
