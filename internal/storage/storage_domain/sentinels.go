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

package storage_domain

import "errors"

var (
	// errDispatcherNil is returned when a nil dispatcher is provided during
	// registration.
	errDispatcherNil = errors.New("dispatcher cannot be nil")

	// errNoDispatcher is returned when a dispatcher operation is attempted but
	// no dispatcher has been registered.
	errNoDispatcher = errors.New("no dispatcher registered")

	// errTransformerNil is returned when a nil transformer is provided during
	// registration.
	errTransformerNil = errors.New("transformer cannot be nil")

	// errTransformerNameEmpty is returned when a transformer is registered
	// with an empty name.
	errTransformerNameEmpty = errors.New("transformer name cannot be empty")

	// errInvalidRepository is returned when a storage operation references an
	// invalid repository.
	errInvalidRepository = errors.New("validation failed: invalid repository")

	// errContentTypeEmpty is returned when a storage upload is attempted
	// without specifying a content type.
	errContentTypeEmpty = errors.New("validation failed: content type cannot be empty")

	// errNoObjectsToUpload is returned when a bulk upload is called with an
	// empty object list.
	errNoObjectsToUpload = errors.New("validation failed: no objects to upload")

	// errNoKeysToRemove is returned when a bulk removal is called with an
	// empty key list.
	errNoKeysToRemove = errors.New("validation failed: no keys to remove")

	// errNegativeConcurrency is returned when a concurrency setting is
	// negative.
	errNegativeConcurrency = errors.New("validation failed: concurrency cannot be negative")

	// errInvalidSourceRepo is returned when a copy or move operation
	// references an invalid source repository.
	errInvalidSourceRepo = errors.New("validation failed: invalid source repository")

	// errInvalidDestRepo is returned when a copy or move operation references
	// an invalid destination repository.
	errInvalidDestRepo = errors.New("validation failed: invalid destination repository")

	// errKeyEmpty is returned when a storage operation is attempted with an
	// empty key.
	errKeyEmpty = errors.New("key cannot be empty")

	// errKeyWithCAS is returned when a key is provided but content-addressable
	// storage is enabled, as the key is derived from the content hash.
	errKeyWithCAS = errors.New("key must be empty when UseContentAddressing is true (key is generated from content hash)")

	// ErrSingleflightObjectTooLarge is returned by the singleflight buffering
	// path when the underlying provider yields more bytes than the configured
	// SingleflightMemoryThreshold permits, defending against a provider that
	// reports a small Stat but streams a large payload.
	ErrSingleflightObjectTooLarge = errors.New("storage: singleflight object exceeds memory threshold")

	// ErrDiskObjectTooLarge is returned by the disk provider when an object
	// being read fully into memory exceeds the configured cap.
	ErrDiskObjectTooLarge = errors.New("storage: disk object exceeds maximum read size")
)
