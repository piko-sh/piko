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

package collection_dto

import "go/ast"

// DynamicCollectionInfo contains instructions for the Generator to emit
// dynamic collection fetcher code.
//
// Created by the Collection Service during annotation and consumed by the
// Generator during code emission. Bridges the gap between collection provider
// blueprints and final Go code generation.
//
// Design Philosophy:
//   - Created at build-time by CollectionService
//   - Attached to GoGeneratorAnnotation
//   - Consumed by Generator's expression emitter
//   - Contains everything needed to emit fetcher function + call
//
// For hybrid mode (ISR), it also contains:
//   - HybridConfig with revalidation settings
//   - SnapshotETag for staleness detection
//   - RevalidatorCode for background refresh
type DynamicCollectionInfo struct {
	// TargetType is the user's struct type (e.g., AST for "Post").
	//
	// This is the element type of the slice that GetCollection returns.
	TargetType ast.Expr

	// FetcherCode holds the runtime fetch code blueprint from the provider.
	//
	// Contains the AST for a function that fetches data at runtime.
	// The generator clones the function, renames it, and adds it to the
	// component file.
	//
	// For pure dynamic providers: always set.
	// For hybrid providers: may be nil (uses RevalidatorCode instead).
	FetcherCode *RuntimeFetcherCode

	// HybridConfig holds the hybrid mode settings when HybridMode is true;
	// nil when hybrid mode is not in use.
	HybridConfig *HybridConfig

	// RevalidatorCode is the AST for the revalidation function.
	//
	// Checks the ETag and fetches new content if it has changed.
	// The Generator clones the function and adds it to the component file.
	//
	// Only set when HybridMode is true.
	RevalidatorCode *RuntimeFetcherCode

	// ProviderName identifies the runtime provider to call, such as "headless-cms",
	// "contentful", "database", or "markdown". This value is passed to
	// pikoruntime.FetchCollection in generated code.
	ProviderName string

	// CollectionName is the name of the collection to fetch, such as "blog",
	// "products", or "team". This value is passed to pikoruntime.FetchCollection
	// in the generated code.
	CollectionName string

	// SnapshotETag is the ETag computed at build time.
	//
	// Used for staleness detection during runtime revalidation.
	// Format depends on provider (e.g., "md-{xxhash64}" for markdown).
	//
	// Only populated when HybridMode is true.
	SnapshotETag string

	// HybridMode indicates this is a hybrid (ISR) collection.
	//
	// When true, the generator emits code that:
	//   1. Returns the embedded static snapshot immediately
	//   2. Triggers background revalidation if TTL expired
	//   3. Updates cache when ETag changes
	HybridMode bool
}
