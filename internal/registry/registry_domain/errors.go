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

import "errors"

var (
	// ErrArtefactNotFound is returned when an artefact cannot be found by its ID.
	ErrArtefactNotFound = errors.New("artefact not found")

	// ErrArtefactExists is returned when attempting to create an artefact that
	// already exists.
	ErrArtefactExists = errors.New("artefact already exists")

	// ErrVariantNotFound is returned when a specific variant is not found in an
	// artefact.
	ErrVariantNotFound = errors.New("variant not found")

	// ErrChunkNotFound is returned when a chunk cannot be found in a variant.
	ErrChunkNotFound = errors.New("chunk not found")

	// ErrBlobNotFound is returned when a blob cannot be found in the storage
	// backend.
	ErrBlobNotFound = errors.New("blob not found")

	// ErrBlobReferenceNotFound is returned when a blob reference count entry
	// does not exist for a given storage key. This is expected after a restart
	// when using in-memory stores, since blob ref counts are ephemeral.
	ErrBlobReferenceNotFound = errors.New("blob reference not found")

	// ErrCacheMiss is returned when an artefact is not found in the cache.
	ErrCacheMiss = errors.New("artefact not found in cache")

	// ErrSearchUnsupported is returned when a search operation is not supported by
	// the adapter.
	ErrSearchUnsupported = errors.New("search operation is not supported by this adapter")

	// ErrRangeNotSatisfiable is returned when a requested byte range cannot be
	// satisfied.
	ErrRangeNotSatisfiable = errors.New("requested range is not satisfiable")
)
