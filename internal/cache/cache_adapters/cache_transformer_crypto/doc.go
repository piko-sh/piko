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

// Package cache_transformer_crypto implements the cache transformer port
// using the centralised crypto service for encryption and decryption.
//
// This adapter delegates all cryptographic operations to the application's
// central crypto service, rather than managing keys directly. It
// centralises key management and supports features such as key rotation.
// The transformer self-registers under the blueprint name "crypto-service"
// so the cache builder can instantiate it from configuration.
//
// It is designed to run after compression in the cache pipeline. Its
// default priority of 250 places it after compression transformers
// (priority 100) on writes, and before them on reads.
//
// # Thread safety
//
// All methods are safe for concurrent use, provided the underlying
// crypto service implementation is also safe for concurrent use.
package cache_transformer_crypto
