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

// Package transcoder_mock is an in-memory test double for
// [video_domain.TranscoderPort] and [video_domain.StreamingTranscoderPort].
//
// It records all method calls for inspection and supports configurable
// return values, error injection, and latency simulation, allowing
// tests to verify transcoding behaviour without FFmpeg.
//
// # Usage
//
//	mock := transcoder_mock.NewProvider()
//	mock.SetError(errors.New("transcode failed"))
//
//	_, err := mock.Transcode(ctx, input, spec)
//	// err contains "transcode failed"
//
//	calls := mock.GetTranscodeCalls()
//	// inspect calls[0].Spec, calls[0].InputData, etc.
//
//	mock.Reset() // clear all recorded state
//
// # Thread safety
//
// All methods on Provider are safe for concurrent use. Internal state
// is protected by a sync.RWMutex, and getter methods return copies of
// recorded call slices to prevent external mutation.
package transcoder_mock
