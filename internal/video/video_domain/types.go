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

package video_domain

import "time"

const (
	// defaultMaxVideoWidth is the maximum allowed video width in pixels, set to
	// 3840 for 4K resolution support.
	defaultMaxVideoWidth = 3840

	// defaultMaxVideoHeight is the maximum video height in pixels (2160 for 4K).
	defaultMaxVideoHeight = 2160

	// defaultMaxVideoPixels is the largest total pixel count allowed for video
	// processing (3840 x 2160 = 8,294,400 pixels, which is 4K resolution).
	defaultMaxVideoPixels = 8_294_400

	// defaultMaxFileSizeBytes is the largest allowed input file size (500 MB).
	defaultMaxFileSizeBytes = 500 * 1024 * 1024

	// defaultTranscodeTimeout is the maximum time allowed for a video transcode
	// operation, set to five minutes.
	defaultTranscodeTimeout = 5 * time.Minute

	// defaultMaxBitrate is the maximum allowed video bitrate in bits per second,
	// set to 20 Mbps.
	defaultMaxBitrate = 20_000_000

	// defaultMaxFramerate is the highest frame rate allowed, in frames per second.
	defaultMaxFramerate = 60.0
)

// codecProfile holds a codec encoding profile with its name and level.
type codecProfile struct {
	// Name is the profile name, such as "baseline", "main", or "high".
	Name string

	// Level is the profile level (for example, "3.0", "4.0", "5.1").
	Level string

	// Description is a human-readable explanation of this codec profile.
	Description string
}

// qualityPreset represents an encoding quality and speed preset.
type qualityPreset struct {
	// Name is the preset identifier such as "ultrafast", "medium", or "veryslow".
	Name string

	// Description is a short text that explains what this quality preset does.
	Description string

	// EncodingSpeed is a relative encoding speed where higher values mean
	// faster encoding.
	EncodingSpeed int

	// QualityScore is a relative quality score where higher values indicate
	// better quality.
	QualityScore int
}

var (
	// h264ProfileBaseline is the H.264 Baseline profile for maximum compatibility.
	h264ProfileBaseline = codecProfile{
		Name:        "baseline",
		Level:       "3.0",
		Description: "H.264 Baseline profile for maximum compatibility",
	}

	// h264ProfileMain is an H.264 codec profile for balanced compatibility and
	// quality.
	h264ProfileMain = codecProfile{
		Name:        "main",
		Level:       "4.0",
		Description: "H.264 Main profile for balanced compatibility and quality",
	}

	// h264ProfileHigh is the H.264 High profile preset for best quality encoding.
	h264ProfileHigh = codecProfile{
		Name:        "high",
		Level:       "4.2",
		Description: "H.264 High profile for best quality",
	}

	// h265ProfileMain is the H.265 Main profile at level 4.0.
	h265ProfileMain = codecProfile{
		Name:        "main",
		Level:       "4.0",
		Description: "H.265 Main profile",
	}

	// h265ProfileMain10 is the H.265 Main10 codec profile with 10-bit colour depth.
	h265ProfileMain10 = codecProfile{
		Name:        "main10",
		Level:       "5.0",
		Description: "H.265 Main10 profile (10-bit colour)",
	}

	// presetUltrafast is the fastest encoding preset with the lowest quality.
	presetUltrafast = qualityPreset{
		Name:          "ultrafast",
		EncodingSpeed: 100,
		QualityScore:  10,
		Description:   "Fastest encoding, lowest quality",
	}

	// presetFast is a quality preset that prioritises encoding speed over quality.
	presetFast = qualityPreset{
		Name:          "fast",
		EncodingSpeed: 75,
		QualityScore:  40,
		Description:   "Fast encoding, moderate quality",
	}

	// presetMedium is a balanced quality preset with moderate encoding speed and
	// quality.
	presetMedium = qualityPreset{
		Name:          "medium",
		EncodingSpeed: 50,
		QualityScore:  65,
		Description:   "Balanced encoding speed and quality",
	}

	// presetSlow is a quality preset that prioritises high quality over speed.
	presetSlow = qualityPreset{
		Name:          "slow",
		EncodingSpeed: 25,
		QualityScore:  85,
		Description:   "Slow encoding, high quality",
	}

	// presetVeryslow is the slowest encoding preset with the highest quality.
	presetVeryslow = qualityPreset{
		Name:          "veryslow",
		EncodingSpeed: 10,
		QualityScore:  95,
		Description:   "Slowest encoding, highest quality",
	}
)
