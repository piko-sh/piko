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

// Capability represents a named feature that can be checked and used. It implements
// fmt.Stringer.
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

	// CapabilityCopyJS is a passthrough capability that copies JavaScript without
	// transformation, used to produce readable non-minified variants.
	CapabilityCopyJS Capability = "copy-js"

	// CapabilityVideoTranscode is the capability for converting video files between
	// different formats.
	CapabilityVideoTranscode Capability = "video-transcode"

	// CapabilityVideoThumbnail is the capability for extracting thumbnail images from
	// videos.
	CapabilityVideoThumbnail Capability = "video-thumbnail"

	// CapabilityTranspileTypeScript identifies the TypeScript-to-JavaScript transpilation
	// capability.
	CapabilityTranspileTypeScript Capability = "transpile-typescript"
)

// String returns the capability as its underlying string value.
//
// Returns string which is the string form of the capability.
func (c Capability) String() string {
	return string(c)
}

var (
	// capabilityVersions records the output version of each transform capability.
	//
	// A version is bumped whenever a change makes a capability produce different bytes for
	// identical input: an encoder setting, an algorithm change, or a dependency upgrade that
	// alters output. Every derived variant produced by that capability then carries an older
	// version than the current one and is treated as stale, so it is regenerated on next
	// read.
	capabilityVersions = map[Capability]uint32{
		CapabilityCompressBrotli:      1,
		CapabilityCompressGzip:        1,
		CapabilityCompileComponent:    1,
		CapabilityImageTransform:      1,
		CapabilityMinifyCSS:           1,
		CapabilityMinifyJS:            1,
		CapabilityMinifySVG:           1,
		CapabilityCopyJS:              1,
		CapabilityVideoTranscode:      1,
		CapabilityVideoThumbnail:      1,
		CapabilityTranspileTypeScript: 1,
	}
)

// AllCapabilities lists every declared transform capability, for tests and iteration.
//
// Returns []Capability which is every capability the package defines.
func AllCapabilities() []Capability {
	return []Capability{
		CapabilityCompressBrotli,
		CapabilityCompressGzip,
		CapabilityCompileComponent,
		CapabilityImageTransform,
		CapabilityMinifyCSS,
		CapabilityMinifyJS,
		CapabilityMinifySVG,
		CapabilityCopyJS,
		CapabilityVideoTranscode,
		CapabilityVideoThumbnail,
		CapabilityTranspileTypeScript,
	}
}

// Version returns the output version of a capability, or zero if it is unknown.
//
// A zero result means the capability has no registered version, which a variant
// fingerprint must reject rather than silently accept, so an unversioned transform cannot
// masquerade as a current one.
//
// Takes c (Capability) which is the capability to look up.
//
// Returns uint32 which is the capability's output version, or zero when unknown.
func Version(c Capability) uint32 {
	return capabilityVersions[c]
}
