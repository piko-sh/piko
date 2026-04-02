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

package capabilities_functions

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/video/video_domain"
)

const (
	// defaultVideoPreset is the default encoding speed preset for video
	// transcoding.
	defaultVideoPreset = "medium"

	// defaultVideoCodec is the default video codec.
	defaultVideoCodec = "h264"

	// paramCodec is the parameter key for specifying the video codec.
	paramCodec = "codec"
)

// VideoTranscode creates a capability function that performs video transcoding.
// It depends on the video domain's Service to select the provider and execute
// the transcoding operation.
//
// Takes videoService (video_domain.Service) which provides the video transcoding
// backend.
//
// Returns capabilities_domain.CapabilityFunc which wraps the transcoding logic
// for use within the capabilities system.
//
// Supported parameters:
//
// Resolution and sizing:
//   - resolution: Target resolution (e.g. "1920x1080", "1280x720")
//   - width: Target width in pixels (integer)
//   - height: Target height in pixels (integer)
//
// Quality and encoding:
//   - bitrate: Target bitrate (e.g. "5000k", "2500000")
//   - crf: Constant Rate Factor 0-51 (lower means better quality)
//   - preset: Encoding preset (ultrafast, fast, medium, slow, veryslow)
//   - codec: Video codec (h264, h265, vp9, av1)
//   - profile: Codec profile (baseline, main, high)
//   - level: Codec level (e.g. "3.0", "4.0")
//
// Audio:
//   - audio_codec: Audio codec (aac, opus, vorbis)
//   - audio_bitrate: Audio bitrate in bits per second
//
// Streaming:
//   - segment_duration: HLS segment duration in seconds
//   - format: Output format (mp4, webm, mkv)
func VideoTranscode(videoService video_domain.Service) capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, params capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, "VideoTranscodeCapability")
		defer span.End()

		transcodeParams := buildVideoTranscodeParams(params)

		l.Trace("Executing video transcoding",
			logger_domain.String(paramCodec, transcodeParams[paramCodec]),
			logger_domain.String("resolution", transcodeParams["resolution"]),
			logger_domain.String("bitrate", transcodeParams["bitrate"]),
		)

		result, err := videoService.Transcode(ctx, inputData, transcodeParams)
		if err != nil {
			l.ReportError(span, err, "Video service failed to transcode")
			return nil, fmt.Errorf("failed to transcode video: %w", err)
		}

		return result, nil
	}
}

// buildVideoTranscodeParams builds a map of video transcoding settings from
// the given capability parameters.
//
// Takes params (CapabilityParams) which holds the input settings to convert.
//
// Returns map[string]string which holds the transcoding settings ready for use.
func buildVideoTranscodeParams(params capabilities_domain.CapabilityParams) map[string]string {
	result := make(map[string]string)

	if resolution, ok := params["resolution"]; ok && resolution != "" {
		width, height := parseResolution(resolution)
		if width > 0 {
			result["width"] = strconv.Itoa(width)
		}
		if height > 0 {
			result["height"] = strconv.Itoa(height)
		}
	}

	copyParamIfPresent(params, result, "width")
	copyParamIfPresent(params, result, "height")
	copyParamIfPresent(params, result, "bitrate")
	copyParamIfPresent(params, result, "crf")
	copyParamIfPresent(params, result, "preset")
	copyParamIfPresent(params, result, paramCodec)
	copyParamIfPresent(params, result, "profile")
	copyParamIfPresent(params, result, "level")
	copyParamIfPresent(params, result, "format")
	copyParamIfPresent(params, result, "audio_codec")
	copyParamIfPresent(params, result, "audio_bitrate")
	copyParamIfPresent(params, result, "segment_duration")
	copyParamIfPresent(params, result, "framerate")

	if _, ok := result[paramCodec]; !ok {
		result[paramCodec] = defaultVideoCodec
	}
	if _, ok := result["preset"]; !ok {
		result["preset"] = defaultVideoPreset
	}

	return result
}

// parseResolution parses a resolution string like "1920x1080" into width and
// height.
//
// Takes resolution (string) which is the resolution string to parse.
//
// Returns width (int) which is the width in pixels.
// Returns height (int) which is the height in pixels.
func parseResolution(resolution string) (width int, height int) {
	resolution = strings.ToLower(strings.TrimSpace(resolution))
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return 0, 0
	}

	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0
	}

	height, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0
	}

	return width, height
}

// copyParamIfPresent copies a parameter to dest if it exists and is non-empty.
//
// Takes source (CapabilityParams) which contains the source parameters.
// Takes destination (map[string]string) which receives the copied parameter.
// Takes key (string) which identifies the parameter to copy.
func copyParamIfPresent(source capabilities_domain.CapabilityParams, destination map[string]string, key string) {
	if value, ok := source[key]; ok && value != "" {
		destination[key] = value
	}
}
