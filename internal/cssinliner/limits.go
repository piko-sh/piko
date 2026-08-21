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

package cssinliner

const (
	// DefaultMaxImportDepth is how deep a chain of CSS imports may nest before the build
	// stops.
	DefaultMaxImportDepth = 64

	// DefaultMaxInlinedBytes is how much CSS one component may pull in through imports
	// before the build stops.
	DefaultMaxInlinedBytes = 32 << 20
)

// Limits bounds how far and how much a single inlining operation may pull in.
type Limits struct {
	// MaxDepth is the deepest chain of nested imports allowed.
	MaxDepth int

	// MaxTotalBytes is the largest total size of imported CSS allowed for one component.
	MaxTotalBytes int
}

// withDefaults fills in any unset field so the rest of the inliner can read the limits
// without checking for zero.
//
// Returns Limits which has every field set to a usable value.
func (l Limits) withDefaults() Limits {
	if l.MaxDepth <= 0 {
		l.MaxDepth = DefaultMaxImportDepth
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = DefaultMaxInlinedBytes
	}
	return l
}
