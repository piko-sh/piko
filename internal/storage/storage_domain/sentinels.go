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
	errDispatcherNil = errors.New("dispatcher cannot be nil")

	errNoDispatcher = errors.New("no dispatcher registered")

	errTransformerNil = errors.New("transformer cannot be nil")

	errTransformerNameEmpty = errors.New("transformer name cannot be empty")

	errInvalidRepository = errors.New("validation failed: invalid repository")

	errContentTypeEmpty = errors.New("validation failed: content type cannot be empty")

	errNoObjectsToUpload = errors.New("validation failed: no objects to upload")

	errNoKeysToRemove = errors.New("validation failed: no keys to remove")

	errNegativeConcurrency = errors.New("validation failed: concurrency cannot be negative")

	errInvalidSourceRepo = errors.New("validation failed: invalid source repository")

	errInvalidDestRepo = errors.New("validation failed: invalid destination repository")

	errKeyEmpty = errors.New("key cannot be empty")

	errKeyWithCAS = errors.New("key must be empty when UseContentAddressing is true (key is generated from content hash)")
)
