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
	"time"
)

// VariantStatus represents the current processing state of a variant.
type VariantStatus string

const (
	// VariantStatusReady indicates the variant is processed and available.
	VariantStatusReady VariantStatus = "READY"

	// VariantStatusStale indicates the variant needs reprocessing.
	VariantStatusStale VariantStatus = "STALE"

	// VariantStatusPending indicates the variant is queued for processing.
	VariantStatusPending VariantStatus = "PENDING"

	// PriorityNeed means the profile must be processed before the content is served.
	PriorityNeed ProfilePriority = "NEED"

	// PriorityWant marks a profile for background processing.
	PriorityWant ProfilePriority = "WANT"
)

// ProfilePriority indicates whether a profile is required or optional.
type ProfilePriority string

// DesiredProfile defines how to generate a variant from a source image. It specifies the
// output format, priority, and any dependencies on other profiles.
type DesiredProfile struct {
	// Priority specifies the task scheduling priority level.
	Priority ProfilePriority `json:"priority"`

	// CapabilityName identifies which capability to run for this profile.
	CapabilityName string `json:"capabilityName"`

	// Params holds key-value settings for this profile's capability.
	Params ProfileParams `json:"params"`

	// ResultingTags holds tags to apply to the output variant.
	ResultingTags Tags `json:"resultingTags"`

	// DependsOn lists variant IDs that must exist before this profile can run.
	DependsOn Dependencies `json:"dependsOn"`
}

// NamedProfile pairs a profile name with its settings. Used in
// ArtefactMeta.DesiredProfiles slice.
type NamedProfile struct {
	// Name identifies this profile for lookup and matching.
	Name string `json:"name"`

	// Profile holds the image transformation settings for this named profile.
	Profile DesiredProfile `json:"profile"`
}

// VariantChunk represents a segment of a chunked variant (e.g., HLS segment).
type VariantChunk struct {
	// CreatedAt is the timestamp when the chunk was created.
	CreatedAt time.Time `json:"createdAt"`

	// DurationSeconds is the playback length in seconds; nil when unknown.
	DurationSeconds *float64 `json:"durationSeconds,omitempty"`

	// ChunkID is the unique identifier for this chunk.
	ChunkID string `json:"chunkId"`

	// StorageKey is the key used to find this chunk in the storage backend.
	StorageKey string `json:"storageKey"`

	// StorageBackendID identifies which storage backend holds this chunk's data.
	StorageBackendID string `json:"storageBackendId"`

	// ContentHash is the SHA256 hash of the chunk content.
	ContentHash string `json:"contentHash,omitempty"`

	// MimeType is the media type of this chunk; must not be empty.
	MimeType string `json:"mimeType"`

	// SizeBytes is the chunk size in bytes; must be positive.
	SizeBytes int64 `json:"sizeBytes"`

	// SequenceNumber is the position of this chunk in the variant's sequence.
	SequenceNumber int `json:"sequenceNumber"`
}

