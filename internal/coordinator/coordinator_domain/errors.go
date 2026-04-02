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

package coordinator_domain

import "errors"

var (
	// ErrCacheMiss is returned by a CachePort's Get method when an item is not found.
	ErrCacheMiss = errors.New("coordinator cache: item not found")

	// ErrInvalidCacheEntry is returned when attempting to cache an invalid entry
	// (e.g., nil values, version mismatch, corrupted data).
	ErrInvalidCacheEntry = errors.New("coordinator cache: invalid cache entry")

	// errBuildInProgress indicates that a build is currently
	// running but no previous successful build is available.
	errBuildInProgress = errors.New("coordinator: a build is in progress, but no previous successful build is available")

	// errNoBuildAvailable indicates that no build has been
	// successfully completed.
	errNoBuildAvailable = errors.New("coordinator: no build result is available")
)
