//go:build ffmpeg

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

package video_provider_astiav

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/asticode/go-astiav"
	"piko.sh/piko/wdk/media"
)

const (
	// presetUltrafast selects the fastest encoding speed.
	presetUltrafast = "ultrafast"

	// presetFast selects a fast encoding speed.
	presetFast = "fast"

	// presetMedium selects a balanced encoding speed.
	presetMedium = "medium"

	// presetSlow selects a slow encoding speed for better
	// compression.
	presetSlow = "slow"

	// presetVeryslow selects the slowest encoding speed for
	// maximum compression.
	presetVeryslow = "veryslow"

	// profileMain selects the main codec profile.
	profileMain = "main"

	// profileMain10 selects the main10 codec profile with
	// 10-bit colour depth.
	profileMain10 = "main10"

	// defaultH264Bitrate is the default bitrate in bits per
	// second for H.264 encoding.
	defaultH264Bitrate = 2000000

	// defaultH264CRF is the default Constant Rate Factor for
	// H.264 encoding.
	defaultH264CRF = 23

	// defaultH265Bitrate is the default bitrate in bits per
	// second for H.265 encoding.
	defaultH265Bitrate = 1500000

	// defaultH265CRF is the default Constant Rate Factor for
	// H.265 encoding.
	defaultH265CRF = 28

	// defaultVP9Bitrate is the default bitrate in bits per
	// second for VP9 encoding.
	defaultVP9Bitrate = 1500000

	// defaultVP9CRF is the default Constant Rate Factor for
	// VP9 encoding.
	defaultVP9CRF = 31

	// defaultAudioBitrate is the default audio bitrate in bits
	// per second.
	defaultAudioBitrate = 128000

	// maxCRFValue is the maximum allowed Constant Rate Factor
	// value.
	maxCRFValue = 51

	// cpuUsedUltrafast is the VP9 cpu-used value for the ultrafast preset.
	cpuUsedUltrafast = 8

	// cpuUsedFast is the VP9 cpu-used value for the fast preset.
	cpuUsedFast = 5

	// cpuUsedMedium is the VP9 cpu-used value for the medium preset (also default).
	cpuUsedMedium = 2

	// defaultGOPSize is the default Group of Pictures size for video encoding.
	defaultGOPSize = 60

	// ioContextBufferSize is the buffer size for FFmpeg IO contexts.
	ioContextBufferSize = 32768

	// defaultMaxConcurrent is the default maximum number of concurrent
	// transcode operations.
	defaultMaxConcurrent = 10
)

// CodecConfig holds settings for a video codec, including encoder details,
// default quality options, and supported profiles and presets.
type CodecConfig struct {
	// Name is the codec name (e.g. "h264", "h265", "vp9").
	Name string

	// EncoderName is the FFmpeg encoder name
	// (e.g. "libx264", "libx265", "libvpx-vp9").
	EncoderName string

	// DefaultPreset is the encoding preset used when none is given.
	DefaultPreset string

	// DefaultProfile is the codec profile used when none is specified.
	DefaultProfile string

	// DefaultLevel is the default compression level for the codec.
	DefaultLevel string

	// SupportedProfiles lists the profiles that this codec supports.
	SupportedProfiles []string

	// SupportedPresets lists the presets that this codec supports.
	SupportedPresets []string

	// PixelFormat specifies the pixel format to use for encoding or decoding.
	PixelFormat astiav.PixelFormat

	// DefaultBitrate is the default bitrate in bits per second.
	DefaultBitrate int

	// DefaultCRF is the default Constant Rate Factor for video encoding.
	DefaultCRF int
}

// CodecRegistry holds a set of codec settings, stored by name.
type CodecRegistry struct {
	// codecs maps MIME types to their codec settings.
	codecs map[string]*CodecConfig
}

