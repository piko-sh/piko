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

import "errors"

var (
	// ErrUnsupportedCodec is returned when the requested output codec is not
	// supported by the transcoder implementation.
	ErrUnsupportedCodec = errors.New("unsupported video codec")

	// ErrUnsupportedFormat is returned when the input or output format is not
	// supported.
	ErrUnsupportedFormat = errors.New("unsupported video format")

	// ErrInvalidResolution is returned when the requested output resolution is
	// invalid, such as zero or negative dimensions, or exceeds security
	// constraints.
	ErrInvalidResolution = errors.New("invalid video resolution")

	// ErrInvalidBitrate is returned when the requested bitrate is invalid or
	// exceeds limits.
	ErrInvalidBitrate = errors.New("invalid video bitrate")

	// ErrInvalidFramerate is returned when the video framerate is not valid.
	ErrInvalidFramerate = errors.New("invalid video framerate")

	// ErrDurationExceedsLimit is returned when the input video duration exceeds
	// the configured maximum duration security constraint.
	ErrDurationExceedsLimit = errors.New("video duration exceeds maximum limit")

	// ErrFileSizeExceedsLimit is returned when the input file size exceeds
	// the configured maximum size security constraint.
	ErrFileSizeExceedsLimit = errors.New("video file size exceeds maximum limit")

	// ErrResolutionExceedsLimit is returned when the requested output resolution
	// exceeds the configured maximum (e.g., 4K limit).
	ErrResolutionExceedsLimit = errors.New("video resolution exceeds maximum limit")

	// ErrTranscodingFailed is returned when the transcoding operation fails due to
	// an error in the underlying transcoder (e.g., FFmpeg encoding error).
	ErrTranscodingFailed = errors.New("video transcoding failed")

	// ErrInvalidStream is returned when the input video has no valid video stream,
	// or the stream is corrupted.
	ErrInvalidStream = errors.New("invalid or missing video stream")

	// ErrContextCancelled is returned when the transcoding operation is cancelled
	// via context cancellation.
	ErrContextCancelled = errors.New("video transcoding cancelled")

	// ErrTimeout is returned when a transcoding operation exceeds the configured
	// timeout.
	ErrTimeout = errors.New("video transcoding timeout exceeded")

	// ErrResourceExhausted is returned when system resources (memory, CPU,
	// concurrent limit) are exhausted and the transcoding request cannot be
	// accepted.
	ErrResourceExhausted = errors.New("system resources exhausted")

	// ErrInvalidHLSSpec is returned when the HLS specification contains invalid
	// parameters.
	ErrInvalidHLSSpec = errors.New("invalid HLS specification")

	// ErrSegmentationFailed is returned when video segmentation for HLS/DASH
	// fails.
	ErrSegmentationFailed = errors.New("video segmentation failed")

	// ErrHLSNotSupported is returned when the default provider does not
	// support HLS transcoding.
	ErrHLSNotSupported = errors.New("default provider does not support HLS transcoding")

	errNoTranscoders = errors.New("at least one video transcoder must be provided")

	errDefaultProviderEmpty = errors.New("default video provider cannot be empty")
)
