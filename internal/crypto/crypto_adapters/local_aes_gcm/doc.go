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

// Package local_aes_gcm implements AES-256-GCM encryption without external
// dependencies.
//
// This adapter provides authenticated encryption using the standard Go
// crypto library. It supports both single-shot operations for small data
// and streaming encryption for large files with constant memory usage.
// It is suitable for development, testing, and single-server deployments
// where external KMS integration is not required.
//
// Single-shot encryption produces base64-encoded output containing a
// 12-byte IV, the ciphertext, and a 16-byte auth tag. Streaming
// encryption uses a chunked format:
//
//	[1-byte version][4-byte header length][JSON header][encrypted chunks...]
//
// Each chunk is encrypted with a unique IV derived from the base IV
// and chunk number.
//
// The streaming primitives are exported so that other encryption
// providers can reuse the same envelope format for consistency.
//
// All Provider methods are safe for concurrent use.
package local_aes_gcm