// NewCodecRegistry creates a new codec registry with default settings.
//
// Returns *CodecRegistry which contains pre-set values for h264, h265, and vp9
// codecs.
func NewCodecRegistry() *CodecRegistry {
	allPresets := []string{presetUltrafast, presetFast, presetMedium, presetSlow, presetVeryslow}

	return &CodecRegistry{
		codecs: map[string]*CodecConfig{
			"h264": {
				Name:              "h264",
				EncoderName:       "libx264",
				DefaultBitrate:    defaultH264Bitrate,
				DefaultCRF:        defaultH264CRF,
				DefaultPreset:     presetMedium,
				DefaultProfile:    profileMain,
				DefaultLevel:      "4.0",
				SupportedProfiles: []string{"baseline", profileMain, "high"},
				SupportedPresets:  allPresets,
				PixelFormat:       astiav.PixelFormatYuv420P,
			},
			"h265": {
				Name:              "h265",
				EncoderName:       "libx265",
				DefaultBitrate:    defaultH265Bitrate,
				DefaultCRF:        defaultH265CRF,
				DefaultPreset:     presetMedium,
				DefaultProfile:    profileMain,
				DefaultLevel:      "4.0",
				SupportedProfiles: []string{profileMain, profileMain10},
				SupportedPresets:  allPresets,
				PixelFormat:       astiav.PixelFormatYuv420P,
			},
			"vp9": {
				Name:              "vp9",
				EncoderName:       "libvpx-vp9",
				DefaultBitrate:    defaultVP9Bitrate,
				DefaultCRF:        defaultVP9CRF,
				DefaultPreset:     presetMedium,
				DefaultProfile:    "0",
				DefaultLevel:      "",
				SupportedProfiles: []string{"0", "1", "2", "3"},
				SupportedPresets:  allPresets,
				PixelFormat:       astiav.PixelFormatYuv420P,
			},
		},
	}
}

// GetCodec retrieves codec configuration by name.
//
// Takes name (string) which specifies the codec to retrieve.
//
// Returns *CodecConfig which contains the codec settings.
// Returns error when the codec name is not registered.
func (r *CodecRegistry) GetCodec(name string) (*CodecConfig, error) {
	config, ok := r.codecs[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", media.ErrUnsupportedCodec, name)
	}
	return config, nil
}

// SupportsCodec checks if a codec is supported.
//
// Takes name (string) which is the codec name to check.
//
// Returns bool which is true if the codec is registered.
func (r *CodecRegistry) SupportsCodec(name string) bool {
	_, ok := r.codecs[name]
	return ok
}

// SupportedCodecs returns a list of all supported codec names.
//
// Returns []string which contains the names of all registered codecs.
func (r *CodecRegistry) SupportedCodecs() []string {
	codecs := make([]string, 0, len(r.codecs))
	for name := range r.codecs {
		codecs = append(codecs, name)
	}
	return codecs
}

// ApplyDefaults applies default values to a transcode spec based on codec
// configuration.
//
// Takes spec (*TranscodeSpec) which is the transcode specification
// to populate with defaults.
//
// Returns error when the codec specified in the spec is not registered.
func (r *CodecRegistry) ApplyDefaults(spec *media.TranscodeSpec) error {
	config, err := r.GetCodec(spec.Codec)
	if err != nil {
		return err
	}

	if spec.Bitrate == 0 {
		spec.Bitrate = config.DefaultBitrate
	}

	if spec.CRF == nil {
		crf := config.DefaultCRF
		spec.CRF = &crf
	}

	if spec.Preset == "" {
		spec.Preset = config.DefaultPreset
	}

	if spec.Profile == "" {
		spec.Profile = config.DefaultProfile
	}

	if spec.Level == "" && config.DefaultLevel != "" {
		spec.Level = config.DefaultLevel
	}

	if spec.Format == "" {
		spec.Format = spec.InferFormat()
	}

	if spec.AudioCodec == "" {
		spec.AudioCodec = spec.InferAudioCodec()
	}

	if spec.AudioBitrate == 0 {
		spec.AudioBitrate = defaultAudioBitrate
	}

	return nil
}

// ValidateSpec validates a transcode spec against codec requirements.
//
// Takes spec (*TranscodeSpec) which contains the codec, profile,
// preset, and CRF settings to validate.
//
// Returns error when the codec is not registered, or when the profile, preset,
// or CRF value is not supported by the codec.
func (r *CodecRegistry) ValidateSpec(spec *media.TranscodeSpec) error {
	config, err := r.GetCodec(spec.Codec)
	if err != nil {
		return err
	}

	if spec.Profile != "" {
		if !slices.Contains(config.SupportedProfiles, spec.Profile) {
			return fmt.Errorf("unsupported profile '%s' for codec %s (supported: %v)",
				spec.Profile, spec.Codec, config.SupportedProfiles)
		}
	}

	if spec.Preset != "" {
		if !slices.Contains(config.SupportedPresets, spec.Preset) {
			return fmt.Errorf("unsupported preset '%s' for codec %s (supported: %v)",
				spec.Preset, spec.Codec, config.SupportedPresets)
		}
	}

	if spec.CRF != nil {
		if *spec.CRF < 0 || *spec.CRF > maxCRFValue {
			return fmt.Errorf("CRF must be between 0 and %d, got %d", maxCRFValue, *spec.CRF)
		}
	}

	return nil
}

