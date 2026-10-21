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

package orchestrator_adapters

import (
	"sync"

	"piko.sh/piko/internal/registry/registry_dto"
)

// EmptyOutputs records profiles built without variants and the source hashes they were
// built from.
//
// A profile counts as built while its source is unchanged.
//
// Safe for concurrent use.
type EmptyOutputs struct {
	// sources maps artefact IDs and profile names to source hashes and is guarded by mu.
	sources map[emptyOutputKey]string

	// mu guards sources.
	mu sync.Mutex
}

// BridgeOption configures an ArtefactWorkflowBridge.
type BridgeOption func(*ArtefactWorkflowBridge)

// WithBridgeEmptyOutputs makes the bridge count a profile built with no output as built,
// reading the record the compiler executor writes.
//
// Takes emptyOutputs (*EmptyOutputs) which is the record the compiler writes.
//
// Returns BridgeOption which applies the setting.
func WithBridgeEmptyOutputs(emptyOutputs *EmptyOutputs) BridgeOption {
	return func(b *ArtefactWorkflowBridge) {
		b.emptyOutputs = emptyOutputs
	}
}

// emptyOutputKey identifies one profile of one artefact.
type emptyOutputKey struct {
	// artefactID identifies the artefact.
	artefactID string

	// profile names the profile.
	profile string
}

// NewEmptyOutputs creates an empty record.
//
// Returns *EmptyOutputs which is ready for use.
func NewEmptyOutputs() *EmptyOutputs {
	return &EmptyOutputs{sources: make(map[emptyOutputKey]string), mu: sync.Mutex{}}
}

// Record notes that a profile built from the given source content produced no output.
//
// Takes artefactID (string) which identifies the artefact.
// Takes profile (string) which names the profile.
// Takes sourceHash (string) which is the content hash of the source it was built from.
//
// Safe for concurrent use; a nil record ignores the call.
func (e *EmptyOutputs) Record(artefactID, profile, sourceHash string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sources[emptyOutputKey{artefactID: artefactID, profile: profile}] = sourceHash
}

// BuiltEmpty reports whether a profile was built from the artefact's current source and
// produced no output, so it is complete although it has no variant.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact's current state.
// Takes profile (string) which names the profile.
// Takes sourceVariantID (string) which identifies the variant the profile is built from.
//
// Returns bool which is true when the recorded build used the source variant's current
// content.
//
// Safe for concurrent use; a nil record reports false.
func (e *EmptyOutputs) BuiltEmpty(artefact *registry_dto.ArtefactMeta, profile, sourceVariantID string) bool {
	if e == nil || artefact == nil {
		return false
	}
	e.mu.Lock()
	sourceHash, recorded := e.sources[emptyOutputKey{artefactID: artefact.ID, profile: profile}]
	e.mu.Unlock()
	if !recorded {
		return false
	}
	source := findVariantByID(artefact.ActualVariants, sourceVariantID)
	return source != nil && source.ContentHash == sourceHash
}
