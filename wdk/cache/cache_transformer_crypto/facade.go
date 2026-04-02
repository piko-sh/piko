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

package cache_transformer_crypto

import (
	"piko.sh/piko/internal/cache/cache_adapters/cache_transformer_crypto"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/wdk/cache"
)

// Config holds settings for the crypto-service cache transformer.
// This is re-exported from the internal adapter package.
type Config = cache_transformer_crypto.Config

// New creates a new crypto-service cache encryption transformer.
//
// This transformer delegates all cryptographic operations to the provided
// crypto service, which centralises key management and supports key rotation.
//
// If name is empty, defaults to "crypto-service". If priority is 0, defaults
// to 250 (runs after compression at priority 100).
//
// Takes cryptoService (crypto_domain.CryptoServicePort) which provides encryption
// and decryption operations for cached data.
// Takes name (string) which identifies this transformer instance.
// Takes priority (int) which determines the order of transformer execution.
//
// Returns cache.TransformerPort which is the configured encryption
// transformer ready for use.
//
// Example:
//
//	cryptoService, _ := bootstrap.GetCryptoService()
//	transformer := cache_transformer_crypto.New(cryptoService, "", 0)
func New(cryptoService crypto_domain.CryptoServicePort, name string, priority int) cache.TransformerPort {
	return cache_transformer_crypto.New(cryptoService, name, priority)
}
