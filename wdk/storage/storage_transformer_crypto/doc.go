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

// Package storage_transformer_crypto provides a storage stream
// transformer that encrypts and decrypts data using the centralised
// crypto service.
//
// This transformer applies chunked AES-256-GCM encryption with
// envelope encryption support for cloud KMS providers. It operates
// with constant memory usage (~64KB) regardless of file size, so
// it handles multi-GB files efficiently.
//
// # Usage
//
// Create a crypto transformer and register it with the storage
// service:
//
//	transformer := storage_transformer_crypto.New(
//		cryptoService, "crypto-service", 250,
//	)
//	service := storage.NewService(
//		storage.WithStreamTransformer(transformer),
//	)
//
// The transformer automatically encrypts data during uploads and
// decrypts during downloads. Set the priority to 250 (default) to
// ensure encryption runs after compression transformers (priority
// 100).
//
// # Thread safety
//
// The transformer is safe for concurrent use. Each call to
// Transform or Reverse creates independent streaming pipelines.
package storage_transformer_crypto
