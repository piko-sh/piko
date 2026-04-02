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

package config

// AssetsConfig holds settings for handling static files such as images and
// videos.
type AssetsConfig struct {
	// Image holds settings for image assets, including profiles, default
	// densities, and screen breakpoints for responsive images.
	Image ImageAssetsConfig `json:"image" yaml:"image"`

	// Video holds settings for video asset processing, including quality
	// presets, segment duration, and reusable profiles.
	Video VideoAssetsConfig `json:"video" yaml:"video"`
}

// ImageAssetsConfig holds image-specific asset configuration.
type ImageAssetsConfig struct {
	// Profiles maps profile names to lists of transformation steps.
	// Each profile can be used in <piko:img> and <pml-img> tags via the
	// profile attribute.
	Profiles map[string][]AssetTransformationStep `json:"profiles" yaml:"profiles"`

	// Screens maps breakpoint names to viewport widths in pixels
	// for responsive image calculation. Used when parsing the
	// sizes attribute (e.g., "sm:50vw").
	Screens map[string]int `json:"screens" yaml:"screens"`

	// DefaultDensities specifies the pixel density multipliers
	// for responsive images when densities are not explicitly
	// set. Default is ["x1", "x2"].
	DefaultDensities []string `json:"defaultDensities" yaml:"defaultDensities" default:"x1,x2"`
}

// VideoAssetsConfig holds video-specific asset configuration.
type VideoAssetsConfig struct {
	// Profiles maps profile names to lists of video transformation steps.
	// Each profile can be used by name in <piko:video> tags with the profile
	// attribute.
	Profiles map[string][]AssetTransformationStep `json:"profiles" yaml:"profiles"`

	// DefaultQualities is the default list of quality levels to
	// generate when a <piko:video> tag does not specify
	// qualities. Common values are "1080p", "720p", and "480p"
	// for adaptive bitrate streaming.
	DefaultQualities []string `json:"defaultQualities" yaml:"defaultQualities" default:"1080p,720p,480p"`

	// DefaultSegmentDuration is the default HLS segment duration
	// in seconds. Shorter segments allow faster quality
	// adaptation but increase overhead.
	DefaultSegmentDuration int `json:"defaultSegmentDuration" yaml:"defaultSegmentDuration" default:"10"`
}

// AssetTransformationStep represents a single transformation to
// apply to an asset.
// Multiple steps can be chained together in a profile.
type AssetTransformationStep struct {
	// Params holds the transformation parameters as key-value pairs,
	// where typical keys for image-transform include width, height,
	// format, quality, and fit, and typical keys for video-transcode
	// include resolution, bitrate, and segment_duration.
	Params map[string]string `json:"params" yaml:"params"`

	// Capability identifies the transformation type
	// (e.g., "image-transform", "video-transcode").
	Capability string `json:"capability" yaml:"capability"`
}
