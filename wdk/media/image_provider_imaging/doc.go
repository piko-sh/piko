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

// Package image_provider_imaging implements the image transformer port
// using pure Go libraries (disintegration/imaging and nativewebp).
//
// This provider requires no CGO or external C libraries, making it
// straightforward to set up for development and testing. For
// production use, the vips provider is recommended as it provides
// lower memory usage, lossy WebP (VP8), AVIF support, and faster
// encoding.
//
// The pure Go nativewebp encoder uses lossless encoding (VP8L),
// which requires holding the entire image in memory multiple times
// and can consume 100-500MB per image depending on resolution.
//
// Output formats: JPEG, PNG, WebP (lossless only), GIF.
//
// # Thread safety
//
// All methods are safe for concurrent use. Concurrent transforms
// are limited via a semaphore to prevent memory exhaustion from
// the WebP encoder's large temporary buffer allocations.
package image_provider_imaging
