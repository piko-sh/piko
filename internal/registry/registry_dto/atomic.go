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

// ActionType identifies the kind of atomic registry operation.
type ActionType string

const (
	// ActionTypeUpsertArtefact creates or updates an artefact record.
	ActionTypeUpsertArtefact ActionType = "UPSERT_ARTEFACT"

	// ActionTypeDeleteArtefact marks an artefact for deletion from the store.
	ActionTypeDeleteArtefact ActionType = "DELETE_ARTEFACT"

	// ActionTypeAddGCHints adds garbage collection hints for orphaned blobs.
	ActionTypeAddGCHints ActionType = "ADD_GC_HINTS"
)

// AtomicAction represents a single operation in an atomic registry transaction.
type AtomicAction struct {
	// Type specifies which operation to perform on the registry.
	Type ActionType

	// ArtefactID identifies the artefact for deletion actions.
	ArtefactID string

	// Artefact holds the artefact metadata for upsert actions.
	Artefact *ArtefactMeta

	// GCHints holds garbage collection hints for orphaned blobs.
	GCHints []GCHint
}