// Variant represents a processed version of an artefact, such as a resized image or
// minified JavaScript file.
type Variant struct {
	// CreatedAt is when the variant was created.
	CreatedAt time.Time `json:"createdAt"`

	// MetadataTags holds key-value metadata about the variant, such as MIME type, ETag, and
	// content encoding.
	MetadataTags Tags `json:"metadataTags"`

	// SRIHash is the SHA-384 Subresource Integrity hash of the uncompressed variant content,
	// in the format "sha384-<base64>". Populated at registration time for variants whose
	// content may be referenced in HTML script or link tags.
	SRIHash string `json:"sriHash,omitempty"`

	// Origin records whether this variant was produced at build time or created at runtime.
	//
	// A runtime variant may be an on-demand resolution variant or an upload. Empty is
	// treated as runtime so a seed re-sync never deletes unmarked data.
	Origin VariantOrigin `json:"origin,omitempty"`

	// StorageKey is the unique key used to find this variant in storage.
	StorageKey string `json:"storageKey"`

	// MimeType is the MIME type of the variant content; must not be empty.
	MimeType string `json:"mimeType"`

	// VariantID identifies this variant; "source" means the original file.
	VariantID string `json:"variantId"`

	// ContentHash is the SHA256 hash of the blob content.
	ContentHash string `json:"contentHash,omitempty"`

	// Status indicates the current lifecycle state of this variant.
	Status VariantStatus `json:"status"`

	// StorageBackendID identifies which storage backend holds this variant's data.
	StorageBackendID string `json:"storageBackendId"`

	// BuildRelease identifies the release that produced a build-origin variant.
	//
	// The value is typically the VCS revision, or an explicit release identifier. It lets
	// multiple releases coexist in one shared backend during a canary or A/B rollout, and
	// lets a retired release be garbage-collected without touching other releases. Empty for
	// runtime variants.
	BuildRelease string `json:"buildRelease,omitempty"`

	// BuildHash is the full build-identity hash of the producing build (vcs.revision plus a
	// dirty-tree marker). It disambiguates two builds that share a release identifier.
	BuildHash string `json:"buildHash,omitempty"`

	// InputFingerprint hashes the inputs that determine this variant's validity.
	//
	// The inputs are the source content hash plus the transform capability and its
	// parameters (which include quality). It is one half of the layered record identity
	// (variantID, inputFingerprint): the layer merge deduplicates records on it, so two
	// releases whose transforms produced identical inputs share one record while differing
	// fingerprints coexist, and validity checking compares it against the current source to
	// stop a stale derived variant from being served.
	InputFingerprint string `json:"inputFingerprint,omitempty"`

	// Transform is the derivation recipe: the parent, the capability and version, and the
	// parameters actually applied. It is the preimage of InputFingerprint for a derived
	// variant, and zero for a source variant.
	Transform VariantTransform `json:"transform,omitzero"`

	// Chunks holds the blob chunks that form this variant.
	Chunks []VariantChunk `json:"chunks,omitempty"`

	// SizeBytes is the size of the variant in bytes; must be positive.
	SizeBytes int64 `json:"sizeBytes"`

	// Producer records who produced this variant's bytes: the build, or the running server.
	//
	// It is one of the two orthogonal provenance axes (the other is Kind). It supersedes
	// Origin, which is retained during the transition until every reader is migrated.
	Producer VariantProducer `json:"producer,omitempty"`

	// Kind records whether this variant has a parent, and so whether it is regenerable cache
	// (derived) or irreplaceable data (source). It is the second provenance axis.
	Kind VariantKind `json:"kind,omitempty"`
}

// VariantOrigin records whether a variant was produced at build time or created at
// runtime.
type VariantOrigin string

const (
	// VariantOriginBuild marks a variant produced by the build/generation step.
	VariantOriginBuild VariantOrigin = "build"

	// VariantOriginRuntime marks a variant created at runtime.
	VariantOriginRuntime VariantOrigin = "runtime"
)

// ArtefactMeta holds the metadata for a registered artefact in the registry. It tracks
// the artefact's variants, desired profiles, and storage details.
type ArtefactMeta struct {
	// CreatedAt is the time when the artefact was first created.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when the artefact metadata was last changed.
	UpdatedAt time.Time `json:"updatedAt"`

	// ID is the unique identifier for this artefact.
	ID string `json:"id"`

	// ReleaseID is the layer this record belongs to.
	//
	// The value is a release identifier for a published build layer, the reserved runtime
	// marker for the mutable runtime layer, or empty for a single-layer store. It is the
	// second half of a layer row's identity and is set by the store on read and by the
	// writer on write. Merging layers reads it to know which layer a record came from.
	ReleaseID string `json:"releaseId,omitempty"`

	// SourcePath is the original file path of the source asset.
	SourcePath string `json:"sourcePath,omitempty"`

	// Status is the current processing state of this artefact.
	Status VariantStatus `json:"status"`

	// DesiredProfiles holds the named profiles for this artefact.
	DesiredProfiles []NamedProfile `json:"desiredProfiles"`

	// ActualVariants holds the variants that exist for this artefact.
	ActualVariants []Variant `json:"actualVariants"`
}

