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

// Package asmgen implements a text-based Plan 9 assembly code
// generator. It defines architecture-neutral handler descriptions
// that are translated to concrete Plan 9 assembly via pluggable
// architecture adapters.
//
// The domain package has no knowledge of any specific processor
// architecture. That knowledge lives entirely in the adapter
// implementations. Assembly is treated as a text generation problem:
// the generator performs no instruction encoding, no operand
// validation, and no register allocation. It maps abstract operations
// to text strings via architecture adapters.
package asmgen
