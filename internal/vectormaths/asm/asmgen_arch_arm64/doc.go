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

// Package asmgen_arch_arm64 provides the ARM64 assembly code
// generation backend for SIMD vector mathematics operations.
//
// [VectormathsARM64Arch] extends the core ARM64 architecture
// with NEON implementations of dot product, squared Euclidean
// distance, and L2 normalisation. All emitted instructions use
// Plan 9 assembly syntax with standard column-aligned
// formatting.
package asmgen_arch_arm64
