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

package video_dto

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// defaultThumbnailFormat is the default image format for thumbnails.
	defaultThumbnailFormat = "jpeg"

	// defaultThumbnailQuality is the default image quality for lossy formats.
	defaultThumbnailQuality = 85

	// maxThumbnailQuality is the maximum allowed quality value.
	maxThumbnailQuality = 100

	// timePartsMMSS is the number of parts when time is in MM:SS format.
	timePartsMMSS = 2

	// timePartsHHMMSS is the number of parts in HH:MM:SS format.
	timePartsHHMMSS = 3
)

// ThumbnailSpec defines the settings for extracting a thumbnail from a video.
type ThumbnailSpec struct {
	// Format specifies the output image format; valid values are jpeg, png, webp.
	Format string `json:"format,omitempty"`

	// Timestamp is the position in the video to extract the frame from.
	// Zero extracts the first frame.
	Timestamp time.Duration `json:"timestamp,omitempty"`

	// Width is the output width in pixels. Zero maintains aspect
	// ratio based on Height.
	Width int `json:"width,omitempty"`

	// Height is the output height in pixels. Zero maintains aspect
	// ratio based on Width.
	Height int `json:"height,omitempty"`

	// Quality specifies the compression quality for lossy formats (1-100).
	// Zero uses the default quality; values outside 0-100 are invalid.
	Quality int `json:"quality,omitempty"`
}

// NewThumbnailSpec creates a ThumbnailSpec with default values.
//
// Returns ThumbnailSpec which is set up with sensible defaults.
func NewThumbnailSpec() ThumbnailSpec {
	return ThumbnailSpec{
		Format:  defaultThumbnailFormat,
		Quality: defaultThumbnailQuality,
	}
}

// Validate reports whether the spec has valid values.
//
// Returns error when any field has an invalid value.
func (s *ThumbnailSpec) Validate() error {
	if s.Timestamp < 0 {
		return fmt.Errorf("timestamp cannot be negative: %v", s.Timestamp)
	}
	if s.Width < 0 {
		return fmt.Errorf("width cannot be negative: %d", s.Width)
	}
	if s.Height < 0 {
		return fmt.Errorf("height cannot be negative: %d", s.Height)
	}
	if s.Quality < 0 || s.Quality > maxThumbnailQuality {
		return fmt.Errorf("quality must be between 0 and %d: %d", maxThumbnailQuality, s.Quality)
	}

	validFormats := map[string]bool{"jpeg": true, "jpg": true, "png": true, "webp": true}
	format := strings.ToLower(s.Format)
	if format != "" && !validFormats[format] {
		return fmt.Errorf("unsupported format: %s", s.Format)
	}

	return nil
}

// WithDefaults returns a copy of the spec with default values for any empty
// fields.
//
// Returns ThumbnailSpec which is a copy with defaults set for format and
// quality.
func (s ThumbnailSpec) WithDefaults() ThumbnailSpec {
	result := s
	if result.Format == "" {
		result.Format = defaultThumbnailFormat
	}
	if result.Quality == 0 {
		result.Quality = defaultThumbnailQuality
	}
	return result
}

// ParseThumbnailTime parses a time string into a Duration.
// Supports formats like "5s", "1m30s", "0.5s", "1:30", "01:30.5".
//
// Takes timeString (string) which is the time string to parse.
//
// Returns time.Duration which is the parsed duration.
// Returns error when the format is not valid.
func ParseThumbnailTime(timeString string) (time.Duration, error) {
	if timeString == "" {
		return 0, nil
	}

	if d, err := time.ParseDuration(timeString); err == nil {
		return d, nil
	}

	parts := strings.Split(timeString, ":")
	switch len(parts) {
	case timePartsMMSS:
		return parseMMSS(parts[0], parts[1])
	case timePartsHHMMSS:
		return parseHHMMSS(parts[0], parts[1], parts[2])
	}

	if seconds, err := strconv.ParseFloat(timeString, 64); err == nil {
		return time.Duration(seconds * float64(time.Second)), nil
	}

	return 0, fmt.Errorf("invalid time format: %s", timeString)
}

// parseMMSS parses MM:SS format into a duration.
//
// Takes minutesString (string) which is the minutes component to parse.
// Takes secondsString (string) which is the seconds component to parse.
//
// Returns time.Duration which is the combined duration from minutes and seconds.
// Returns error when minutesString is not a valid integer or
// secondsString is not a valid float.
func parseMMSS(minutesString, secondsString string) (time.Duration, error) {
	minutes, err := strconv.Atoi(minutesString)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %s", minutesString)
	}
	seconds, err := strconv.ParseFloat(secondsString, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %s", secondsString)
	}
	return time.Duration(minutes)*time.Minute + time.Duration(seconds*float64(time.Second)), nil
}

// parseHHMMSS parses HH:MM:SS format into a duration.
//
// Takes hoursString (string) which is the hours component.
// Takes minutesString (string) which is the minutes component.
// Takes secondsString (string) which is the seconds component (may
// include decimals).
//
// Returns time.Duration which is the total duration.
// Returns error when any component cannot be parsed as a number.
func parseHHMMSS(hoursString, minutesString, secondsString string) (time.Duration, error) {
	hours, err := strconv.Atoi(hoursString)
	if err != nil {
		return 0, fmt.Errorf("invalid hours: %s", hoursString)
	}
	minutes, err := strconv.Atoi(minutesString)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %s", minutesString)
	}
	seconds, err := strconv.ParseFloat(secondsString, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %s", secondsString)
	}
	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds*float64(time.Second)), nil
}
