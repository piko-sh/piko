//go:build vips

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

// Package image_provider_vips implements the image transformer port
// using libvips via govips.
//
// libvips is a demand-driven, horizontally threaded image processing
// library that handles large images with low memory usage. This is
// the recommended provider for production deployments.
//
// Output formats: JPEG, PNG, WebP, AVIF, GIF.
//
// # Resource management
//
// The [Provider.Close] method must be called during application
// shutdown to release libvips resources.
//
// # Thread safety
//
// All methods are safe for concurrent use. The provider uses an
// internal semaphore to limit concurrent operations to
// [runtime.NumCPU], preventing unbounded resource consumption
// under load.
package image_provider_vips
