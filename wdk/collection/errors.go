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

package collection

import "errors"

var (
	// ErrCodeGenerationNotSupported is returned when a provider does not
	// support AST-based code generation for runtime fetching.
	ErrCodeGenerationNotSupported = errors.New("code generation not supported by this provider")

	// ErrProviderNotFound is returned when a requested provider is not registered.
	ErrProviderNotFound = errors.New("collection provider not found")

	// ErrCollectionNotFound is returned when a requested collection does not
	// exist.
	ErrCollectionNotFound = errors.New("collection not found")

	// ErrETagNotSupported is returned when a provider does not support ETag
	// operations.
	ErrETagNotSupported = errors.New("ETag operations not supported by this provider")
)
