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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package vectormaths provides float32 vector similarity and
// normalisation functions with SIMD acceleration.
//
// It supports cosine, Euclidean, and dot-product similarity via
// [ComputeSimilarity], plus in-place normalisation with
// [NormaliseF32]. The low-level kernels use hand-written Plan 9
// assembly on amd64 and arm64 for SIMD throughput, with a pure Go
// fallback for other architectures and the safe build tag. This
// package contains only pure maths with no domain types.
package vectormaths
