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

import "testing"

func TestGlobalKeyBindings(t *testing.T) {
	bindings := GlobalKeyBindings()

	if len(bindings) == 0 {
		t.Fatal("expected non-empty keybindings")
	}

	expectedCount := 14
	if len(bindings) != expectedCount {
		t.Errorf("expected %d bindings, got %d", expectedCount, len(bindings))
	}

	for i, kb := range bindings {
		if kb.Key == "" {
			t.Errorf("binding %d: Key is empty", i)
		}
		if kb.Description == "" {
			t.Errorf("binding %d: Description is empty", i)
		}
	}
}

func TestNavigationKeyBindings(t *testing.T) {
	bindings := NavigationKeyBindings()

	if len(bindings) == 0 {
		t.Fatal("expected non-empty keybindings")
	}

	expectedCount := 7
	if len(bindings) != expectedCount {
		t.Errorf("expected %d bindings, got %d", expectedCount, len(bindings))
	}

	for i, kb := range bindings {
		if kb.Key == "" {
			t.Errorf("binding %d: Key is empty", i)
		}
		if kb.Description == "" {
			t.Errorf("binding %d: Description is empty", i)
		}
	}
}

func TestTableKeyBindings(t *testing.T) {
	bindings := TableKeyBindings()

	if len(bindings) == 0 {
		t.Fatal("expected non-empty keybindings")
	}

	expectedCount := 8
	if len(bindings) != expectedCount {
		t.Errorf("expected %d bindings, got %d", expectedCount, len(bindings))
	}

	for i, kb := range bindings {
		if kb.Key == "" {
			t.Errorf("binding %d: Key is empty", i)
		}
		if kb.Description == "" {
			t.Errorf("binding %d: Description is empty", i)
		}
	}
}

func TestMetricsPanelKeyBindings(t *testing.T) {
	bindings := MetricsPanelKeyBindings()

	if len(bindings) == 0 {
		t.Fatal("expected non-empty keybindings")
	}

	expectedCount := 3
	if len(bindings) != expectedCount {
		t.Errorf("expected %d bindings, got %d", expectedCount, len(bindings))
	}

	for i, kb := range bindings {
		if kb.Key == "" {
			t.Errorf("binding %d: Key is empty", i)
		}
		if kb.Description == "" {
			t.Errorf("binding %d: Description is empty", i)
		}
	}
}

func TestTracesPanelKeyBindings(t *testing.T) {
	bindings := TracesPanelKeyBindings()

	if len(bindings) == 0 {
		t.Fatal("expected non-empty keybindings")
	}

	expectedCount := 3
	if len(bindings) != expectedCount {
		t.Errorf("expected %d bindings, got %d", expectedCount, len(bindings))
	}

	for i, kb := range bindings {
		if kb.Key == "" {
			t.Errorf("binding %d: Key is empty", i)
		}
		if kb.Description == "" {
			t.Errorf("binding %d: Description is empty", i)
		}
	}
}

func TestResourcesPanelKeyBindings(t *testing.T) {
	bindings := ResourcesPanelKeyBindings()

	if len(bindings) == 0 {
		t.Fatal("expected non-empty keybindings")
	}

	expectedCount := 2
	if len(bindings) != expectedCount {
		t.Errorf("expected %d bindings, got %d", expectedCount, len(bindings))
	}

	for i, kb := range bindings {
		if kb.Key == "" {
			t.Errorf("binding %d: Key is empty", i)
		}
		if kb.Description == "" {
			t.Errorf("binding %d: Description is empty", i)
		}
	}
}

func TestKeyBindingGenerators_AllReturnValidBindings(t *testing.T) {
	testCases := []struct {
		generator   func() []KeyBinding
		name        string
		minExpected int
	}{
		{name: "GlobalKeyBindings", generator: GlobalKeyBindings, minExpected: 5},
		{name: "NavigationKeyBindings", generator: NavigationKeyBindings, minExpected: 5},
		{name: "TableKeyBindings", generator: TableKeyBindings, minExpected: 5},
		{name: "MetricsPanelKeyBindings", generator: MetricsPanelKeyBindings, minExpected: 1},
		{name: "TracesPanelKeyBindings", generator: TracesPanelKeyBindings, minExpected: 1},
		{name: "ResourcesPanelKeyBindings", generator: ResourcesPanelKeyBindings, minExpected: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bindings := tc.generator()

			if len(bindings) < tc.minExpected {
				t.Errorf("expected at least %d bindings, got %d", tc.minExpected, len(bindings))
			}

			for i, kb := range bindings {
				if kb.Key == "" {
					t.Errorf("binding %d: Key must not be empty", i)
				}
				if kb.Description == "" {
					t.Errorf("binding %d: Description must not be empty", i)
				}
			}
		})
	}
}
