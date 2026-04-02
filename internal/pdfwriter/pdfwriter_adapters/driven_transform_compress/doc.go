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

// Package driven_transform_compress implements byte-level compression as a
// PDF post-processing transformer.
//
// It compresses the final PDF output using a configurable algorithm,
// defaulting to Zstandard. The compressed output is no longer a valid PDF
// and must be decompressed before use. This transformer should run after
// all PDF-structural transforms in the chain.
package driven_transform_compress
