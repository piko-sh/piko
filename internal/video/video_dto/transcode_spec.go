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
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	// crfMinValue is the smallest allowed CRF (Constant Rate Factor) value.
	crfMinValue = 0

	// crfMaxValue is the upper limit for CRF in video encoding.
	crfMaxValue = 51
)

// TranscodeSpec defines the settings for a video transcoding operation.
type TranscodeSpec struct {
	// CRF is the Constant Rate Factor for video encoding quality.
	// Valid range is 0-51; lower values give better quality.
	CRF *int `json:"crf,omitempty"`

	// SegmentDuration is the number of seconds per segment for HLS or DASH output.
	// A nil value uses the default; must be positive if set.
	SegmentDuration *int `json:"segmentDuration,omitempty"`

	// Format specifies the container format (mp4, webm, mkv); empty uses InferFormat.
	Format string `json:"format,omitempty"`

	// Preset is the encoding speed preset (ultrafast, fast, medium, slow, veryslow).
	Preset string `json:"preset,omitempty"`

	// AudioCodec is the output audio codec, such as aac, opus, or vorbis.
	AudioCodec string `json:"audioCodec,omitempty"`

	// Codec is the output video codec (h264, h265, vp9, av1).
	Codec string `json:"codec"`

	// Profile specifies the codec profile (e.g. baseline, main, high).
	Profile string `json:"profile,omitempty"`

	// Level specifies the codec profile level (e.g. 3.0, 3.1, 4.0).
	Level string `json:"level,omitempty"`

	// Description is a short text that explains what this transcode does.
	Description string `json:"description,omitempty"`

	// Bitrate is the video bitrate in bits per second; must be non-negative.
	Bitrate int `json:"bitrate,omitempty"`

	// Framerate specifies the target frames per second; must be non-negative.
	Framerate float64 `json:"framerate,omitempty"`

	// Height is the output height in pixels; 0 means no scaling.
	Height int `json:"height,omitempty"`

	// AudioBitrate is the audio bitrate in bits per second; must be non-negative.
	AudioBitrate int `json:"audioBitrate,omitempty"`

	// Width is the output width in pixels; 0 means no resize.
	Width int `json:"width,omitempty"`
}

// InferFormat infers the output format from the codec if not explicitly set.
// Defaults to mp4 for h264/h265 and unknown codecs; uses webm for vp9/av1.
//
// Returns string which is the format to use for transcoding output.
func (s *TranscodeSpec) InferFormat() string {
	if s.Format != "" {
		return s.Format
	}

	switch s.Codec {
	case "vp9", "av1":
		return "webm"
	default:
		return "mp4"
	}
}

// InferAudioCodec infers the audio codec from the output format if not
// explicitly set. Uses opus for webm/mkv formats; defaults to aac for mp4 and
// other formats.
//
// Returns string which is the audio codec to use.
func (s *TranscodeSpec) InferAudioCodec() string {
	if s.AudioCodec != "" {
		return s.AudioCodec
	}

	format := s.InferFormat()
	switch format {
	case "webm", "mkv":
		return "opus"
	default:
		return "aac"
	}
}

var (
	// validCodecs is the set of supported video codecs.
	validCodecs = map[string]bool{
		"h264": true,
		"h265": true,
		"vp9":  true,
		"av1":  true,
	}

	// validPresets holds the allowed encoding preset names.
	validPresets = map[string]bool{
		"ultrafast": true,
		"fast":      true,
		"medium":    true,
		"slow":      true,
		"veryslow":  true,
	}
)

// Validate checks the transcode specification.
//
// Returns error when the codec, dimensions, rates, or encoding options are
// invalid.
func (s *TranscodeSpec) Validate() error {
	if err := s.validateCodec(); err != nil {
		return fmt.Errorf("validating transcode spec codec: %w", err)
	}
	if err := s.validateDimensions(); err != nil {
		return fmt.Errorf("validating transcode spec dimensions: %w", err)
	}
	if err := s.validateRates(); err != nil {
		return fmt.Errorf("validating transcode spec rates: %w", err)
	}
	return s.validateEncodingOptions()
}

// validateCodec checks that the codec is one of the supported values.
//
// Returns error when the codec is not h264, h265, vp9, or av1.
func (s *TranscodeSpec) validateCodec() error {
	if !validCodecs[s.Codec] {
		return fmt.Errorf("unsupported codec: %s (supported: h264, h265, vp9, av1)", s.Codec)
	}
	return nil
}

// validateDimensions checks that width and height are valid.
//
// Returns error when dimensions are negative or only one is set.
func (s *TranscodeSpec) validateDimensions() error {
	if s.Width < 0 || s.Height < 0 {
		return errors.New("invalid dimensions: width and height must be non-negative")
	}
	if (s.Width > 0 && s.Height == 0) || (s.Width == 0 && s.Height > 0) {
		return errors.New("invalid dimensions: both width and height must be specified together")
	}
	return nil
}

