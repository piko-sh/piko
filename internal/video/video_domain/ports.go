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

import (
	"context"
	"io"

	"piko.sh/piko/internal/video/video_dto"
)

// TranscoderPort defines the interface for video transcoding implementations.
// This port follows the hexagonal architecture pattern, allowing different
// transcoding backends (FFmpeg, cloud services, etc.) to be swapped without
// affecting the domain logic.
type TranscoderPort interface {
	// Transcode converts video from one format or codec to another using the
	// provided specification.
	//
	// Takes input (io.Reader) which provides the source video as a stream.
	// Takes spec (video_dto.TranscodeSpec) which defines the target format and
	// codec settings.
	//
	// Returns io.ReadCloser which streams the transcoded output. The caller must
	// close this when finished.
	// Returns error when transcoding fails.
	Transcode(ctx context.Context, input io.Reader, spec video_dto.TranscodeSpec) (io.ReadCloser, error)

	// ExtractCapabilities analyses an input video stream and returns metadata
	// including detected codecs, resolution, aspect ratio, duration, bitrate, and
	// supported output formats.
	//
	// Takes input (io.Reader) which provides the video data to analyse.
	//
	// Returns video_dto.VideoCapabilities which contains the extracted metadata.
	// Returns error when the video cannot be analysed.
	ExtractCapabilities(ctx context.Context, input io.Reader) (video_dto.VideoCapabilities, error)

	// SupportsCodec checks if the given codec is supported by this transcoder.
	//
	// Takes codec (string) which is the codec name to check, such as "h264",
	// "h265", "vp9", or "av1".
	//
	// Returns bool which is true if the codec can be used as an output format.
	SupportsCodec(codec string) bool

	// ExtractThumbnail extracts a single frame from a video at the specified
	// position and encodes it as an image.
	//
	// Takes input (io.Reader) which provides the source video data.
	// Takes spec (video_dto.ThumbnailSpec) which defines the extraction parameters
	// including timestamp, dimensions, format, and quality.
	//
	// Returns io.ReadCloser which streams the encoded image data. The caller must
	// close this when finished.
	// Returns error when extraction fails or the video cannot be decoded.
	ExtractThumbnail(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error)
}

// StreamingTranscoderPort extends TranscoderPort with chunked streaming support.
// This enables adaptive bitrate streaming formats like HLS and DASH.
type StreamingTranscoderPort interface {
	TranscoderPort

	// TranscodeHLS generates HLS (HTTP Live Streaming) output with manifest and
	// segments. The implementation creates a master playlist (m3u8), variant
	// playlists for each bitrate, and video segments (.ts files).
	//
	// Takes input (io.Reader) which provides the source video data.
	// Takes spec (video_dto.HLSSpec) which defines the HLS encoding parameters.
	//
	// Returns video_dto.HLSResult which contains all components as streaming
	// readers.
	// Returns error when transcoding fails.
	TranscodeHLS(ctx context.Context, input io.Reader, spec video_dto.HLSSpec) (video_dto.HLSResult, error)
}

// Service provides the main entry point for video operations.
// It handles transcoding, thumbnail extraction, and HLS streaming.
type Service interface {
	// Transcode transcodes a video according to the provided parameters.
	// The parameters are parsed into a TranscodeSpec and validated against
	// service limits.
	//
	// Takes input (io.Reader) which provides the video data to transcode.
	// Takes params (map[string]string) which specifies the transcoding options.
	//
	// Returns io.ReadCloser which streams the transcoded output.
	// Returns error when the parameters are invalid or transcoding fails.
	Transcode(ctx context.Context, input io.Reader, params map[string]string) (io.ReadCloser, error)

	// ExtractCapabilities examines an input video and returns its metadata.
	//
	// Takes input (io.Reader) which provides the video data to examine.
	//
	// Returns video_dto.VideoCapabilities which contains the video metadata.
	// Returns error when the video cannot be read or examined.
	ExtractCapabilities(ctx context.Context, input io.Reader) (video_dto.VideoCapabilities, error)

	// TranscodeHLS creates HLS manifest and segment files for adaptive streaming.
	//
	// Takes input (io.Reader) which provides the source video data.
	// Takes spec (video_dto.HLSSpec) which defines the output format settings.
	//
	// Returns video_dto.HLSResult which contains the manifest and segment data.
	// Returns error when transcoding fails.
	TranscodeHLS(ctx context.Context, input io.Reader, spec video_dto.HLSSpec) (video_dto.HLSResult, error)

	// ExtractThumbnail extracts a frame from a video and returns it as an image.
	//
	// Takes input (io.Reader) which provides the source video data.
	// Takes spec (video_dto.ThumbnailSpec) which defines the extraction parameters.
	//
	// Returns io.ReadCloser which streams the encoded image.
	// Returns error when extraction fails.
	ExtractThumbnail(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error)
}
