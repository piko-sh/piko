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

package storage_transformer_crypto

import (
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/storage/storage_adapters/transformer_crypto"
	"piko.sh/piko/wdk/storage"
)

// Config holds settings for the crypto service encryption transformer.
type Config = transformer_crypto.Config

// New creates a crypto service encryption transformer.
//
// The transformer supports centralised key management and key rotation,
// chunked AES-256-GCM encryption (64KB chunks), automatic envelope
// encryption for cloud KMS providers, and constant memory usage regardless
// of file size.
//
// Takes cryptoService (crypto_domain.CryptoServicePort) which provides the
// crypto service instance to use for encryption and decryption.
// Takes name (string) which identifies this transformer instance.
// Takes priority (int) which sets the execution order priority.
//
// Returns storage.StreamTransformerPort which can encrypt and decrypt data
// streams using the centralised crypto service.
func New(cryptoService crypto_domain.CryptoServicePort, name string, priority int) storage.StreamTransformerPort {
	return transformer_crypto.New(cryptoService, name, priority)
}