// validateRates checks that bitrate, framerate, and audio bitrate are valid.
//
// Returns error when any rate value is negative.
func (s *TranscodeSpec) validateRates() error {
	if s.Bitrate < 0 {
		return errors.New("invalid bitrate: must be non-negative")
	}
	if s.Framerate < 0 {
		return errors.New("invalid framerate: must be non-negative")
	}
	if s.AudioBitrate < 0 {
		return errors.New("invalid audio bitrate: must be non-negative")
	}
	return nil
}

// validateEncodingOptions checks that CRF, preset, and segment duration values
// are within valid ranges.
//
// Returns error when any encoding option is out of range or unsupported.
func (s *TranscodeSpec) validateEncodingOptions() error {
	if s.CRF != nil && (*s.CRF < crfMinValue || *s.CRF > crfMaxValue) {
		return fmt.Errorf("invalid CRF: must be between %d and %d", crfMinValue, crfMaxValue)
	}
	if s.Preset != "" && !validPresets[s.Preset] {
		return fmt.Errorf("invalid preset: %s (supported: ultrafast, fast, medium, slow, veryslow)", s.Preset)
	}
	if s.SegmentDuration != nil && *s.SegmentDuration <= 0 {
		return errors.New("invalid segment duration: must be positive")
	}
	return nil
}

// ParseTranscodeSpec parses capability parameters into a TranscodeSpec.
// The parameters are expected in the format used by the orchestrator
// capability system.
//
// Takes params (map[string]string) which contains the transcode parameters.
//
// Returns TranscodeSpec which contains the parsed transcode settings.
// Returns error when a required parameter is missing or a value is invalid.
func ParseTranscodeSpec(params map[string]string) (TranscodeSpec, error) {
	spec := TranscodeSpec{}

	codec := parseLowerStringParam(params, "codec")
	if codec == "" {
		return spec, errors.New("missing required parameter: codec")
	}
	spec.Codec = codec

	var err error
	if spec.Width, err = parseIntParam(params, "width"); err != nil {
		return spec, err
	}
	if spec.Height, err = parseIntParam(params, "height"); err != nil {
		return spec, err
	}
	if spec.Bitrate, err = parseIntParam(params, "bitrate"); err != nil {
		return spec, err
	}
	if spec.AudioBitrate, err = parseIntParam(params, "audioBitrate"); err != nil {
		return spec, err
	}

	if spec.CRF, err = parseIntPtrParam(params, "crf"); err != nil {
		return spec, err
	}
	if spec.SegmentDuration, err = parseIntPtrParam(params, "segmentDuration"); err != nil {
		return spec, err
	}

	if spec.Framerate, err = parseFloatParam(params, "framerate"); err != nil {
		return spec, err
	}

	spec.Preset = parseLowerStringParam(params, "preset")
	spec.AudioCodec = parseLowerStringParam(params, "audioCodec")
	spec.Format = parseLowerStringParam(params, "format")
	spec.Profile = parseLowerStringParam(params, "profile")

	if level, ok := params["level"]; ok {
		spec.Level = level
	}
	if description, ok := params["description"]; ok {
		spec.Description = description
	}

	return spec, nil
}

// parseIntParam extracts and parses an integer parameter from a params map.
// Returns 0 if the key is not present or empty.
//
// Takes params (map[string]string) which contains the parameter key-value pairs.
// Takes key (string) which specifies the parameter name to extract.
//
// Returns int which is the parsed integer value, or 0 if the key is missing.
// Returns error when the value cannot be parsed as an integer.
func parseIntParam(params map[string]string, key string) (int, error) {
	value, ok := params[key]
	if !ok || value == "" {
		return 0, nil
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: %w", key, err)
	}
	return result, nil
}

// parseIntPtrParam gets an integer pointer value from a map of parameters.
//
// Takes params (map[string]string) which holds the parameter values.
// Takes key (string) which is the name of the parameter to get.
//
// Returns *int which is nil if the key is missing or the value is empty.
// Returns error when the value cannot be read as an integer.
func parseIntPtrParam(params map[string]string, key string) (*int, error) {
	value, ok := params[key]
	if !ok || value == "" {
		return nil, nil
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s parameter: %w", key, err)
	}
	return &result, nil
}

// parseFloatParam extracts and parses a float parameter from the params map.
// Returns 0 if the key is not present or empty.
//
// Takes params (map[string]string) which contains the parameter key-value pairs.
// Takes key (string) which specifies the parameter name to look up.
//
// Returns float64 which is the parsed value, or 0 if the key is missing or empty.
// Returns error when the value cannot be parsed as a float.
func parseFloatParam(params map[string]string, key string) (float64, error) {
	value, ok := params[key]
	if !ok || value == "" {
		return 0, nil
	}
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter: %w", key, err)
	}
	return result, nil
}

// parseLowerStringParam retrieves a string value from a map and converts it
// to lowercase.
//
// Takes params (map[string]string) which holds the parameter values to search.
// Takes key (string) which is the name of the parameter to find.
//
// Returns string which is the lowercase value, or empty if the key is missing
// or has an empty value.
func parseLowerStringParam(params map[string]string, key string) string {
	if value, ok := params[key]; ok && value != "" {
		return strings.ToLower(value)
	}
	return ""
}
