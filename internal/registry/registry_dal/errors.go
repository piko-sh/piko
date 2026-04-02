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

package registry_dal

import "errors"

var (
	// ErrArtefactNotFound is returned when an artefact cannot be found by ID.
	ErrArtefactNotFound = errors.New("artefact not found")

	// ErrBlobReferenceNotFound is returned when a blob reference cannot be found
	// by storage key.
	ErrBlobReferenceNotFound = errors.New("blob reference not found")

	// ErrSearchUnsupported is returned when a search query type is not supported
	// by the backend.
	ErrSearchUnsupported = errors.New("search query type not supported")

	// ErrTransactionFailed is returned when a transaction cannot be committed or
	// rolled back.
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrDatabaseClosed is returned when operations are attempted on a closed
	// database connection.
	ErrDatabaseClosed = errors.New("database connection is closed")
)
