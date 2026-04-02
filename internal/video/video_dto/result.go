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
	"io"
	"time"
)

// VideoCapabilities holds the properties and supported output options for a
// video file. It is returned by ExtractCapabilities and describes what the
// input video contains and what formats can be produced from it.
type VideoCapabilities struct {
	// Format is the container format (e.g. "mp4", "webm", "mkv").
	Format string `json:"format"`

	// AudioCodec is the audio codec used in the video file.
	AudioCodec string `json:"audioCodec"`

	// VideoCodec is the codec used for the video stream.
	VideoCodec string `json:"videoCodec"`

	// SupportedCodecs lists the video codecs that can be used for output.
	SupportedCodecs []string `json:"supportedCodecs"`

	// SupportedFormats lists the output formats that are available.
	SupportedFormats []string `json:"supportedFormats"`

	// Width is the video width in pixels.
	Width int `json:"width"`

	// Height is the video height in pixels.
	Height int `json:"height"`

	// Framerate is the number of frames per second.
	Framerate float64 `json:"framerate"`

	// FileSize is the size of the video file in bytes.
	FileSize int64 `json:"fileSize"`

	// Bitrate is the average bitrate in bits per second.
	Bitrate int64 `json:"bitrate"`

	// Duration is the total length of the video.
	Duration time.Duration `json:"duration"`

	// SupportsHLS indicates whether HLS streaming is supported.
	SupportsHLS bool `json:"supportsHLS"`

	// SupportsDASH indicates whether DASH streaming is supported.
	SupportsDASH bool `json:"supportsDASH"`
}

// HLSSpec holds the settings for HLS (HTTP Live Streaming) output.
type HLSSpec struct {
	// PlaylistType specifies the HLS playlist type: VOD or EVENT.
	PlaylistType string `json:"playlistType"`

	// VariantBitrates contains multiple bitrate ladders for adaptive bitrate
	// streaming.
	VariantBitrates []int `json:"variantBitrates"`

	// TranscodeSpec contains the base transcode settings
	TranscodeSpec

	// SegmentDuration is the length of each segment in seconds; defaults to 6.
	SegmentDuration int `json:"segmentDuration"`

	// IncludeIFramePlaylist indicates whether to include an I-frame only playlist.
	IncludeIFramePlaylist bool `json:"includeIFramePlaylist"`
}

// HLSResult represents the output of an HLS generation operation.
// It contains the master playlist, variant playlists, and video segments.
type HLSResult struct {
	// MasterPlaylist is the main m3u8 file that points to variant playlists.
	MasterPlaylist io.Reader `json:"-"`

	// MasterPlaylistContent is the text content of the master playlist.
	MasterPlaylistContent string `json:"masterPlaylistContent"`

	// Variants holds the list of bitrate variants from the transcode.
	Variants []HLSVariant `json:"variants"`

	// TotalSegments is the total number of segments across all variants.
	TotalSegments int `json:"totalSegments"`

	// TotalDuration is the total length of the video.
	TotalDuration time.Duration `json:"totalDuration"`
}

// HLSVariant represents a single bitrate variant in an HLS stream.
type HLSVariant struct {
	// Playlist is the m3u8 file specific to this HLS variant.
	Playlist io.Reader `json:"-"`

	// PlaylistContent is the text content of the variant playlist.
	PlaylistContent string `json:"playlistContent"`

	// Resolution is the video resolution for this variant (e.g. "1920x1080").
	Resolution string `json:"resolution"`

	// Codec is the video codec used for this variant.
	Codec string `json:"codec"`

	// Segments holds all video segments for this variant.
	Segments []HLSSegment `json:"segments"`

	// Bitrate is the target bitrate for this variant in bits per second.
	Bitrate int `json:"bitrate"`
}

// HLSSegment represents a single video segment in an HLS stream.
type HLSSegment struct {
	// Data provides the segment content as a readable stream in .ts file format.
	Data io.Reader `json:"-"`

	// StorageKey is the blob storage key for this segment.
	StorageKey string `json:"storageKey,omitempty"`

	// SequenceNumber is the segment's position in the stream.
	SequenceNumber int `json:"sequenceNumber"`

	// Duration is the segment length in seconds.
	Duration float64 `json:"duration"`

	// SizeBytes is the segment size in bytes.
	SizeBytes int64 `json:"sizeBytes"`
}

// DASHSpec defines the parameters for DASH (Dynamic Adaptive Streaming over
// HTTP) generation.
type DASHSpec struct {
	// ProfileType is the DASH profile; either "live" or "on-demand".
	ProfileType string `json:"profileType"`

	// VariantBitrates contains multiple bitrate ladders for adaptive bitrate
	// streaming.
	VariantBitrates []int `json:"variantBitrates"`

	// TranscodeSpec contains the base transcode settings
	TranscodeSpec

	// SegmentDuration is the length of each segment in seconds; default is 4.
	SegmentDuration int `json:"segmentDuration"`
}

// DASHResult holds the output from a DASH generation operation.
type DASHResult struct {
	// MPDManifest is the Media Presentation Description (MPD) file content.
	MPDManifest io.Reader `json:"-"`

	// MPDContent is the text content of the MPD manifest.
	MPDContent string `json:"mpdContent"`

	// Representations holds all bitrate options for this DASH stream.
	Representations []DASHRepresentation `json:"representations"`

	// TotalSegments is the total number of segments across all representations.
	TotalSegments int `json:"totalSegments"`

	// TotalDuration is the total length of the video.
	TotalDuration time.Duration `json:"totalDuration"`
}

// DASHRepresentation represents a single bitrate representation in a DASH
// stream.
type DASHRepresentation struct {
	// Resolution is the video resolution in width by height format (e.g.
	// "1920x1080").
	Resolution string `json:"resolution"`

	// Codec is the video codec used for this representation.
	Codec string `json:"codec"`

	// RepresentationID is the unique identifier for this representation.
	RepresentationID string `json:"representationId"`

	// Segments holds the video segments for this representation.
	Segments []DASHSegment `json:"segments"`

	// Bitrate is the target bitrate for this representation in bits per second.
	Bitrate int `json:"bitrate"`
}

// DASHSegment represents a single video segment in a DASH stream.
type DASHSegment struct {
	// Data provides the segment content as a stream of bytes.
	Data io.Reader `json:"-"`

	// StorageKey is the blob storage key for this segment.
	StorageKey string `json:"storageKey,omitempty"`

	// SequenceNumber is the segment's position in the stream.
	SequenceNumber int `json:"sequenceNumber"`

	// Duration is the segment length in seconds.
	Duration float64 `json:"duration"`

	// SizeBytes is the segment size in bytes.
	SizeBytes int64 `json:"sizeBytes"`
}

// TranscodeResult represents the result of a basic transcode operation.
type TranscodeResult struct {
	// Output is the transcoded video data as a readable stream.
	Output io.ReadCloser `json:"-"`

	// Codec is the output codec used for transcoding.
	Codec string `json:"codec"`

	// Format is the output format of the transcoded media.
	Format string `json:"format"`

	// OutputSize is the size of the transcoded video in bytes.
	OutputSize int64 `json:"outputSize,omitempty"`

	// Duration is the time taken to complete the transcoding.
	Duration time.Duration `json:"duration"`

	// FramesProcessed is the number of frames that were transcoded.
	FramesProcessed int64 `json:"framesProcessed"`

	// AverageFPS is the average transcoding speed in frames per second.
	AverageFPS float64 `json:"averageFps"`
}
