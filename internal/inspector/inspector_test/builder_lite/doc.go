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

// Package builder_lite provides integration tests for the
// LiteBuilder, the lightweight AST-only code inspection path that
// extracts type information without invoking go/packages.
//
// Each subdirectory under testdata/ represents a single golden-file
// scenario. Tests pre-generate a minimal stdlib type data set in
// TestMain and compare serialised output against the golden files.
package builder_lite
