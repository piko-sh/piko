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

// Package video_dto defines data transfer objects for the video module.
//
// It contains specification types for transcoding, thumbnail extraction,
// and adaptive streaming (HLS and DASH) workflows, plus result types
// that carry output data between the domain layer and adapters.
//
// # Parsing
//
// The package provides helpers for constructing specs from raw
// parameter maps and time strings:
//
//   - [ParseTranscodeSpec]: Builds a TranscodeSpec from a
//     map[string]string of capability parameters
//   - [ParseThumbnailTime]: Parses time strings in Go duration,
//     MM:SS, or HH:MM:SS format
//
// # Usage
//
//	spec := video_dto.TranscodeSpec{
//		Codec:   "h264",
//		Width:   1920,
//		Height:  1080,
//		Bitrate: 5_000_000,
//	}
//	if err := spec.Validate(); err != nil {
//		return err
//	}
package video_dto
