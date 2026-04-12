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

//go:build !nosimd

package interp_domain

const (
	// simdKernelRecogniserEnabled gates the SIMD AST-pattern recogniser.
	//
	// Detects canonical numerical loop shapes and emits SIMD sub-opcodes. Default build:
	// enabled. To disable globally (e.g. to isolate a regression to the recogniser path)
	// build with the `nosimd` tag, which selects the sibling constant in
	// flag_recogniser_simd_off.go.
	simdKernelRecogniserEnabled = true
)
