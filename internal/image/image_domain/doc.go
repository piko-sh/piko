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

// Package image_domain coordinates image transformations, responsive
// variant generation, and placeholder creation through the
// [TransformerPort] interface. It defines port interfaces ([Service],
// [TransformerPort]) for provider-agnostic image processing (VIPS,
// ImageMagick, etc.) and includes security validation for dimensions,
// pixel counts, and formats.
//
// # Security
//
// The package provides security validation to prevent resource
// exhaustion and enforce format restrictions:
//
//   - [ValidateImageDimensions]: Checks width and height against
//     configured limits
//   - [ValidateImagePixelCount]: Prevents excessive memory allocation
//     from large images
//   - [ValidateImageFormat]: Restricts output to allowed formats
//   - [LimitedReader]: Enforces maximum input size during streaming
//     reads
//
// # Thread safety
//
// The [Service] implementation is safe for concurrent use. Responsive
// variant generation runs transformations concurrently using
// goroutines.
package image_domain
