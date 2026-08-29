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

import (
	"context"

	"piko.sh/piko/internal/registry/registry_dto"
)

// ArtefactLocker is an optional MetadataStore capability for a row-locking read.
type ArtefactLocker interface {
	// GetArtefactForUpdate reads an artefact under a row-level lock held for the enclosing
	// transaction.
	//
	// Takes artefactID (string) which identifies the artefact to read.
	//
	// Returns *registry_dto.ArtefactMeta which is the locked artefact metadata.
	// Returns error when the artefact cannot be read.
	GetArtefactForUpdate(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)
}

// ReadArtefactForLockedUpdate reads an artefact for a read-modify-write.
//
// It uses the store's row-locking read (GetArtefactForUpdate) when the store implements
// ArtefactLocker, and otherwise falls back to GetArtefact. Call it inside a RunAtomic so
// the lock is held across the read and the subsequent write.
//
// Takes store which is the (transactional) metadata store.
// Takes artefactID which identifies the artefact to read.
//
// Returns the artefact, or ErrArtefactNotFound when absent.
func ReadArtefactForLockedUpdate(ctx context.Context, store MetadataStore, artefactID string) (*registry_dto.ArtefactMeta, error) {
	if locker, ok := store.(ArtefactLocker); ok {
		return locker.GetArtefactForUpdate(ctx, artefactID)
	}
	return store.GetArtefact(ctx, artefactID)
}
