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

func TestBuildDLQDetailSectionsNilEmpty(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		summaries []*pb.DispatcherSummary
		name      string
	}{
		{name: "nil summaries", summaries: nil},
		{name: "empty summaries", summaries: []*pb.DispatcherSummary{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := BuildDLQDetailSections(tc.summaries, "")
			if len(got) != 0 {
				t.Errorf("got %d sections, want 0", len(got))
			}

			if got == nil {
				t.Errorf("got nil slice, want non-nil empty slice")
			}
		})
	}
}

func TestBuildDLQDetailSectionsPopulated(t *testing.T) {
	t.Parallel()

	summaries := []*pb.DispatcherSummary{
		{
			Type:            "email",
			QueuedItems:     5,
			RetryQueueSize:  2,
			DeadLetterCount: 3,
			TotalProcessed:  100,
			TotalSuccessful: 90,
			TotalFailed:     10,
			TotalRetries:    7,
			UptimeMs:        123_456,
		},
	}

	sections := BuildDLQDetailSections(summaries, "")
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}
	section := sections[0]
	if section.Heading != "Dispatcher email" {
		t.Errorf("Heading = %q, want %q", section.Heading, "Dispatcher email")
	}

	want := map[string]string{
		"Type":             "email",
		"Queued":           "5",
		"Retry Queue":      "2",
		"Dead Letter":      "3",
		"Total Processed":  "100",
		"Total Successful": "90",
		"Total Failed":     "10",
		"Total Retries":    "7",
	}

	got := map[string]string{}
	for _, row := range section.Rows {
		got[row.Label] = row.Value
	}

	for label, value := range want {
		if got[label] != value {
			t.Errorf("row %q = %q, want %q", label, got[label], value)
		}
	}

	uptime, ok := got["Uptime"]
	if !ok {
		t.Errorf("Uptime row missing")
	}
	if uptime == "" {
		t.Errorf("Uptime row value empty")
	}
}

func TestBuildDLQDetailSectionsFilter(t *testing.T) {
	t.Parallel()

	summaries := []*pb.DispatcherSummary{
		{Type: "email", QueuedItems: 5},
		{Type: "sms", QueuedItems: 1},
		{Type: "webhook", QueuedItems: 3},
	}

	testCases := []struct {
		name         string
		filter       string
		wantHeadings []string
	}{
		{name: "no filter passes all", filter: "", wantHeadings: []string{"Dispatcher email", "Dispatcher sms", "Dispatcher webhook"}},
		{name: "exact match email", filter: "email", wantHeadings: []string{"Dispatcher email"}},
		{name: "case insensitive prefix", filter: "EMAIL", wantHeadings: []string{"Dispatcher email"}},
		{name: "no match", filter: "nope", wantHeadings: []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sections := BuildDLQDetailSections(summaries, tc.filter)
			if len(sections) != len(tc.wantHeadings) {
				t.Fatalf("got %d sections, want %d", len(sections), len(tc.wantHeadings))
			}
			for index, section := range sections {
				if section.Heading != tc.wantHeadings[index] {
					t.Errorf("section %d heading = %q, want %q", index, section.Heading, tc.wantHeadings[index])
				}
			}
		})
	}
}

func TestBuildDLQDetailSectionsRowOrder(t *testing.T) {
	t.Parallel()

	summaries := []*pb.DispatcherSummary{{Type: "x"}}
	sections := BuildDLQDetailSections(summaries, "")
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}

	wantOrder := []string{
		"Type",
		"Queued",
		"Retry Queue",
		"Dead Letter",
		"Total Processed",
		"Total Successful",
		"Total Failed",
		"Total Retries",
		"Uptime",
	}

	if len(sections[0].Rows) != len(wantOrder) {
		t.Fatalf("got %d rows, want %d", len(sections[0].Rows), len(wantOrder))
	}
	for index, label := range wantOrder {
		if sections[0].Rows[index].Label != label {
			t.Errorf("row %d label = %q, want %q", index, sections[0].Rows[index].Label, label)
		}
	}
}
