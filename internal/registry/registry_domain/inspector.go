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

package registry_domain

import "context"

// ArtefactSummary holds the number of artefacts for each status on dashboards.
type ArtefactSummary struct {
	// Status is the artefact status value.
	Status string

	// Count is the number of artefacts with this status.
	Count int64
}

// VariantSummary holds variant counts grouped by status for dashboard display.
type VariantSummary struct {
	// Status is the current state of the variant.
	Status string

	// Count is the number of variants with this status.
	Count int64
}

// ArtefactListItem holds artefact data for list display in the TUI.
type ArtefactListItem struct {
	// ID is the unique identifier for this artefact.
	ID string

	// SourcePath is the path to the source file for this artefact.
	SourcePath string

	// Status is the current state of the artefact.
	Status string

	// VariantCount is the number of variants for this artefact.
	VariantCount int64

	// TotalSize is the combined size of all variants in bytes.
	TotalSize int64

	// CreatedAt is the Unix timestamp when the artefact was created.
	CreatedAt int64

	// UpdatedAt is the Unix timestamp when the artefact was last modified.
	UpdatedAt int64
}

// RegistryInspector provides read-only access to registry state.
// The monitoring service uses this interface to show artefact and variant
// data in the TUI without needing direct database access.
type RegistryInspector interface {
	// ListArtefactSummary returns artefact counts grouped by
	// artefact status.
	//
	// Returns []ArtefactSummary which contains the count for each status.
	// Returns error when the query fails.
	ListArtefactSummary(ctx context.Context) ([]ArtefactSummary, error)

	// ListVariantSummary returns variant counts grouped by status.
	//
	// Returns []VariantSummary which contains the count for each status.
	// Returns error when the query fails.
	ListVariantSummary(ctx context.Context) ([]VariantSummary, error)

	// ListRecentArtefacts returns the most recently updated artefacts.
	//
	// Takes limit (int32) which specifies the maximum number of artefacts to return.
	//
	// Returns []ArtefactListItem which contains the artefact data for display.
	// Returns error when the query fails.
	ListRecentArtefacts(ctx context.Context, limit int32) ([]ArtefactListItem, error)
}
