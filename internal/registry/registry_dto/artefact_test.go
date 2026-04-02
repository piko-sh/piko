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

package registry_dto

import "testing"

func TestArtefactMeta_GetProfile(t *testing.T) {
	meta := &ArtefactMeta{
		DesiredProfiles: []NamedProfile{
			{Name: "thumb", Profile: DesiredProfile{CapabilityName: "resize"}},
			{Name: "webp", Profile: DesiredProfile{CapabilityName: "convert"}},
		},
	}

	p, ok := meta.GetProfile("thumb")
	if !ok || p.CapabilityName != "resize" {
		t.Errorf("GetProfile(thumb) = %v, %v; want resize, true", p, ok)
	}

	_, ok = meta.GetProfile("nonexistent")
	if ok {
		t.Error("GetProfile(nonexistent) should return false")
	}
}

func TestArtefactMeta_SetProfile_New(t *testing.T) {
	meta := &ArtefactMeta{}
	meta.SetProfile("thumb", &DesiredProfile{CapabilityName: "resize"})

	if len(meta.DesiredProfiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(meta.DesiredProfiles))
	}
	if meta.DesiredProfiles[0].Profile.CapabilityName != "resize" {
		t.Error("profile not stored correctly")
	}
}

func TestArtefactMeta_SetProfile_Update(t *testing.T) {
	meta := &ArtefactMeta{
		DesiredProfiles: []NamedProfile{
			{Name: "thumb", Profile: DesiredProfile{CapabilityName: "resize"}},
		},
	}
	meta.SetProfile("thumb", &DesiredProfile{CapabilityName: "updated"})

	if len(meta.DesiredProfiles) != 1 {
		t.Fatalf("expected 1 profile after update, got %d", len(meta.DesiredProfiles))
	}
	if meta.DesiredProfiles[0].Profile.CapabilityName != "updated" {
		t.Error("profile not updated")
	}
}

func TestArtefactMeta_HasProfile(t *testing.T) {
	meta := &ArtefactMeta{
		DesiredProfiles: []NamedProfile{
			{Name: "thumb"},
		},
	}
	if !meta.HasProfile("thumb") {
		t.Error("HasProfile(thumb) should be true")
	}
	if meta.HasProfile("missing") {
		t.Error("HasProfile(missing) should be false")
	}
}

func TestArtefactMeta_ComputeStatus(t *testing.T) {
	tests := []struct {
		name     string
		want     VariantStatus
		variants []Variant
	}{
		{name: "no variants", want: VariantStatusPending, variants: nil},
		{name: "one ready", want: VariantStatusReady, variants: []Variant{{Status: VariantStatusReady}}},
		{name: "mixed with ready", want: VariantStatusReady, variants: []Variant{
			{Status: VariantStatusStale},
			{Status: VariantStatusReady},
		}},
		{name: "all stale", want: VariantStatusStale, variants: []Variant{
			{Status: VariantStatusStale},
			{Status: VariantStatusPending},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &ArtefactMeta{ActualVariants: tt.variants}
			if got := meta.ComputeStatus(); got != tt.want {
				t.Errorf("ComputeStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
