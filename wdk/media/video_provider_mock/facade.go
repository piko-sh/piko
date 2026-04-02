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

package video_provider_mock

import (
	"piko.sh/piko/internal/video/video_adapters/transcoder_mock"
)

// Provider is a thread-safe, in-memory mock implementation of
// VideoTranscoderPort and StreamingTranscoderPort for testing. It supports
// call inspection and simulation of transcodes and errors.
type Provider = transcoder_mock.Provider

// TranscodeCall records the parameters passed to a single call of the
// Transcode method. It allows tests to inspect what the service is
// asking the transcoder to do.
type TranscodeCall = transcoder_mock.TranscodeCall

// ExtractCapabilitiesCall records calls to ExtractCapabilities.
type ExtractCapabilitiesCall = transcoder_mock.ExtractCapabilitiesCall

// TranscodeHLSCall records calls made to the TranscodeHLS method.
type TranscodeHLSCall = transcoder_mock.TranscodeHLSCall

// ExtractThumbnailCall is a type alias that records calls to ExtractThumbnail.
type ExtractThumbnailCall = transcoder_mock.ExtractThumbnailCall

// NewProvider creates a new, initialised mock video transcoder.
var NewProvider = transcoder_mock.NewProvider
