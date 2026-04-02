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

// Package hnsw implements a Hierarchical Navigable Small World (HNSW)
// graph for approximate nearest neighbour search over float32 vectors.
//
// The implementation is generic over the key type K (any comparable
// type) and supports concurrent reads and writes. It provides
// configurable parameters for index quality versus build speed
// trade-offs, including maxNeighboursPerLayer (neighbours per node),
// constructionCandidateCount (build-time candidate list size), and
// searchCandidateCount (query-time candidate list size). Supported
// distance metrics include Euclidean, cosine similarity, and dot
// product.
//
// # Design rationale
//
// HNSW was chosen over other approximate nearest neighbour algorithms
// (LSH, KD-trees, IVF) because its layered skip-list structure gives
// logarithmic search complexity whilst maintaining high recall, and it
// handles incremental inserts and deletions without full reindexing.
// The parameters M (maxNeighboursPerLayer) and efConstruction
// (constructionCandidateCount) let callers trade index build time and
// memory for query accuracy, making the same code suitable for both
// latency-sensitive serving and offline batch indexing.
//
// # Usage
//
//	g := hnsw.New[string](128, vectormaths.Cosine)
//	g.Insert("doc1", embedding)
//	results := g.Search(query, 10, 0)
package hnsw
