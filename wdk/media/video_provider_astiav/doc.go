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

// Package video_provider_astiav implements video transcoding using FFmpeg
// via the go-astiav bindings.
//
// This provider offers codec-aware video transcoding, thumbnail
// extraction, and video metadata analysis. It requires FFmpeg
// libraries to be installed on the system and is only built when
// the "ffmpeg" build tag is set.
//
// # Usage
//
//	output, err := provider.Transcode(ctx, inputReader, media.TranscodeSpec{
//	    Codec:  "h264",
//	    Width:  1920,
//	    Height: 1080,
//	    Preset: "medium",
//	})
//
//	thumb, err := provider.ExtractThumbnail(ctx, inputReader,
//	    media.ThumbnailSpec{
//	        Timestamp: 5 * time.Second,
//	        Width:     320,
//	        Format:    "jpeg",
//	    },
//	)
//
// # Thread safety
//
// [Provider] methods are safe for concurrent use. A semaphore limits
// the number of concurrent transcode and thumbnail operations to the
// value set by [Config].MaxConcurrentTranscodes (default 10).
package video_provider_astiav
