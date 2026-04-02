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

// Package capabilities_functions supplies built-in capability
// implementations for content transformation and processing.
//
// Each factory function returns a capabilities_domain.CapabilityFunc
// that processes streaming data via io.Reader and returns transformed
// output. Built-in capabilities cover compression (gzip, Brotli),
// minification (CSS, JavaScript, SVG), media processing (image
// transforms, video thumbnails, video transcoding), and component
// compilation.
//
// # Usage
//
// Capabilities are registered with the capability service during
// initialisation:
//
//	service.Register("gzip", capabilities_functions.Gzip())
//	service.Register("minify-css", capabilities_functions.MinifyCSS())
//	service.Register("image-transform",
//	    capabilities_functions.ImageTransform(imageService))
//
// # Thread safety
//
// All factory functions return stateless CapabilityFunc instances that
// are safe for concurrent use. Compression capabilities use sync.Pool
// for writer reuse to reduce allocations under high load.
package capabilities_functions