// GetEncoderOptions builds encoder-specific options for FFmpeg.
//
// Takes spec (*TranscodeSpec) which specifies the transcoding
// settings including codec, preset, profile, and quality parameters.
//
// Returns map[string]string which contains the FFmpeg encoder options.
// Returns error when the specified codec is not found in the registry.
func (r *CodecRegistry) GetEncoderOptions(spec *media.TranscodeSpec) (map[string]string, error) {
	_, err := r.GetCodec(spec.Codec)
	if err != nil {
		return nil, err
	}

	switch spec.Codec {
	case "h264":
		return h264Options(spec), nil
	case "h265":
		return h265Options(spec), nil
	case "vp9":
		return vp9Options(spec), nil
	default:
		return make(map[string]string), nil
	}
}

// formatCRF returns the CRF value as a string for use in
// encoder options.
//
// Takes crf (int) which is the Constant Rate Factor value.
//
// Returns string which is the CRF value as a decimal string.
func formatCRF(crf int) string {
	return strconv.Itoa(crf)
}

// h264Options returns FFmpeg encoder options for the H.264
// codec.
//
// Takes spec (*media.TranscodeSpec) which provides the
// encoding parameters.
//
// Returns map[string]string which contains the H.264 encoder
// options.
func h264Options(spec *media.TranscodeSpec) map[string]string {
	options := make(map[string]string)
	if spec.Preset != "" {
		options["preset"] = spec.Preset
	}
	if spec.Profile != "" {
		options["profile"] = spec.Profile
	}
	if spec.Level != "" {
		options["level"] = spec.Level
	}
	if spec.CRF != nil {
		options["crf"] = formatCRF(*spec.CRF)
	}
	options["tune"] = "film"
	options["movflags"] = "+faststart"
	return options
}

// h265Options returns FFmpeg encoder options for the
// H.265/HEVC codec.
//
// Takes spec (*media.TranscodeSpec) which provides the
// encoding parameters.
//
// Returns map[string]string which contains the H.265 encoder
// options.
func h265Options(spec *media.TranscodeSpec) map[string]string {
	options := make(map[string]string)
	if spec.Preset != "" {
		options["preset"] = spec.Preset
	}
	if spec.Profile != "" {
		switch spec.Profile {
		case profileMain:
			options["profile"] = profileMain
		case profileMain10:
			options["profile"] = profileMain10
		}
	}
	if spec.CRF != nil {
		options["crf"] = formatCRF(*spec.CRF)
	}
	options["x265-params"] = "log-level=error"
	return options
}

// vp9Options returns FFmpeg encoder options for the VP9
// codec.
//
// Takes spec (*media.TranscodeSpec) which provides the
// encoding parameters.
//
// Returns map[string]string which contains the VP9 encoder
// options.
func vp9Options(spec *media.TranscodeSpec) map[string]string {
	options := make(map[string]string)
	if spec.Preset != "" {
		cpuUsed := presetToCPUUsed(spec.Preset)
		options["cpu-used"] = strconv.Itoa(cpuUsed)
	}
	if spec.CRF != nil {
		options["crf"] = formatCRF(*spec.CRF)
	}
	options["b:v"] = "0"
	options["row-mt"] = "1"
	options["tile-columns"] = "2"
	return options
}

// presetToCPUUsed maps preset names to VP9 cpu-used values, where lower
// cpu-used means slower encoding but better quality.
//
// Takes preset (string) which is the encoding speed preset name.
//
// Returns int which is the cpu-used value for VP9 encoding.
func presetToCPUUsed(preset string) int {
	switch preset {
	case presetUltrafast:
		return cpuUsedUltrafast
	case presetFast:
		return cpuUsedFast
	case presetSlow:
		return 1
	case presetVeryslow:
		return 0
	default:

		return cpuUsedMedium
	}
}
