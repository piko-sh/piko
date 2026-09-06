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

package tui_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/piko/internal/inspector"
)

func TestResourceDetailBody(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		resource       Resource
		wantResource   []string
		wantMetadata   []string
		wantSections   int
		wantChildCount string
	}{
		{
			name: "resource without metadata or children",
			resource: Resource{
				Kind:       "artefact",
				ID:         "a-1",
				Name:       "logo.png",
				StatusText: "ready",
			},
			wantResource: []string{"Kind", "ID", "Name", "Status", "Created", "Updated"},
			wantSections: 1,
		},
		{
			name: "children are counted",
			resource: Resource{
				Kind:       "artefact",
				ID:         "a-2",
				Name:       "hero.jpg",
				StatusText: "ready",
				Children:   []Resource{{ID: "v-1"}, {ID: "v-2"}},
			},
			wantResource:   []string{"Kind", "ID", "Name", "Status", "Created", "Updated", "Children"},
			wantSections:   1,
			wantChildCount: "2",
		},
		{
			name: "metadata is sorted by label",
			resource: Resource{
				Kind:       "artefact",
				ID:         "a-3",
				Name:       "banner.webp",
				StatusText: "pending",
				Metadata:   map[string]string{"width": "640", "format": "webp", "height": "480"},
			},
			wantResource: []string{"Kind", "ID", "Name", "Status", "Created", "Updated"},
			wantMetadata: []string{"format", "height", "width"},
			wantSections: 2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			body := resourceDetailBody(&testCase.resource)

			assert.Equal(t, testCase.resource.Name, body.Title)
			assert.Equal(t, testCase.resource.Kind+" · "+testCase.resource.StatusText, body.Subtitle)
			require.Len(t, body.Sections, testCase.wantSections)
			assert.Equal(t, testCase.wantResource, detailLabels(body.Sections[0]))
			if testCase.wantChildCount != "" {
				rows := body.Sections[0].Rows
				assert.Equal(t, testCase.wantChildCount, rows[len(rows)-1].Value)
			}
			if testCase.wantMetadata != nil {
				assert.Equal(t, "Metadata", body.Sections[1].Heading)
				assert.Equal(t, testCase.wantMetadata, detailLabels(body.Sections[1]))
			}
		})
	}
}

func TestKindDetailBody(t *testing.T) {
	t.Parallel()

	body := kindDetailBody("artefact", map[ResourceStatus]int{
		ResourceStatusHealthy:   3,
		ResourceStatusDegraded:  1,
		ResourceStatusUnhealthy: 2,
		ResourceStatusPending:   4,
	})

	assert.Equal(t, "artefact", body.Title)
	assert.Equal(t, "10 total", body.Subtitle)
	require.Len(t, body.Sections, 1)
	assert.Equal(t, []string{"Total", "Healthy", "Degraded", "Unhealthy", "Pending", "Unknown"}, detailLabels(body.Sections[0]))
	assert.Equal(t, "10", body.Sections[0].Rows[0].Value)
}

func TestRegistryPanelOverviewDetailBody(t *testing.T) {
	t.Parallel()

	panel := NewRegistryPanel()
	panel.SetSummary(map[string]map[ResourceStatus]int{
		"artefact": {ResourceStatusHealthy: 2, ResourceStatusPending: 1},
		"task":     {ResourceStatusUnhealthy: 4},
	})

	body := panel.overviewDetailBody()

	assert.Equal(t, "Registry overview", body.Title)
	assert.Equal(t, "2 kinds", body.Subtitle)
	require.Len(t, body.Sections, 1)
	assert.Equal(t, []string{"artefact", "task"}, detailLabels(body.Sections[0]))
	assert.Equal(t, "3", body.Sections[0].Rows[0].Value)
	assert.Equal(t, "4", body.Sections[0].Rows[1].Value)
}

func TestRegistryPanelDetailView(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setup    func(panel *RegistryPanel)
		name     string
		contains string
	}{
		{
			name:     "overview when nothing is listed",
			setup:    func(*RegistryPanel) {},
			contains: "Registry overview",
		},
		{
			name: "kind breakdown under the cursor",
			setup: func(panel *RegistryPanel) {
				panel.SetSummary(map[string]map[ResourceStatus]int{"artefact": {ResourceStatusHealthy: 2}})
			},
			contains: "STATUS BREAKDOWN",
		},
		{
			name: "resource under the cursor",
			setup: func(panel *RegistryPanel) {
				panel.SetSummary(map[string]map[ResourceStatus]int{"artefact": {ResourceStatusHealthy: 1}})
				panel.selectedKind = "artefact"
				panel.SetResources([]Resource{{
					Kind:       "artefact",
					ID:         "a-1",
					Name:       "logo.png",
					StatusText: "ready",
					Metadata:   map[string]string{"format": "png"},
				}})
				panel.SetCursor(1)
			},
			contains: "logo.png",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			panel := NewRegistryPanel()
			testCase.setup(panel)

			assert.Contains(t, panel.DetailView(80, 24), testCase.contains)
		})
	}
}

func detailLabels(section inspector.DetailSection) []string {
	labels := make([]string, 0, len(section.Rows))
	for _, row := range section.Rows {
		labels = append(labels, row.Label)
	}
	return labels
}
