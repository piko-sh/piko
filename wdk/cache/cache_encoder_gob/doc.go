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

// Package cache_encoder_gob provides Gob-based encoding for cache values.
//
// Gob is Go's native binary encoding format. It outperforms JSON for complex Go
// types, supports unexported fields (when registered with
// [encoding/gob.Register]), but is not human-readable and only works between Go
// processes.
//
// Implements [cache.EncoderPort] and works with any cache provider that
// supports encoding registries.
package cache_encoder_gob
