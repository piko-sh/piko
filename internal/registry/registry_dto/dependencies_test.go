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

import (
	"slices"
	"testing"
)

func TestDependencies_AddAndGet(t *testing.T) {
	var d Dependencies

	d.Add("a")
	d.Add("b")
	d.Add("c")

	if d.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", d.Len())
	}
	if got := d.Get(0); got != "a" {
		t.Errorf("Get(0) = %q, want %q", got, "a")
	}
	if got := d.Get(1); got != "b" {
		t.Errorf("Get(1) = %q, want %q", got, "b")
	}
	if got := d.Get(2); got != "c" {
		t.Errorf("Get(2) = %q, want %q", got, "c")
	}
}

func TestDependencies_IsEmpty(t *testing.T) {
	var d Dependencies
	if !d.IsEmpty() {
		t.Error("new Dependencies should be empty")
	}
	d.Add("x")
	if d.IsEmpty() {
		t.Error("Dependencies with one item should not be empty")
	}
}

func TestDependencies_First(t *testing.T) {
	var d Dependencies
	if got := d.First(); got != "" {
		t.Errorf("First() on empty = %q, want empty", got)
	}

	d.Add("first")
	if got := d.First(); got != "first" {
		t.Errorf("First() = %q, want %q", got, "first")
	}
}

func TestDependencies_All(t *testing.T) {
	var d Dependencies
	d.Add("a")
	d.Add("b")
	d.Add("c")

	var got []string
	for v := range d.All() {
		got = append(got, v)
	}
	want := []string{"a", "b", "c"}
	if !slices.Equal(got, want) {
		t.Errorf("All() = %v, want %v", got, want)
	}
}

func TestDependencies_AllEarlyExit(t *testing.T) {
	var d Dependencies
	d.Add("a")
	d.Add("b")
	d.Add("c")

	var got []string
	for v := range d.All() {
		got = append(got, v)
		if v == "b" {
			break
		}
	}
	want := []string{"a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("All() early exit = %v, want %v", got, want)
	}
}

func TestDependencies_ToSlice(t *testing.T) {
	var d Dependencies
	if got := d.ToSlice(); got != nil {
		t.Errorf("ToSlice() on empty = %v, want nil", got)
	}

	d.Add("a")
	d.Add("b")
	d.Add("c")

	got := d.ToSlice()
	want := []string{"a", "b", "c"}
	if !slices.Equal(got, want) {
		t.Errorf("ToSlice() = %v, want %v", got, want)
	}
}

func TestDependencies_Clone(t *testing.T) {
	var d Dependencies
	d.Add("a")
	d.Add("b")
	d.Add("c")

	clone := d.Clone()
	if !slices.Equal(clone.ToSlice(), d.ToSlice()) {
		t.Errorf("Clone() = %v, want %v", clone.ToSlice(), d.ToSlice())
	}

	d.Add("d")
	if clone.Len() != 3 {
		t.Errorf("Clone was mutated by original: len=%d", clone.Len())
	}
}

func TestDependencies_JSON(t *testing.T) {
	var d Dependencies
	d.Add("x")
	d.Add("y")
	d.Add("z")

	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var d2 Dependencies
	if err := d2.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if !slices.Equal(d.ToSlice(), d2.ToSlice()) {
		t.Errorf("round-trip: got %v, want %v", d2.ToSlice(), d.ToSlice())
	}
}

func TestDependencies_UnmarshalJSON_Invalid(t *testing.T) {
	var d Dependencies
	if err := d.UnmarshalJSON([]byte(`not json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDependenciesFromSlice(t *testing.T) {
	d := DependenciesFromSlice([]string{"a", "b", "c"})
	if d.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", d.Len())
	}
	if got := d.Get(2); got != "c" {
		t.Errorf("Get(2) = %q, want %q", got, "c")
	}
}
