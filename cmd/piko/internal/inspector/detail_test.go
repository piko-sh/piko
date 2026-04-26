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
)

func TestDetailBodyWithoutHeader(t *testing.T) {
	t.Parallel()

	body := DetailBody{
		Title:    "Some title",
		Subtitle: "Subtitle text",
		Sections: []DetailSection{
			{
				Heading: "Group A",
				Rows:    []DetailRow{{Label: "Key", Value: "Value"}},
			},
		},
	}

	stripped := body.WithoutHeader()
	if stripped.Title != "" {
		t.Errorf("Title = %q, want empty", stripped.Title)
	}
	if stripped.Subtitle != "" {
		t.Errorf("Subtitle = %q, want empty", stripped.Subtitle)
	}
	if len(stripped.Sections) != 1 {
		t.Fatalf("Sections length = %d, want 1 (sections must be preserved)", len(stripped.Sections))
	}
	if stripped.Sections[0].Heading != "Group A" {
		t.Errorf("Section heading = %q, want %q", stripped.Sections[0].Heading, "Group A")
	}

	if body.Title != "Some title" {
		t.Errorf("Original Title mutated: %q", body.Title)
	}
	if body.Subtitle != "Subtitle text" {
		t.Errorf("Original Subtitle mutated: %q", body.Subtitle)
	}
}
