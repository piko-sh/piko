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

	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/internal/video/video_dto"
)

// VideoThumbnail creates a capability function that extracts thumbnails from
// video files. It depends on the video domain's Service to perform the
// extraction operation.
//
// Takes videoService (video_domain.Service) which provides the video processing
// backend.
//
// Returns capabilities_domain.CapabilityFunc which wraps the thumbnail
// extraction logic for use within the capabilities system.
//
// Supported Parameters:
//
// Timing:
//   - timestamp: Position in video to extract frame (e.g., "5s", "1:30", "0")
//   - time: Alias for timestamp
//
// Sizing:
//   - width: Target width in pixels (0 = auto from height)
//   - height: Target height in pixels (0 = auto from width)
//
// Output:
//   - format: Output image format (jpeg, png, webp). Default: jpeg
//   - quality: Compression quality 1-100 for lossy formats. Default: 85
func VideoThumbnail(videoService video_domain.Service) capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, params capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, "VideoThumbnailCapability")
		defer span.End()

		spec, err := buildThumbnailSpec(params)
		if err != nil {
			l.ReportError(span, err, "Invalid thumbnail parameters")
			return nil, fmt.Errorf("invalid thumbnail parameters: %w", err)
		}

		l.Trace("Extracting video thumbnail",
			logger_domain.Duration("timestamp", spec.Timestamp),
			logger_domain.String("format", spec.Format),
			logger_domain.Int("width", spec.Width),
			logger_domain.Int("height", spec.Height),
		)

		result, err := videoService.ExtractThumbnail(ctx, inputData, spec)
		if err != nil {
			l.ReportError(span, err, "Video service failed to extract thumbnail")
			return nil, fmt.Errorf("failed to extract thumbnail: %w", err)
		}

		return result, nil
	}
}

// buildThumbnailSpec creates a ThumbnailSpec from capability parameters.
//
// Takes params (CapabilityParams) which contains the capability parameters.
//
// Returns video_dto.ThumbnailSpec which contains the parsed spec.
// Returns error when parsing fails.
func buildThumbnailSpec(params capabilities_domain.CapabilityParams) (video_dto.ThumbnailSpec, error) {
	spec := video_dto.NewThumbnailSpec()

	if err := parseThumbnailTimestamp(params, &spec); err != nil {
		return spec, err
	}
	if err := parseThumbnailDimensions(params, &spec); err != nil {
		return spec, err
	}
	parseThumbnailOutputSettings(params, &spec)

	return spec, nil
}

// parseThumbnailTimestamp extracts and parses a timestamp from the given
// parameters.
//
// Takes params (CapabilityParams) which holds the capability settings.
// Takes spec (*ThumbnailSpec) which receives the parsed timestamp value.
//
// Returns error when the timestamp string cannot be parsed.
func parseThumbnailTimestamp(params capabilities_domain.CapabilityParams, spec *video_dto.ThumbnailSpec) error {
	ts := getFirstNonEmpty(params, "timestamp", "time")
	if ts == "" {
		return nil
	}
	duration, err := video_dto.ParseThumbnailTime(ts)
	if err != nil {
		return fmt.Errorf("parsing timestamp: %w", err)
	}
	spec.Timestamp = duration
	return nil
}

// parseThumbnailDimensions reads width and height values from params and stores
// them in spec.
//
// Takes params (CapabilityParams) which contains the width and height values.
// Takes spec (*ThumbnailSpec) which receives the parsed dimensions.
//
// Returns error when width or height cannot be parsed as a number.
func parseThumbnailDimensions(params capabilities_domain.CapabilityParams, spec *video_dto.ThumbnailSpec) error {
	if w, ok := params["width"]; ok && w != "" {
		width, err := strconv.Atoi(w)
		if err != nil {
			return fmt.Errorf("parsing width: %w", err)
		}
		spec.Width = width
	}
	if h, ok := params["height"]; ok && h != "" {
		height, err := strconv.Atoi(h)
		if err != nil {
			return fmt.Errorf("parsing height: %w", err)
		}
		spec.Height = height
	}
	return nil
}

// parseThumbnailOutputSettings reads format and quality from params and sets
// them on spec.
//
// Takes params (CapabilityParams) which holds the capability settings.
// Takes spec (*ThumbnailSpec) which receives the parsed output settings.
func parseThumbnailOutputSettings(params capabilities_domain.CapabilityParams, spec *video_dto.ThumbnailSpec) {
	if f, ok := params["format"]; ok && f != "" {
		spec.Format = f
	}
	if q, ok := params["quality"]; ok && q != "" {
		if quality, err := strconv.Atoi(q); err == nil {
			spec.Quality = quality
		}
	}
}

// getFirstNonEmpty returns the first non-empty value for the given keys.
//
// Takes params (CapabilityParams) which contains the parameters to search.
// Takes keys (...string) which are the parameter names to check in order.
//
// Returns string which is the first non-empty value found, or an empty string.
func getFirstNonEmpty(params capabilities_domain.CapabilityParams, keys ...string) string {
	for _, key := range keys {
		if value, ok := params[key]; ok && value != "" {
			return value
		}
	}
	return ""
}
