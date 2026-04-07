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

import (
	"fmt"
	"strings"
)

// CircularDependencyError indicates a circular CSS @import chain was detected.
type CircularDependencyError struct {
	// Path is the chain of file paths forming the cycle.
	Path []string
}

// NewCircularDependencyError creates a new circular dependency error.
//
// Takes path ([]string) which is the chain of paths forming the cycle.
//
// Returns *CircularDependencyError which describes the circular import.
func NewCircularDependencyError(path []string) *CircularDependencyError {
	return &CircularDependencyError{Path: path}
}

// Error implements the error interface.
//
// Returns string which describes the circular dependency path.
func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Path, " -> "))
}
