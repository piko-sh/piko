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

package layouter_dto

// Defines the output types produced by the layout engine.

// LayoutResult is the output of a full layout operation including pagination.
type LayoutResult struct {
	// RootBox is the root of the laid-out box tree. The concrete type is
	// *layouter_domain.LayoutBox but stored as any to avoid a circular
	// import between dto and domain packages.
	RootBox any

	// RootFragment is the root of the fragment tree with
	// parent-relative offsets, stored as any to avoid a
	// circular import.
	RootFragment any

	// Pages holds the paginated output. Each PageOutput describes a single
	// page with its assigned boxes.
	Pages []PageOutput

	// DiagnosticCount is the number of non-fatal issues encountered during
	// layout.
	DiagnosticCount int
}

// PageOutput describes a single page in the paginated output.
type PageOutput struct {
	// Index is the zero-based page number.
	Index int

	// Width is the page width in points.
	Width float64

	// Height is the page height in points.
	Height float64
}
