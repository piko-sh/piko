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

// Package driven_transform_redaction removes sensitive content from PDFs
// as a post-processing transformer.
//
// It supports text pattern redaction (replacing matched text with spaces
// in decoded content streams), region redaction (drawing filled black
// rectangles over specified areas), and optional metadata stripping
// (removing /Info from the trailer and /Metadata from the catalog).
package driven_transform_redaction
