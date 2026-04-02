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

// Package video_domain owns the core video processing abstractions and
// business logic.
//
// It defines the [TranscoderPort] and [StreamingTranscoderPort] port
// interfaces, a service layer that orchestrates transcoding with
// validation and metrics, and domain types for video metadata and
// encoding profiles.
//
// # Usage
//
// Create a service with registered transcoders:
//
//	transcoders := map[string]video_domain.TranscoderPort{
//		"ffmpeg": ffmpegAdapter,
//	}
//	service, err := video_domain.NewService(transcoders, "ffmpeg",
//		video_domain.DefaultServiceConfig())
//
// Then use the service for transcoding:
//
//	output, err := service.Transcode(ctx, videoReader, params)
//
// # Thread safety
//
// The service is safe for concurrent use. Each transcoding operation runs
// independently with its own context.
package video_domain
