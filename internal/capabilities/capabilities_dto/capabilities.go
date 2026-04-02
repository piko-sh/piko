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

package capabilities_dto

// Capability represents a named feature that can be checked and used.
// It implements fmt.Stringer.
type Capability string

const (
	// CapabilityCompressBrotli identifies Brotli compression support.
	CapabilityCompressBrotli Capability = "compress-brotli"

	// CapabilityCompressGzip identifies the gzip compression capability.
	CapabilityCompressGzip Capability = "compress-gzip"

	// CapabilityCompileComponent identifies the capability to compile components.
	CapabilityCompileComponent Capability = "compile-component"

	// CapabilityImageTransform is the capability for transforming images.
	CapabilityImageTransform Capability = "image-transform"

	// CapabilityMinifyCSS identifies the CSS minification capability.
	CapabilityMinifyCSS Capability = "minify-css"

	// CapabilityMinifyJS identifies the JavaScript minification capability.
	CapabilityMinifyJS Capability = "minify-js"

	// CapabilityMinifySVG identifies the SVG minification capability.
	CapabilityMinifySVG Capability = "minify-svg"

	// CapabilityCopyJS is a passthrough capability that copies JavaScript
	// without transformation, used to produce readable non-minified variants.
	CapabilityCopyJS Capability = "copy-js"

	// CapabilityVideoTranscode is the capability for converting video files
	// between different formats.
	CapabilityVideoTranscode Capability = "video-transcode"

	// CapabilityVideoThumbnail is the capability for extracting thumbnail images
	// from videos.
	CapabilityVideoThumbnail Capability = "video-thumbnail"
)

// String returns the capability as its underlying string value.
//
// Returns string which is the string form of the capability.
func (c Capability) String() string {
	return string(c)
}
