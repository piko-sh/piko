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
	"fmt"
	"iter"

	"piko.sh/piko/internal/json"
)

// Dependencies stores profile dependencies with inline storage for the common
// case. It implements json.Marshaler and json.Unmarshaler.
//
// Most profiles have zero to two dependencies (usually "source" or another
// profile name). This avoids slice allocation for the common case. For more
// than two dependencies, overflow storage is used.
type Dependencies struct {
	// inline holds the first two dependencies for fast access without heap allocation.
	inline [2]string

	// extra holds dependencies that exceed the inline array capacity.
	extra []string

	// count is the number of dependencies stored in the inline array.
	count uint8
}

// Len returns the total number of dependencies.
//
// Returns int which is the count of all dependencies.
func (d *Dependencies) Len() int {
	return int(d.count) + len(d.extra)
}

// IsEmpty reports whether there are no dependencies.
//
// Returns bool which is true when both the count is zero and no extra
// dependencies exist.
func (d *Dependencies) IsEmpty() bool {
	return d.count == 0 && len(d.extra) == 0
}

// Add appends a dependency to the collection.
//
// Takes dependency (string) which is the dependency to add.
func (d *Dependencies) Add(dependency string) {
	if d.count < 2 {
		d.inline[d.count] = dependency
		d.count++
		return
	}
	d.extra = append(d.extra, dependency)
}

// Get returns the dependency at the given index.
//
// Takes i (int) which specifies the index of the dependency to retrieve.
//
// Returns string which is the dependency at the specified index.
func (d *Dependencies) Get(i int) string {
	if i < int(d.count) {
		return d.inline[i]
	}
	return d.extra[i-int(d.count)]
}

// First returns the first dependency, or an empty string if there are none.
//
// Returns string which is the first dependency or empty if none exist.
func (d *Dependencies) First() string {
	if d.count > 0 {
		return d.inline[0]
	}
	if len(d.extra) > 0 {
		return d.extra[0]
	}
	return ""
}

// All returns an iterator over all dependencies.
//
// Returns iter.Seq[string] which yields each dependency in order.
func (d *Dependencies) All() iter.Seq[string] {
	return func(yield func(string) bool) {
		for i := uint8(0); i < d.count; i++ {
			if !yield(d.inline[i]) {
				return
			}
		}
		for _, dependency := range d.extra {
			if !yield(dependency) {
				return
			}
		}
	}
}

// ToSlice returns all dependencies as a slice.
//
// Returns []string which contains all dependencies, or nil if empty to avoid
// allocation.
func (d *Dependencies) ToSlice() []string {
	total := d.Len()
	if total == 0 {
		return nil
	}
	result := make([]string, 0, total)
	for i := uint8(0); i < d.count; i++ {
		result = append(result, d.inline[i])
	}
	return append(result, d.extra...)
}

// Clone returns a deep copy of the dependencies.
//
// Returns Dependencies which is a separate copy with its own storage.
func (d *Dependencies) Clone() Dependencies {
	clone := Dependencies{
		inline: d.inline,
		extra:  nil,
		count:  d.count,
	}
	if len(d.extra) > 0 {
		clone.extra = make([]string, len(d.extra))
		copy(clone.extra, d.extra)
	}
	return clone
}

// MarshalJSON implements json.Marshaler.
// Serialises as a JSON array for API compatibility.
//
// Returns []byte which contains the JSON-encoded array.
// Returns error when serialisation fails.
func (d Dependencies) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler by deserialising from a JSON
// array.
//
// Takes data ([]byte) which contains the JSON array of dependency strings.
//
// Returns error when the JSON data is not a valid array of strings.
func (d *Dependencies) UnmarshalJSON(data []byte) error {
	var slice []string
	if err := json.Unmarshal(data, &slice); err != nil {
		return fmt.Errorf("unmarshalling dependencies JSON: %w", err)
	}
	for _, dependency := range slice {
		d.Add(dependency)
	}
	return nil
}

// DependenciesFromSlice creates a Dependencies from a slice of strings.
// Use it in tests and when moving from the old slice-based format.
//
// Takes dependencies ([]string) which contains the dependency strings to add.
//
// Returns Dependencies which contains all the provided dependency strings.
func DependenciesFromSlice(dependencies []string) Dependencies {
	var newDependencies Dependencies
	for _, dependency := range dependencies {
		newDependencies.Add(dependency)
	}
	return newDependencies
}
