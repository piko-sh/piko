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

package collection_dto

import "piko.sh/piko/wdk/safedisk"

// ContentSource describes where a collection's content files are located.
//
// Created per collection directive and passed to provider methods, keeping
// providers stateless with respect to file location. This separates the
// concern of "how to process content" (the provider) from "where content
// lives" (the source).
type ContentSource struct {
	// Sandbox provides restricted filesystem access to the content directory.
	Sandbox safedisk.Sandbox

	// BasePath is the root directory for resolving content file paths.
	BasePath string

	// IsExternal indicates the content comes from an external Go module
	// rather than the local project's content/ directory.
	IsExternal bool
}
