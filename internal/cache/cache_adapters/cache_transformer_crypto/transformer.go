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
	"context"
	"fmt"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/crypto/crypto_domain"
)

const (
	// defaultTransformerName is the name used when no custom name is given.
	defaultTransformerName = "crypto-service"

	// defaultPriority is the default execution priority for encryption
	// transformers. Set to 250 so encryption runs after compression, which uses
	// priority 100.
	defaultPriority = 250
)

// CryptoCacheTransformer implements CacheTransformerPort to encrypt and decrypt
// cache data using the central crypto service.
type CryptoCacheTransformer struct {
	// cryptoService provides encryption and decryption for cache data.
	cryptoService crypto_domain.CryptoServicePort

	// name is the transformer's identifier.
	name string

	// priority is the execution order; lower values run first.
	priority int
}

var _ cache_domain.CacheTransformerPort = (*CryptoCacheTransformer)(nil)

// Config holds configuration for the crypto-service cache transformer.
type Config struct {
	// CryptoService is the crypto service instance to use.
	// If nil, the global crypto service from bootstrap is used.
	CryptoService crypto_domain.CryptoServicePort

	// Name is the unique identifier for this transformer instance.
	// Default: "crypto-service".
	Name string

	// Priority determines execution order where lower values run first on Set.
	// Defaults to 250; recommended 250 for encryption transformers (after
	// compression at 100).
	Priority int
}

// Name returns the transformer's name.
//
// Returns string which is the identifier for this transformer.
func (t *CryptoCacheTransformer) Name() string {
	return t.name
}

// Type returns the transformer type for encryption.
//
// Returns cache_dto.TransformerType which identifies this as an encryption
// transformer.
func (*CryptoCacheTransformer) Type() cache_dto.TransformerType {
	return cache_dto.TransformerEncryption
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's execution order value.
func (t *CryptoCacheTransformer) Priority() int {
	return t.priority
}

// Transform encrypts the input data using the crypto service.
// This is called when setting values in the cache.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes input ([]byte) which contains the plaintext data to
// encrypt.
//
// Returns []byte which contains the encrypted ciphertext.
// Returns error when encryption fails.
func (t *CryptoCacheTransformer) Transform(ctx context.Context, input []byte, _ any) ([]byte, error) {
	if len(input) == 0 {
		return input, nil
	}

	ciphertext, err := t.cryptoService.Encrypt(ctx, string(input))
	if err != nil {
		return nil, fmt.Errorf("crypto service encryption failed for cache: %w", err)
	}

	return []byte(ciphertext), nil
}

// Reverse decrypts the input data using the crypto service.
// This is called when getting values from the cache.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes input ([]byte) which contains the encrypted ciphertext
// to decrypt.
//
// Returns []byte which contains the decrypted plaintext.
// Returns error when decryption fails.
func (t *CryptoCacheTransformer) Reverse(ctx context.Context, input []byte, _ any) ([]byte, error) {
	if len(input) == 0 {
		return input, nil
	}

	plaintext, err := t.cryptoService.Decrypt(ctx, string(input))
	if err != nil {
		return nil, fmt.Errorf("crypto service decryption failed for cache: %w", err)
	}

	return []byte(plaintext), nil
}

// New creates a new crypto-service cache transformer.
//
// If name is empty, defaults to "crypto-service". If priority is 0, defaults
// to 250.
//
// Takes cryptoService (CryptoServicePort) which provides encryption and decryption
// operations for cached data.
// Takes name (string) which identifies this transformer.
// Takes priority (int) which determines the order of transformer execution.
//
// Returns *CryptoCacheTransformer which is ready for use in a cache pipeline.
func New(cryptoService crypto_domain.CryptoServicePort, name string, priority int) *CryptoCacheTransformer {
	if name == "" {
		name = defaultTransformerName
	}
	if priority == 0 {
		priority = defaultPriority
	}

	return &CryptoCacheTransformer{
		name:          name,
		priority:      priority,
		cryptoService: cryptoService,
	}
}

// createTransformerFromConfig creates a crypto transformer from a config value.
//
// Takes config (any) which provides the transformer configuration settings.
//
// Returns cache_domain.CacheTransformerPort which is the configured crypto
// transformer.
// Returns error when the crypto service cannot be obtained.
func createTransformerFromConfig(config any) (cache_domain.CacheTransformerPort, error) {
	cryptoService, name, priority := parseConfigValues(config)

	if cryptoService == nil {
		var err error
		cryptoService, err = bootstrap.GetCryptoService()
		if err != nil {
			return nil, fmt.Errorf("crypto transformer: %w", err)
		}
	}

	return New(cryptoService, name, priority), nil
}

// parseConfigValues extracts crypto service, name, and priority from config.
//
// Takes config (any) which is the configuration to parse, either a Config
// struct or map[string]any.
//
// Returns crypto_domain.CryptoServicePort which is the extracted crypto
// service, or nil if config is nil or invalid.
// Returns string which is the extracted name, or empty if config is invalid.
// Returns int which is the extracted priority, or zero if config is invalid.
func parseConfigValues(config any) (crypto_domain.CryptoServicePort, string, int) {
	if config == nil {
		return nil, "", 0
	}

	switch c := config.(type) {
	case Config:
		return c.CryptoService, c.Name, c.Priority
	case map[string]any:
		return parseMapConfig(c)
	default:
		return nil, "", 0
	}
}

// parseMapConfig extracts config values from a map.
//
// Takes c (map[string]any) which contains the configuration key-value pairs.
//
// Returns crypto_domain.CryptoServicePort which is the crypto service if found.
// Returns string which is the name value from the config.
// Returns int which is the priority value from the config.
func parseMapConfig(c map[string]any) (crypto_domain.CryptoServicePort, string, int) {
	var cryptoService crypto_domain.CryptoServicePort
	var name string
	var priority int

	if service, ok := c["cryptoService"].(crypto_domain.CryptoServicePort); ok {
		cryptoService = service
	}
	if n, ok := c["name"].(string); ok {
		name = n
	}
	if p, ok := c["priority"].(int); ok {
		priority = p
	}

	return cryptoService, name, priority
}

func init() {
	cache_domain.RegisterTransformerBlueprint(defaultTransformerName, createTransformerFromConfig)
}
