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

// Package crypto_provider_local_aes_gcm provides a local AES-256-GCM
// encryption provider.
//
// This provider performs encryption and decryption entirely on the
// local machine without any external service dependencies. It is
// suitable for development, testing, single-server deployments, and
// scenarios where network latency to a KMS is unacceptable.
//
// It uses AES-256-GCM with a random 96-bit IV per encryption and a
// 128-bit authentication tag for tamper detection. The encryption
// key is held in memory, so it should be loaded from environment
// variables or a secrets manager rather than committed to version
// control.
//
// # Thread safety
//
// The provider returned by [NewProvider] is safe for concurrent use.
package crypto_provider_local_aes_gcm