// GetProfile returns a profile by name.
//
// Takes name (string) which specifies the profile name to look up.
//
// Returns DesiredProfile which contains the profile data if found, or an empty profile if
// not found.
// Returns bool which is true if the profile was found, false otherwise.
func (a *ArtefactMeta) GetProfile(name string) (DesiredProfile, bool) {
	for i := range a.DesiredProfiles {
		if a.DesiredProfiles[i].Name == name {
			return a.DesiredProfiles[i].Profile, true
		}
	}
	return DesiredProfile{}, false
}

// SetProfile sets a profile by name.
//
// If a profile with this name exists, it is updated. Otherwise, a new profile is added.
//
// Takes name (string) which identifies the profile to set.
// Takes profile (*DesiredProfile) which specifies the profile settings.
func (a *ArtefactMeta) SetProfile(name string, profile *DesiredProfile) {
	for i := range a.DesiredProfiles {
		if a.DesiredProfiles[i].Name == name {
			a.DesiredProfiles[i].Profile = *profile
			return
		}
	}
	a.DesiredProfiles = append(a.DesiredProfiles, NamedProfile{
		Name:    name,
		Profile: *profile,
	})
}

// HasProfile reports whether a profile with the given name exists.
//
// Takes name (string) which specifies the profile name to search for.
//
// Returns bool which is true if a matching profile exists, false otherwise.
func (a *ArtefactMeta) HasProfile(name string) bool {
	for i := range a.DesiredProfiles {
		if a.DesiredProfiles[i].Name == name {
			return true
		}
	}
	return false
}

// ComputeStatus determines artefact status based on variants.
//
// Returns VariantStatus which is PENDING if no variants exist, READY if any variant is
// ready, or STALE otherwise.
func (a *ArtefactMeta) ComputeStatus() VariantStatus {
	if len(a.ActualVariants) == 0 {
		return VariantStatusPending
	}
	for i := range a.ActualVariants {
		if a.ActualVariants[i].Status == VariantStatusReady {
			return VariantStatusReady
		}
	}
	return VariantStatusStale
}

// Clone returns a deep copy that shares no mutable state with the original.
//
// Returns *ArtefactMeta which is an independent copy, or nil when the receiver is nil.
func (a *ArtefactMeta) Clone() *ArtefactMeta {
	if a == nil {
		return nil
	}

	clone := *a

	clone.ActualVariants = make([]Variant, len(a.ActualVariants))
	for i := range a.ActualVariants {
		variant := a.ActualVariants[i]
		variant.MetadataTags = a.ActualVariants[i].MetadataTags.Clone()
		variant.Transform.Params = a.ActualVariants[i].Transform.Params.Clone()
		if a.ActualVariants[i].Chunks != nil {
			variant.Chunks = make([]VariantChunk, len(a.ActualVariants[i].Chunks))
			copy(variant.Chunks, a.ActualVariants[i].Chunks)
		}
		clone.ActualVariants[i] = variant
	}

	clone.DesiredProfiles = make([]NamedProfile, len(a.DesiredProfiles))
	for i := range a.DesiredProfiles {
		profile := a.DesiredProfiles[i]
		profile.Profile.Params = a.DesiredProfiles[i].Profile.Params.Clone()
		profile.Profile.ResultingTags = a.DesiredProfiles[i].Profile.ResultingTags.Clone()
		profile.Profile.DependsOn = a.DesiredProfiles[i].Profile.DependsOn.Clone()
		clone.DesiredProfiles[i] = profile
	}

	return &clone
}
