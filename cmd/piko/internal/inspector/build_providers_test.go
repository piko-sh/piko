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

func TestBuildProviderDetailSectionsNil(t *testing.T) {
	t.Parallel()

	got := BuildProviderDetailSections(nil)
	if len(got) != 0 {
		t.Errorf("got %d sections, want 0", len(got))
	}
	if got == nil {
		t.Errorf("got nil slice, want allocated empty slice")
	}
}

func TestBuildProviderDetailSectionsEmpty(t *testing.T) {
	t.Parallel()

	response := &pb.DescribeProviderResponse{}
	got := BuildProviderDetailSections(response)
	if len(got) != 0 {
		t.Errorf("got %d sections, want 0", len(got))
	}
}

func TestBuildProviderDetailSectionsEmptyEntries(t *testing.T) {
	t.Parallel()

	response := &pb.DescribeProviderResponse{
		Sections: []*pb.ProviderInfoSection{
			{Title: "Section A"},
			{Title: "Section B", Entries: []*pb.ProviderInfoEntry{}},
		},
	}
	got := BuildProviderDetailSections(response)
	if len(got) != 2 {
		t.Fatalf("got %d sections, want 2", len(got))
	}
	for index, section := range got {
		if len(section.Rows) != 0 {
			t.Errorf("section %d rows = %d, want 0", index, len(section.Rows))
		}
	}
	if got[0].Heading != "Section A" {
		t.Errorf("got[0].Heading = %q, want Section A", got[0].Heading)
	}
	if got[1].Heading != "Section B" {
		t.Errorf("got[1].Heading = %q, want Section B", got[1].Heading)
	}
}

func TestBuildProviderDetailSectionsPopulated(t *testing.T) {
	t.Parallel()

	response := &pb.DescribeProviderResponse{
		Sections: []*pb.ProviderInfoSection{
			{
				Title: "Overview",
				Entries: []*pb.ProviderInfoEntry{
					{Key: "Name", Value: "otter"},
					{Key: "Type", Value: "in-memory"},
				},
			},
			{
				Title: "Stats",
				Entries: []*pb.ProviderInfoEntry{
					{Key: "Hits", Value: "1234"},
					{Key: "Misses", Value: "56"},
					{Key: "Evictions", Value: "0"},
				},
			},
		},
	}

	sections := BuildProviderDetailSections(response)
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}

	if sections[0].Heading != "Overview" {
		t.Errorf("sections[0].Heading = %q, want Overview", sections[0].Heading)
	}
	if len(sections[0].Rows) != 2 {
		t.Fatalf("Overview rows = %d, want 2", len(sections[0].Rows))
	}
	if sections[0].Rows[0].Label != "Name" || sections[0].Rows[0].Value != "otter" {
		t.Errorf("first row = %+v, want {Name otter}", sections[0].Rows[0])
	}
	if sections[0].Rows[1].Label != "Type" || sections[0].Rows[1].Value != "in-memory" {
		t.Errorf("second row = %+v, want {Type in-memory}", sections[0].Rows[1])
	}

	if sections[1].Heading != "Stats" {
		t.Errorf("sections[1].Heading = %q, want Stats", sections[1].Heading)
	}
	if len(sections[1].Rows) != 3 {
		t.Fatalf("Stats rows = %d, want 3", len(sections[1].Rows))
	}
}

func TestBuildProviderDetailSectionsRowOrder(t *testing.T) {
	t.Parallel()

	response := &pb.DescribeProviderResponse{
		Sections: []*pb.ProviderInfoSection{
			{
				Title: "Ordered",
				Entries: []*pb.ProviderInfoEntry{
					{Key: "z", Value: "1"},
					{Key: "a", Value: "2"},
					{Key: "m", Value: "3"},
				},
			},
		},
	}

	sections := BuildProviderDetailSections(response)
	wantKeys := []string{"z", "a", "m"}
	for index, want := range wantKeys {
		if sections[0].Rows[index].Label != want {
			t.Errorf("row %d Label = %q, want %q (entry order must be preserved)", index, sections[0].Rows[index].Label, want)
		}
	}
}
