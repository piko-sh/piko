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

package storage_dto

// MigrateParams holds all settings for moving objects
// between storage providers.
type MigrateParams struct {
	// SourceProvider is the name of the registered provider to migrate data from.
	SourceProvider string

	// DestinationProvider is the name of the registered
	// provider to migrate data to.
	DestinationProvider string

	// Repository is the name of the repository that holds the objects to migrate.
	Repository string

	// Keys lists the object keys to migrate.
	Keys []string

	// Concurrency is the number of files to migrate at the same time.
	// Defaults to 10 if not set.
	Concurrency int

	// RemoveSourceAfterSuccess, if true, deletes the object from the source
	// provider after a successful copy, turning the operation into a move.
	// Defaults to false.
	RemoveSourceAfterSuccess bool

	// ContinueOnError determines if the batch should
	// continue after individual failures.
	// Defaults to true.
	ContinueOnError bool
}
