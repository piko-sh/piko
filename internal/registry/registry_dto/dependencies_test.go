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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencies_AddAndGet(t *testing.T) {
	var d Dependencies

	d.Add("a")
	d.Add("b")
	d.Add("c")

	require.Equal(t, 3, d.Len())
	assert.Equal(t, "a", d.Get(0))
	assert.Equal(t, "b", d.Get(1))
	assert.Equal(t, "c", d.Get(2))
}

func TestDependencies_IsEmpty(t *testing.T) {
	var d Dependencies
	assert.True(t, d.IsEmpty(), "new Dependencies should be empty")
	d.Add("x")
	assert.False(t, d.IsEmpty(), "Dependencies with one item should not be empty")
}

func TestDependencies_First(t *testing.T) {
	var d Dependencies
	assert.Empty(t, d.First(), "First() on empty should be empty")

	d.Add("first")
	assert.Equal(t, "first", d.First())
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
	assert.Equal(t, want, got)
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
	assert.Equal(t, want, got)
}

func TestDependencies_ToSlice(t *testing.T) {
	var d Dependencies
	assert.Nil(t, d.ToSlice(), "ToSlice() on empty should be nil")

	d.Add("a")
	d.Add("b")
	d.Add("c")

	want := []string{"a", "b", "c"}
	assert.Equal(t, want, d.ToSlice())
}

func TestDependencies_Clone(t *testing.T) {
	var d Dependencies
	d.Add("a")
	d.Add("b")
	d.Add("c")

	clone := d.Clone()
	assert.Equal(t, d.ToSlice(), clone.ToSlice())

	d.Add("d")
	assert.Equal(t, 3, clone.Len(), "Clone was mutated by original")
}

func TestDependencies_JSON(t *testing.T) {
	var d Dependencies
	d.Add("x")
	d.Add("y")
	d.Add("z")

	data, err := d.MarshalJSON()
	require.NoError(t, err, "MarshalJSON")

	var d2 Dependencies
	require.NoError(t, d2.UnmarshalJSON(data), "UnmarshalJSON")
	assert.Equal(t, d.ToSlice(), d2.ToSlice())
}

func TestDependencies_UnmarshalJSON_Invalid(t *testing.T) {
	var d Dependencies
	assert.Error(t, d.UnmarshalJSON([]byte(`not json`)), "expected error for invalid JSON")
}

func TestDependenciesFromSlice(t *testing.T) {
	d := DependenciesFromSlice([]string{"a", "b", "c"})
	require.Equal(t, 3, d.Len())
	assert.Equal(t, "c", d.Get(2))
}
