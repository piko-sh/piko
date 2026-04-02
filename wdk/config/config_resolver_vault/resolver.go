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

package config_resolver_vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"piko.sh/piko/wdk/json"
	"piko.sh/piko/wdk/config"
)

var _ config.Resolver = (*Resolver)(nil)

// Resolver fetches secrets from HashiCorp Vault.
// It implements the config.Resolver interface.
//
// Circuit breaker protection is provided by the config Loader layer,
// not by this resolver directly.
//
// AUTHENTICATION:
// The resolver uses the official Vault Go client, which authenticates
// automatically via standard environment variables (VAULT_ADDR, VAULT_TOKEN,
// VAULT_NAMESPACE, etc.). It is compatible with various auth methods
// configured via these variables.
//
// USAGE FORMAT:
// The value is expected in the format: "vault:mount/path/to/secret#key"
//  1. Plain secret: "vault:secret/data/my-app/production" -> returns the
//     entire JSON secret object.
//  2. JSON key: "vault:secret/data/my-app/production#db_password" -> fetches
//     "db_password" from the secret.
//
// NOTE: This implementation is designed for Vault's KVv2 secrets engine,
// which is the modern standard. The path should include the mount point
// (e.g., "secret").
type Resolver struct {
	// client is the Vault API client used to fetch secrets.
	client *api.Client
}

// NewResolver creates a new HashiCorp Vault resolver.
// It uses the default Vault client settings, which rely on environment
// variables (VAULT_ADDR, VAULT_TOKEN, etc.) for authentication.
//
// Returns *Resolver which is the configured resolver ready for use.
// Returns error when the Vault client cannot be created.
func NewResolver() (*Resolver, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	return &Resolver{
		client: client,
	}, nil
}

// GetPrefix returns the prefix this resolver handles.
//
// Returns string which is the prefix "vault:" that identifies secrets to be
// resolved by this resolver.
func (*Resolver) GetPrefix() string {
	return "vault:"
}

// Resolve fetches the secret value from HashiCorp Vault.
//
// Takes value (string) which specifies the secret path, optionally followed by
// a "#" and a JSON key to extract a specific field.
//
// Returns string which is the secret value, or the entire secret as JSON if no
// key is specified.
// Returns error when the path format is invalid, the secret is not found, or
// the Vault API call fails.
func (r *Resolver) Resolve(ctx context.Context, value string) (string, error) {
	secretPath, jsonKey, _ := strings.Cut(value, "#")
	if secretPath == "" {
		return "", fmt.Errorf("invalid Vault secret format: %q; path must not be empty", value)
	}

	parts := strings.SplitN(secretPath, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid Vault path %q: must include mount point (e.g., 'secret/data/...')", secretPath)
	}
	mountPath := parts[0]
	pathInMount := parts[1]

	secret, err := r.client.KVv2(mountPath).Get(ctx, pathInMount)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %q from Vault: %w", secretPath, err)
	}
	if secret == nil {
		return "", fmt.Errorf("secret %q not found in Vault", secretPath)
	}

	secretData, ok := secret.Data["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected data format for KVv2 secret %q", secretPath)
	}

	var secretValue string
	if jsonKey != "" {
		value, exists := secretData[jsonKey]
		if !exists {
			return "", fmt.Errorf("key %q not found in Vault secret %q", jsonKey, secretPath)
		}
		secretValue = fmt.Sprintf("%v", value)
	} else {
		jsonBytes, err := json.Marshal(secretData)
		if err != nil {
			return "", fmt.Errorf("failed to marshal Vault secret data for %q: %w", secretPath, err)
		}
		secretValue = string(jsonBytes)
	}

	return secretValue, nil
}

// Register creates a new Vault resolver and registers it in the global
// resolver registry. This is a convenience function equivalent to
// NewResolver() followed by config.RegisterResolver().
//
// Returns error when resolver creation or registration fails.
//
// Example:
//
//	func init() {
//	    if err := config_resolver_vault.Register(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func Register() error {
	resolver, err := NewResolver()
	if err != nil {
		return fmt.Errorf("creating vault resolver: %w", err)
	}
	return config.RegisterResolver(resolver)
}
