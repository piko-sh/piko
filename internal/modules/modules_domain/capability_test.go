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

package modules_domain

import (
	"reflect"
	"testing"
)

func TestParseCapability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Capability
	}{
		{"axis only", "filesystem.read", Capability{Axis: "filesystem.read"}},
		{"axis and scope", "filesystem.read(/etc/*)", Capability{Axis: "filesystem.read", Scope: "/etc/*"}},
		{"whitespace", "  exec ( ls ) ", Capability{Axis: "exec", Scope: "ls"}},
		{"empty scope parens", "network()", Capability{Axis: "network", Scope: ""}},
		{"unterminated paren", "exec(ls", Capability{Axis: "exec", Scope: "ls"}},
		{"empty input", "", Capability{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseCapability(tt.input); got != tt.expected {
				t.Fatalf("ParseCapability(%q) = %+v, want %+v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCapabilityString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    Capability
		expected string
	}{
		{Capability{Axis: "exec"}, "exec"},
		{Capability{Axis: "filesystem.read", Scope: "/etc/*"}, "filesystem.read(/etc/*)"},
	}
	for _, tt := range tests {
		if got := tt.input.String(); got != tt.expected {
			t.Errorf("Capability.String() = %q, want %q", got, tt.expected)
		}
	}
}

func TestCapabilitySetNormalise(t *testing.T) {
	t.Parallel()
	input := CapabilitySet{
		{Axis: "network"},
		{Axis: "filesystem.read", Scope: "/var/*"},
		{},
		{Axis: "filesystem.read", Scope: "/etc/*"},
		{Axis: "exec", Scope: "ls"},
	}
	got := input.Normalise()
	want := CapabilitySet{
		{Axis: "exec", Scope: "ls"},
		{Axis: "filesystem.read", Scope: "/etc/*"},
		{Axis: "filesystem.read", Scope: "/var/*"},
		{Axis: "network"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalise() = %+v\nwant %+v", got, want)
	}
}

func TestCapabilitySetContainsAndSubset(t *testing.T) {
	t.Parallel()
	full := CapabilitySet{
		{Axis: "filesystem.read", Scope: "/etc/*"},
		{Axis: "network"},
	}
	partial := CapabilitySet{
		{Axis: "network"},
	}
	if !partial.IsSubsetOf(full) {
		t.Fatalf("partial set should be subset of full set")
	}
	if full.IsSubsetOf(partial) {
		t.Fatalf("full set should NOT be subset of partial set")
	}
	if !full.Contains(Capability{Axis: "network"}) {
		t.Fatalf("full set should contain Capability{network}")
	}
	if full.Contains(Capability{Axis: "exec"}) {
		t.Fatalf("full set should not contain Capability{exec}")
	}
}
