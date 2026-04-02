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

// Package collection provides the public API for integrating
// external data sources with the Piko collection system.
//
// This package exposes the types and interfaces needed to create
// custom collection providers that connect external data sources
// (headless CMSes, databases, APIs) to the Piko framework. It is
// the stable public facade over the internal collection domain
// and DTO packages.
//
// The framework supports three data-fetching strategies: static
// (build-time), dynamic (runtime), and hybrid (build-time snapshot
// with background revalidation via ISR). Implement the
// [CollectionProvider] interface for full control, or use the
// simpler [SimpleProvider] interface with
// [NewSimpleProviderAdapter] when AST-based code generation is
// not needed.
//
// Filtering, sorting, and pagination types are included for
// querying collection data. Utility functions [ApplyFilterGroup],
// [SortItems], and [PaginateItems] apply these operations to
// content item slices.
package collection
