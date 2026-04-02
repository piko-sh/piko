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

// Package cache_transformer_crypto provides encryption transformation
// for cache values using the centralised crypto service.
//
// It delegates to the application's crypto service, so key management
// and rotation are handled centrally rather than per-cache. The
// encryption algorithm is determined by the crypto service provider
// (typically AES-256-GCM).
//
// When combining with compression, compress first (lower priority
// number) so encryption operates on the smaller payload.
//
// All methods are safe for concurrent use, provided the underlying
// crypto service is also safe for concurrent use.
package cache_transformer_crypto
