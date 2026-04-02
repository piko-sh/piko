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

// Package json wraps JSON encoding and decoding behind a pluggable
// [Provider] interface. By default all operations delegate to
// [encoding/json]; a higher-performance provider (such as sonic) can
// replace the defaults at runtime via [Activate].
//
// # Provider activation
//
// Providers are activated during application bootstrap via
// [piko.WithJSONProvider]. All function variables ([Marshal], [Unmarshal],
// etc.) and API variables ([ConfigStd], [ConfigDefault]) are swapped
// atomically. Frozen configurations created via [Freeze] resolve lazily on
// first use, so they automatically pick up whichever provider is active at
// call time.
package json
