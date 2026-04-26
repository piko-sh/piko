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

package inspector

import (
	"testing"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestBuildHealthDetailSectionsNil(t *testing.T) {
	t.Parallel()

	got := BuildHealthDetailSections(nil, "")
	if len(got) != 0 {
		t.Errorf("got %d sections, want 0 (both probes nil)", len(got))
	}
	if got == nil {
		t.Errorf("got nil slice, want allocated empty slice")
	}
}

func TestBuildHealthDetailSectionsLivenessOnly(t *testing.T) {
	t.Parallel()

	response := &pb.GetHealthResponse{
		Liveness: &pb.HealthStatus{State: "HEALTHY", Message: "OK"},
	}
	sections := BuildHealthDetailSections(response, "")
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1 (readiness nil should be skipped)", len(sections))
	}
	if sections[0].Heading != "Liveness" {
		t.Errorf("Heading = %q, want Liveness", sections[0].Heading)
	}
}

func TestBuildHealthDetailSectionsReadinessOnly(t *testing.T) {
	t.Parallel()

	response := &pb.GetHealthResponse{
		Readiness: &pb.HealthStatus{State: "DEGRADED"},
	}
	sections := BuildHealthDetailSections(response, "")
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}
	if sections[0].Heading != "Readiness" {
		t.Errorf("Heading = %q, want Readiness", sections[0].Heading)
	}
}

func TestBuildHealthDetailSectionsBoth(t *testing.T) {
	t.Parallel()

	response := &pb.GetHealthResponse{
		Liveness: &pb.HealthStatus{
			Name:    "Liveness",
			State:   "HEALTHY",
			Message: "All good",
			Dependencies: []*pb.HealthStatus{
				{Name: "Database", State: "HEALTHY", Duration: "0.5ms"},
				{Name: "Cache", State: "HEALTHY", Duration: "0.3ms"},
			},
		},
		Readiness: &pb.HealthStatus{
			Name:    "Readiness",
			State:   "DEGRADED",
			Message: "Issues found",
			Dependencies: []*pb.HealthStatus{
				{Name: "Database", State: "HEALTHY"},
				{Name: "Queue", State: "UNHEALTHY", Message: "timeout"},
			},
		},
	}

	testCases := []struct {
		name         string
		filter       string
		wantTitle    string
		wantSections int
		wantSubCount int
	}{
		{name: "no filter returns both", filter: "", wantSections: 2},
		{name: "filter Liveness", filter: "Liveness", wantSections: 1, wantTitle: "Liveness", wantSubCount: 2},
		{name: "filter Readiness", filter: "Readiness", wantSections: 1, wantTitle: "Readiness", wantSubCount: 2},
		{name: "no match", filter: "nonexistent", wantSections: 0},
		{name: "case insensitive", filter: "liveness", wantSections: 1, wantTitle: "Liveness"},
		{name: "prefix match", filter: "live", wantSections: 1, wantTitle: "Liveness"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sections := BuildHealthDetailSections(response, tc.filter)
			if len(sections) != tc.wantSections {
				t.Errorf("got %d sections, want %d", len(sections), tc.wantSections)
			}
			if tc.wantTitle != "" && len(sections) > 0 && sections[0].Heading != tc.wantTitle {
				t.Errorf("first section title = %q, want %q", sections[0].Heading, tc.wantTitle)
			}
			if tc.wantSubCount > 0 && len(sections) > 0 && len(sections[0].SubSections) != tc.wantSubCount {
				t.Errorf("sub-sections = %d, want %d", len(sections[0].SubSections), tc.wantSubCount)
			}
		})
	}
}

func TestHealthStatusSectionRows(t *testing.T) {
	t.Parallel()

	status := &pb.HealthStatus{
		State:       "HEALTHY",
		Message:     "All good",
		Duration:    "1ms",
		TimestampMs: 0,
		Dependencies: []*pb.HealthStatus{
			{Name: "DB", State: "HEALTHY"},
			{Name: "Cache", State: "UNHEALTHY"},
		},
	}

	section := healthStatusSection("Liveness", status)

	if section.Heading != "Liveness" {
		t.Errorf("Heading = %q, want Liveness", section.Heading)
	}

	rowsByLabel := map[string]DetailRow{}
	for _, row := range section.Rows {
		rowsByLabel[row.Label] = row
	}

	state, ok := rowsByLabel["State"]
	if !ok {
		t.Fatalf("State row missing")
	}
	if !state.IsStatus {
		t.Errorf("State.IsStatus = false, want true")
	}
	if state.Value != "HEALTHY" {
		t.Errorf("State.Value = %q, want HEALTHY", state.Value)
	}

	if rowsByLabel["Message"].Value != "All good" {
		t.Errorf("Message = %q, want All good", rowsByLabel["Message"].Value)
	}
	if rowsByLabel["Duration"].Value != "1ms" {
		t.Errorf("Duration = %q, want 1ms", rowsByLabel["Duration"].Value)
	}
	if rowsByLabel["Dependencies"].Value != "1/2" {
		t.Errorf("Dependencies = %q, want 1/2", rowsByLabel["Dependencies"].Value)
	}

	if len(section.SubSections) != 2 {
		t.Errorf("SubSections = %d, want 2", len(section.SubSections))
	}
}

func TestHealthDependencySection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		dependency *pb.HealthStatus
		name       string
		wantRows   []string
	}{
		{
			name:       "no message",
			dependency: &pb.HealthStatus{Name: "Database", State: "HEALTHY", Duration: "0.5ms"},
			wantRows:   []string{"State", "Duration"},
		},
		{
			name:       "with message",
			dependency: &pb.HealthStatus{Name: "Queue", State: "UNHEALTHY", Duration: "10s", Message: "timeout"},
			wantRows:   []string{"State", "Duration", "Message"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			section := healthDependencySection(tc.dependency)
			if section.Heading != tc.dependency.GetName() {
				t.Errorf("Heading = %q, want %q", section.Heading, tc.dependency.GetName())
			}
			if len(section.Rows) != len(tc.wantRows) {
				t.Fatalf("rows = %d, want %d", len(section.Rows), len(tc.wantRows))
			}
			for index, label := range tc.wantRows {
				if section.Rows[index].Label != label {
					t.Errorf("row %d label = %q, want %q", index, section.Rows[index].Label, label)
				}
			}

			if !section.Rows[0].IsStatus {
				t.Errorf("State row IsStatus = false, want true")
			}
		})
	}
}

func TestFormatHealthReady(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status *pb.HealthStatus
		name   string
		want   string
	}{
		{name: "no dependencies", status: &pb.HealthStatus{}, want: "-"},
		{
			name: "all healthy",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "healthy"},
					{State: "HEALTHY"},
				},
			},
			want: "2/2",
		},
		{
			name: "mixed",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "healthy"},
					{State: "degraded"},
					{State: "unhealthy"},
				},
			},
			want: "1/3",
		},
		{
			name: "none healthy",
			status: &pb.HealthStatus{
				Dependencies: []*pb.HealthStatus{
					{State: "unhealthy"},
				},
			},
			want: "0/1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatHealthReady(tc.status)
			if got != tc.want {
				t.Errorf("formatHealthReady() = %q, want %q", got, tc.want)
			}
		})
	}
}
