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

// Package search_adapters provides adapters for registering and querying
// full-text search indexes.
//
// Register an index during initialisation:
//
//	func init() {
//	    search_adapters.RegisterSearchIndex("docs", "fast", searchFastBlob)
//	}
//
// At runtime, retrieve a reader for querying:
//
//	reader, err := search_adapters.GetSearchIndex("docs", "fast")
//	if err != nil {
//	    return err
//	}
//	postings, idf, err := reader.GetTermPostings("example")
//
// # Thread safety
//
// The search index registry is safe for concurrent reads after initialisation.
// Individual reader instances are safe for concurrent use.
package search_adapters
